package transhttp

import (
	"math/rand"
	"net/http"
	"strings"
)

func IsWebSocket(r *http.Request) bool {
	contains := func(key, val string) bool {
		vv := strings.Split(r.Header.Get(key), ",")
		for _, v := range vv {
			if val == strings.ToLower(strings.TrimSpace(v)) {
				return true
			}
		}
		return false
	}

	if contains("Connection", "upgrade") && contains("Upgrade", "websocket") {
		return true
	}

	return false
}

func GenerateRandomETag() string {
	strLen := 20

	bytes := make([]byte, strLen)
	for i := 0; i < strLen; i++ {
		bytes[i] = byte(65 + rand.Intn(25))
	}
	return string(bytes)
}
