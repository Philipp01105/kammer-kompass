package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Philipp01105/kammer-kompass/backend/internal/netx"
)

func TestRealIPIgnoresForwardedHeadersFromUntrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "198.51.100.10:12345"
	req.Header.Set("X-Real-IP", "203.0.113.77")

	var got string
	handler := RealIP(netx.ParseCIDRs([]string{"10.0.0.0/8"}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = netx.ClientIP(r)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got != "198.51.100.10" {
		t.Fatalf("expected remote address, got %q", got)
	}
}

func TestRealIPTrustsForwardedHeadersFromTrustedPeer(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.1.2.3:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.77, 10.1.2.3")

	var got string
	handler := RealIP(netx.ParseCIDRs([]string{"10.0.0.0/8"}))(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = netx.ClientIP(r)
	}))

	handler.ServeHTTP(httptest.NewRecorder(), req)

	if got != "203.0.113.77" {
		t.Fatalf("expected trusted forwarded address, got %q", got)
	}
}
