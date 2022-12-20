package utils

import (
	"net"
	"net/http"
	"strconv"
	"strings"
)

func ParseAddr(addr string) (string, int) {
	parts := strings.Split(addr, ":")
	if len(parts) != 2 {
		return addr, 0
	}

	if port, err := strconv.ParseInt(parts[1], 10, 64); err != nil {
		return addr, 0
	} else {
		return parts[0], int(port)
	}
}

func GetIPAddressFromHeader(r http.Header) (ip string) {
	if r.Get("CF-Connecting-IP") != "" {
		return r.Get("CF-Connecting-IP")
	}

	if r.Get("X-Original-Forwarded-For") != "" {
		return r.Get("X-Original-Forwarded-For")
	}

	// If ip not provided, get real ip from Header (behind nginx)
	ip = r.Get("X-Real-IP")
	return
}

func GetIpAddressFromRequest(r *http.Request) (ip string) {
	ip = GetIPAddressFromHeader(r.Header)

	// get from remote address
	if len(ip) == 0 {
		ip, _, _ = net.SplitHostPort(r.RemoteAddr)
	}
	return
}