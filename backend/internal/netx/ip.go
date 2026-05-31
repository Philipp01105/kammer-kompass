package netx

import (
	"context"
	"net"
	"net/http"
	"strings"
)

type ctxClientIPKey struct{}

// WithClientIP stores a resolved IP in the request context (set by RealIP middleware).
func WithClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, ctxClientIPKey{}, ip)
}

// ClientIP returns the client IP. If the RealIP middleware ran, it returns the
// validated forwarded IP. Otherwise it falls back to r.RemoteAddr directly —
// no forwarded headers are trusted without middleware validation.
func ClientIP(r *http.Request) string {
	if ip, ok := r.Context().Value(ctxClientIPKey{}).(string); ok && ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// ParseCIDRs parses a slice of CIDR strings. Invalid entries are silently skipped.
func ParseCIDRs(cidrs []string) []*net.IPNet {
	out := make([]*net.IPNet, 0, len(cidrs))
	for _, s := range cidrs {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		_, ipNet, err := net.ParseCIDR(s)
		if err == nil {
			out = append(out, ipNet)
		}
	}
	return out
}

// peerIP extracts the bare IP from r.RemoteAddr.
func peerIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err != nil {
		host = strings.TrimSpace(r.RemoteAddr)
	}
	return net.ParseIP(host)
}

// ResolveForwardedIP reads X-Real-IP / X-Forwarded-For only when the direct
// peer is in trustedCIDRs. Returns empty string when untrusted.
func ResolveForwardedIP(r *http.Request, trusted []*net.IPNet) string {
	if len(trusted) == 0 {
		return ""
	}
	peer := peerIP(r)
	if peer == nil {
		return ""
	}
	isTrusted := false
	for _, cidr := range trusted {
		if cidr.Contains(peer) {
			isTrusted = true
			break
		}
	}
	if !isTrusted {
		return ""
	}
	if ip := strings.TrimSpace(r.Header.Get("X-Real-IP")); net.ParseIP(ip) != nil {
		return ip
	}
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		if len(parts) > 0 {
			if ip := strings.TrimSpace(parts[0]); net.ParseIP(ip) != nil {
				return ip
			}
		}
	}
	return ""
}
