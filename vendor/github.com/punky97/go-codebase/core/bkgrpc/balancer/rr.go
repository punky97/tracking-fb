/*
 *
 * Copyright 2016 gRPC authors.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package balancer

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/net/context"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/grpclog"
	"google.golang.org/grpc/naming"
	"google.golang.org/grpc/status"
)

var (
	errBalancerClosed = errors.New("grpc: balancer is closed")
)

// downErr implements net.Error. It is constructed by gRPC internals and passed to the down
// call of Balancer.
type downErr struct {
	timeout   bool
	temporary bool
	desc      string
}

func (e downErr) Error() string   { return e.desc }
func (e downErr) Timeout() bool   { return e.timeout }
func (e downErr) Temporary() bool { return e.temporary }

func downErrorf(timeout, temporary bool, format string, a ...interface{}) downErr {
	return downErr{
		timeout:   timeout,
		temporary: temporary,
		desc:      fmt.Sprintf(format, a...),
	}
}

// RoundRobin returns a Balancer that selects addresses round-robin. It uses r to watch
// the name resolution updates and updates the addresses available correspondingly.
func RoundRobin(r naming.Resolver, localPreferred bool) grpc.Balancer {
	return &roundRobin{r: r, localPreferred: localPreferred}
}

type addrInfo struct {
	addr      grpc.Address
	connected bool
}

type roundRobin struct {
	r              naming.Resolver
	w              naming.Watcher
	addrs          []*addrInfo // all the addresses the client should potentially connect
	mu             sync.Mutex
	addrCh         chan []grpc.Address // the channel to notify gRPC internals the list of addresses the client should connect to.
	next           int                 // index of the next address to return for Get()
	waitCh         chan struct{}       // the channel to block when there is no connected address available
	done           bool                // The Balancer is closed.
	localPreferred bool
}

func (rr *roundRobin) watchAddrUpdates() error {
	updates, err := rr.w.Next()
	if err != nil {
		grpclog.Warningf("grpc: the naming watcher stops working due to %v.", err)
		return err
	}
	rr.mu.Lock()
	defer rr.mu.Unlock()
	for _, update := range updates {
		addr := grpc.Address{
			Addr:     update.Addr,
			Metadata: update.Metadata,
		}
		switch update.Op {
		case naming.Add:
			var exist bool
			for _, v := range rr.addrs {
				if addr == v.addr {
					exist = true
					grpclog.Infoln("grpc: The name resolver wanted to add an existing address: ", addr)
					break
				}
			}
			if exist {
				continue
			}
			rr.addrs = append(rr.addrs, &addrInfo{addr: addr})
		case naming.Delete:
			for i, v := range rr.addrs {
				if addr == v.addr {
					copy(rr.addrs[i:], rr.addrs[i+1:])
					rr.addrs = rr.addrs[:len(rr.addrs)-1]
					break
				}
			}
		default:
			grpclog.Errorln("Unknown update.Op ", update.Op)
		}
	}
	// Make a copy of rr.addrs and write it onto rr.addrCh so that gRPC internals gets notified.
	open := make([]grpc.Address, len(rr.addrs))
	for i, v := range rr.addrs {
		open[i] = v.addr
	}
	if rr.done {
		return grpc.ErrClientConnClosing
	}
	select {
	case <-rr.addrCh:
	default:
	}
	rr.addrCh <- open
	return nil
}

func (rr *roundRobin) Start(target string, config grpc.BalancerConfig) error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.done {
		return grpc.ErrClientConnClosing
	}
	if rr.r == nil {
		// If there is no name resolver installed, it is not needed to
		// do name resolution. In this case, target is added into rr.addrs
		// as the only address available and rr.addrCh stays nil.
		rr.addrs = append(rr.addrs, &addrInfo{addr: grpc.Address{Addr: target}})
		return nil
	}
	w, err := rr.r.Resolve(target)
	if err != nil {
		return err
	}
	rr.w = w
	rr.addrCh = make(chan []grpc.Address, 1)
	go func() {
		for {
			if err := rr.watchAddrUpdates(); err != nil {
				return
			}
		}
	}()
	return nil
}

// Up sets the connected state of addr and sends notification if there are pending
// Get() calls.
func (rr *roundRobin) Up(addr grpc.Address) func(error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	var cnt int
	for _, a := range rr.addrs {
		if a.addr == addr {
			if a.connected {
				return nil
			}
			a.connected = true
		}
		if a.connected {
			cnt++
		}
	}
	// addr is only one which is connected. Notify the Get() callers who are blocking.
	if cnt == 1 && rr.waitCh != nil {
		close(rr.waitCh)
		rr.waitCh = nil
	}
	return func(err error) {
		rr.down(addr, err)
	}
}

// down unsets the connected state of addr.
func (rr *roundRobin) down(addr grpc.Address, err error) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	for _, a := range rr.addrs {
		if addr == a.addr {
			a.connected = false
			break
		}
	}
}

var loopbacks = []string{"::1", "127.0."}

// Get returns the next addr in the rotation.
func (rr *roundRobin) Get(ctx context.Context, opts grpc.BalancerGetOptions) (addr grpc.Address, put func(), err error) {
	var ch chan struct{}
	rr.mu.Lock()
	if rr.done {
		rr.mu.Unlock()
		err = grpc.ErrClientConnClosing
		return
	}

	if len(rr.addrs) > 0 {
		rr.next = 0
		next := rr.next
		for {
			a := rr.addrs[next]
			next = (next + 1) % len(rr.addrs)
			if a.connected {
				isOk := true
				if rr.localPreferred {
					isOk = false
				local_loop:
					for _, lv := range loopbacks {
						if strings.Contains(a.addr.Addr, lv) {
							isOk = true
							break local_loop
						}
					}
				}

				if isOk {
					addr = a.addr
					rr.next = next
					rr.mu.Unlock()
					return
				}
			}
			if next == rr.next {
				// Has iterated all the possible address but none is connected.
				break
			}
		}
		// not found on local, get any available connection
		next = 0
		for {
			a := rr.addrs[next]
			next = (next + 1) % len(rr.addrs)
			if a.connected {
				addr = a.addr
				rr.next = next
				rr.mu.Unlock()
				return
			}
			if next == rr.next {
				// Has iterated all the possible address but none is connected.
				break
			}
		}
	}

	if !opts.BlockingWait {
		if len(rr.addrs) == 0 {
			rr.mu.Unlock()
			err = status.Errorf(codes.Unavailable, "there is no address available")
			return
		}
		// Returns the next addr on rr.addrs for failfast RPCs.
		addr = rr.addrs[rr.next].addr
		rr.next++
		rr.mu.Unlock()
		return
	}
	// Wait on rr.waitCh for non-failfast RPCs.
	if rr.waitCh == nil {
		ch = make(chan struct{})
		rr.waitCh = ch
	} else {
		ch = rr.waitCh
	}
	rr.mu.Unlock()
	for {
		select {
		case <-ctx.Done():
			err = ctx.Err()
			return
		case <-ch:
			rr.mu.Lock()
			if rr.done {
				rr.mu.Unlock()
				err = grpc.ErrClientConnClosing
				return
			}

			if len(rr.addrs) > 0 {
				if rr.next >= len(rr.addrs) {
					rr.next = 0
				}
				next := rr.next
				for {
					a := rr.addrs[next]
					next = (next + 1) % len(rr.addrs)
					if a.connected {
						addr = a.addr
						rr.next = next
						rr.mu.Unlock()
						return
					}
					if next == rr.next {
						// Has iterated all the possible address but none is connected.
						break
					}
				}
			}
			// The newly added addr got removed by Down() again.
			if rr.waitCh == nil {
				ch = make(chan struct{})
				rr.waitCh = ch
			} else {
				ch = rr.waitCh
			}
			rr.mu.Unlock()
		}
	}
}

func (rr *roundRobin) Notify() <-chan []grpc.Address {
	return rr.addrCh
}

func (rr *roundRobin) Close() error {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.done {
		return errBalancerClosed
	}
	rr.done = true
	if rr.w != nil {
		rr.w.Close()
	}
	if rr.waitCh != nil {
		close(rr.waitCh)
		rr.waitCh = nil
	}
	if rr.addrCh != nil {
		close(rr.addrCh)
	}
	return nil
}

// pickFirst is used to test multi-addresses in one addrConn in which all addresses share the same addrConn.
// It is a wrapper around roundRobin balancer. The logic of all methods works fine because balancer.Get()
// returns the only address Up by resetTransport().
type pickFirst struct {
	*roundRobin
}

func pickFirstBalancerV1(r naming.Resolver) grpc.Balancer {
	return &pickFirst{&roundRobin{r: r}}
}
