package middleware

import (
	"net"
	"net/http"
	"strings"

	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
)

// RealIP resolves a client IP once per request. Forwarded headers are trusted
// only when the direct peer matches a configured trusted proxy CIDR.
func RealIP(trustedProxies []*net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := netx.ResolveForwardedIP(r, trustedProxies)
			if ip == "" {
				ip = remoteIP(r)
			}
			next.ServeHTTP(w, r.WithContext(netx.WithClientIP(r.Context(), ip)))
		})
	}
}

func remoteIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
