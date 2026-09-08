package ws

import (
	"net/http/httptest"
	"testing"
)

func TestCheckOriginAllowsSameHostnameWithProxyPort(t *testing.T) {
	req := httptest.NewRequest("GET", "http://192.168.0.117/api/v1/ws", nil)
	req.Host = "192.168.0.117"
	req.Header.Set("Origin", "http://192.168.0.117:18081")

	if !checkOrigin(req) {
		t.Fatal("expected same hostname with different proxy port to be allowed")
	}
}

func TestCheckOriginRejectsDifferentHostname(t *testing.T) {
	req := httptest.NewRequest("GET", "http://192.168.0.117/api/v1/ws", nil)
	req.Host = "192.168.0.117"
	req.Header.Set("Origin", "http://example.com")

	if checkOrigin(req) {
		t.Fatal("expected different hostname to be rejected")
	}
}
