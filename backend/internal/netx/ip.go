package netx

import (
	"net"
	"net/http"
	"strings"
)

// ClientIP returns the client ip
func ClientIP(r *http.Request) string {
	ip := strings.TrimSpace(r.Header.Get("X-Real-IP"))
	if ip != "" {
		return ip
	}
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			ip = strings.TrimSpace(parts[0])
			if ip != "" {
				return ip
			}
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
