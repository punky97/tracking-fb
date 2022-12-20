package registry

import (
	"crypto/tls"
	"errors"
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/punky97/go-codebase/core/logger"
	"net"
	"net/http"
	"runtime"
	"sync"
	"time"

	hash "github.com/mitchellh/hashstructure"
)

const loopbackIp = "127.0.0.1"

type consulRegistry struct {
	Address      string
	Client       *consul.Client
	Options      Options
	queryOptions *consul.QueryOptions
	sync.Mutex
	register    map[string]uint64
	lastChecked map[string]time.Time
}

func newTransport(config *tls.Config) *http.Transport {
	if config == nil {
		config = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	t := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).Dial,
		TLSHandshakeTimeout: 10 * time.Second,
		TLSClientConfig:     config,
	}
	runtime.SetFinalizer(&t, func(tr **http.Transport) {
		(*tr).CloseIdleConnections()
	})
	return t
}

func newConsulRegistry(opts ...Option) Registry {
	var options Options
	for _, o := range opts {
		o(&options)
	}

	// use default config
	config := consul.DefaultConfig()
	if options.Context != nil {
		// Use the consul config passed in the options, if available
		if c, ok := options.Context.Value("consul_config").(*consul.Config); ok {
			config = c
		}
	}
	if config.HttpClient == nil {
		config.HttpClient = new(http.Client)
	}

	// set timeout
	if options.Timeout > 0 {
		config.HttpClient.Timeout = options.Timeout
	}

	// check if there are any addrs
	if len(options.Addrs) > 0 {
		addr, port, err := net.SplitHostPort(options.Addrs[0])
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "8500"
			addr = options.Addrs[0]
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		} else if err == nil {
			config.Address = fmt.Sprintf("%s:%s", addr, port)
		}
	}

	// requires secure connection?
	if options.Secure || options.TLSConfig != nil {
		config.Scheme = "https"
		// We're going to support InsecureSkipVerify
		config.HttpClient.Transport = newTransport(options.TLSConfig)
	}

	// create the client
	client, _ := consul.NewClient(config)

	cr := &consulRegistry{
		Address:     config.Address,
		Client:      client,
		Options:     options,
		register:    make(map[string]uint64),
		lastChecked: make(map[string]time.Time),
	}

	return cr
}

func (c *consulRegistry) Deregister(s *Service) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	// delete our hash of the service
	c.Lock()
	delete(c.register, s.Name)
	c.Unlock()

	node := s.Nodes[0]
	return c.Client.Agent().ServiceDeregister(node.Id)
}

func (c *consulRegistry) DeregisterMalformedService(s *Service) error {
	if s == nil || len(s.Nodes) == 0 {
		return errors.New("Invalid service definition")
	}
	c.Lock()
	// delete if needed
	delete(c.register, s.Name)
	defer c.Unlock()
	self := s.Nodes[0]

	services, err := c.Client.Agent().Services()
	if err != nil {
		return errors.New("Cannot query services from consul to deregister malformed service(s)")
	}

	// temporary disable this with loopback IP
	if self.Address == loopbackIp {
		return nil
	}

	logger.BkLog.Infof("Checking %v service(s)", len(services))
	for _, v := range services {
		// on staging & production, each pod is assigned for just one IP and it is unique
		// on devvm, all service use same IP but different ports
		if self.Address == v.Address && self.Port == v.Port {
			err = c.Client.Agent().ServiceDeregister(v.ID)
			if err != nil {
				logger.BkLog.Errorw(fmt.Sprintf("Cannot deregister node: %v", err), "id", v.ID, "service", v.Service, "addr", v.Address, "port", v.Port, "addr", self.Address)
			} else {
				logger.BkLog.Infof("Deregister node %v %v:%v, self: %v successfully",
					v.ID, v.Address, v.Port, self.Address)
			}
		}
	}
	return nil
}

func getDeregisterTTL(t time.Duration) time.Duration {
	// splay slightly for the watcher?
	splay := time.Second * 5
	deregTTL := t + splay

	// consul has a minimum timeout on deregistration of 1 minute.
	if t < time.Minute {
		deregTTL = time.Minute + splay
	}

	return deregTTL
}

func (c *consulRegistry) Register(s *Service, opts ...RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("Require at least one node")
	}

	var regTCPCheck bool
	var regGrpcCheck bool
	var regHTTPCheck bool
	var regInterval time.Duration

	var options RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	if s.Nodes[0].Address != loopbackIp && c.Options.Context != nil {
		// grpc health-check
		if grpcCheckInterval, ok := c.Options.Context.Value("consul_grpc_check_interval").(time.Duration); ok {
			regGrpcCheck = true
			regInterval = grpcCheckInterval
		} else if httpCheckInterval, ok := c.Options.Context.Value("consul_http_check_interval").(time.Duration); ok {
			regHTTPCheck = true
			regInterval = httpCheckInterval
		} else if tcpCheckInterval, ok := c.Options.Context.Value("consul_tcp_check_interval").(time.Duration); ok {
			regTCPCheck = true
			regInterval = tcpCheckInterval
		}
	}

	// create hash of service; uint64
	h, err := hash.Hash(s, nil)
	if err != nil {
		return err
	}

	// use first node
	node := s.Nodes[0]

	// get existing hash
	c.Lock()
	v, ok := c.register[s.Name]
	lastCheck := c.lastChecked[s.Name]
	c.Unlock()

	// if it's already registered and matches then just pass the check
	if ok && v == h {
		if options.TTL == time.Duration(0) {
			// ensure that our service hasn't been deregistered by Consul
			if time.Since(lastCheck) <= getDeregisterTTL(regInterval) {
				return nil
			}
			services, _, err := c.Client.Health().Checks(s.Name, c.queryOptions)
			if err == nil {
				for _, v := range services {
					if v.ServiceID == node.Id {
						return nil
					}
				}
			}
		} else {
			// if the err is nil we're all good, bail out
			// if not, we don't know what the state is, so full re-register
			if err := c.Client.Agent().PassTTL("service:"+node.Id, ""); err == nil {
				return nil
			}
		}
	}

	// encode the tags
	tags := encodeMetadata(node.Metadata)
	tags = append(tags, encodeServiceMetadata(s.Metadata)...)
	tags = append(tags, encodeEndpoints(s.Endpoints)...)
	tags = append(tags, encodeVersion(s.Version)...)

	var check *consul.AgentServiceCheck

	if regGrpcCheck {
		deregTTL := getDeregisterTTL(regInterval)
		check = &consul.AgentServiceCheck{
			GRPC:                           fmt.Sprintf("%s:%d/%v", node.Address, node.Port, s.IDName),
			Interval:                       fmt.Sprintf("%v", regInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	} else if regHTTPCheck {
		deregTTL := getDeregisterTTL(regInterval)
		check = &consul.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/service-health?svc=%v", node.Address, node.Port, s.IDName),
			Interval:                       fmt.Sprintf("%v", regInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	} else if regTCPCheck {
		deregTTL := getDeregisterTTL(regInterval)
		check = &consul.AgentServiceCheck{
			TCP:                            fmt.Sprintf("%s:%d", node.Address, node.Port),
			Interval:                       fmt.Sprintf("%v", regInterval),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
		// if the TTL is greater than 0 create an associated check
	} else if options.TTL > time.Duration(0) {
		deregTTL := getDeregisterTTL(options.TTL)

		check = &consul.AgentServiceCheck{
			TTL:                            fmt.Sprintf("%v", options.TTL),
			DeregisterCriticalServiceAfter: fmt.Sprintf("%v", deregTTL),
		}
	}

	consulReg := &consul.AgentServiceRegistration{
		ID:      node.Id,
		Name:    s.Name,
		Tags:    tags,
		Port:    node.Port,
		Address: node.Address,
		Check:   check,
	}

	// register the service
	if err := c.Client.Agent().ServiceRegister(consulReg); err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Error when register service: %v ", err))
		return err
	}

	// save our hash of the service
	c.Lock()
	c.register[s.Name] = h
	c.lastChecked[s.Name] = time.Now()
	c.Unlock()

	// if the TTL is 0 we don't mess with the checks
	if options.TTL == time.Duration(0) {
		return nil
	}

	// pass the healthcheck
	return c.Client.Agent().PassTTL("service:"+node.Id, "")
}

func (c *consulRegistry) GetService(name string) ([]*Service, error) {
	rsp, _, err := c.Client.Health().Service(name, "", false, nil)
	if err != nil {
		return nil, err
	}

	serviceMap := map[string]*Service{}

	for _, s := range rsp {
		if s.Service.Service != name {
			continue
		}

		// version is now a tag
		version, found := decodeVersion(s.Service.Tags)
		// service ID is now the node id
		id := s.Service.ID
		// key is always the version
		key := version
		// address is service address
		address := s.Service.Address

		if address == loopbackIp && c.Options.IgnoreLoopback {
			continue
		}

		// if we can't get the version we bail
		// use old the old ways
		if !found {
			continue
		}

		svc, ok := serviceMap[key]
		if !ok {
			svc = &Service{
				Endpoints: decodeEndpoints(s.Service.Tags),
				Metadata:  decodeServiceMetadata(s.Service.Tags),
				Name:      s.Service.Service,
				Version:   version,
			}
			serviceMap[key] = svc
		}

		var del bool
		for _, check := range s.Checks {
			// delete the node if the status is critical
			if check.Status == "critical" {
				del = true
				break
			}
		}

		// if delete then skip the node
		if del {
			continue
		}

		svc.Nodes = append(svc.Nodes, &Node{
			Id:       id,
			Address:  address,
			Port:     s.Service.Port,
			Metadata: decodeMetadata(s.Service.Tags),
		})
	}

	var services []*Service
	for _, service := range serviceMap {
		services = append(services, service)
	}
	return services, nil
}

func (c *consulRegistry) ListServices() ([]*Service, error) {
	rsp, _, err := c.Client.Catalog().Services(nil)
	if err != nil {
		return nil, err
	}

	var services []*Service

	for service := range rsp {
		services = append(services, &Service{Name: service})
	}

	return services, nil
}

func (c *consulRegistry) Watch() (Watcher, error) {
	return newConsulWatcher(c)
}

func (c *consulRegistry) String() string {
	return "consul"
}
