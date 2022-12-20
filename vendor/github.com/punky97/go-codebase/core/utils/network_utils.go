package utils

import (
	"github.com/punky97/go-codebase/core/logger"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
)

var loopbacks = []string{"::1", "127.0"}

const (
	loopbackIP = "127.0.0.1"
)

// MyIP -- get current IP
func MyIP(allowLoopback bool) string {
	if allowLoopback {
		return loopbackIP
	}
	addrs := getIPByHostname()
	shouldGetExternal := false
	for _, add := range addrs {
		for _, lb := range loopbacks {
			if strings.Contains(add, lb) {
				shouldGetExternal = true
				goto External
			}
		}
	}
External:
	ip := ""
	if shouldGetExternal {
		var err error
		ip, err = getLocalAddr()
		if err != nil {
			logger.BkLog.Info("Get from external")
			ip, err = externalIP()
			if err != nil {
				logger.BkLog.Fatalf("Cannot get ip address from external and hostname")
			}
		}
	} else if len(addrs) > 0 {
		ip = addrs[0]
	} else {
		logger.BkLog.Fatalf("Cannot get ip")
	}
	logger.BkLog.Info("Got ip from ", convertSource(shouldGetExternal), ": ", ip)
	return ip
}

func convertSource(i bool) string {
	if i {
		return "external method"
	}
	return "hostname  method"
}

func externalIP() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return "", err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			return ip.String(), nil
		}
	}
	return "", errors.New("are you connected to the network?")
}

func getLocalAddr() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()
	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP.String(), nil
}

func getIPByHostname() []string {
	addrs := make([]string, 0)
	hostname, err := os.Hostname()
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cannot get hostname: %v", err))
		return addrs
	}
	addrs, err = net.LookupHost(hostname)
	if err != nil {
		logger.BkLog.Errorw(fmt.Sprintf("Cannot lookup hostname: %v", err))
		return addrs
	}
	logger.BkLog.Info("addrs ", " ", hostname, " ", strings.Join(addrs, ","))
	if len(addrs) < 1 {
		logger.BkLog.Errorw(fmt.Sprintf("cannot get ip address from hostname"))
	}
	return addrs
}
