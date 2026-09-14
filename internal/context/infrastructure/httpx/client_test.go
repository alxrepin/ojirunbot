package httpx

import (
	"net/http"
	"testing"
	"time"
)

func TestNewClientNoProxy(t *testing.T) {
	client, err := NewClient(5*time.Second, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if client.Timeout != 5*time.Second {
		t.Fatalf("timeout = %v, want 5s", client.Timeout)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}
	if transport.Proxy != nil {
		t.Fatal("proxy set without configuration, want direct connection")
	}
}

func TestNewClientZeroTimeout(t *testing.T) {
	for name, proxy := range map[string]string{
		"direct":  "",
		"proxied": "http://proxy.example.com:3128",
	} {
		client, err := NewClient(0, proxy)
		if err != nil {
			t.Fatalf("%s: unexpected error: %v", name, err)
		}
		if client.Timeout != 0 {
			t.Errorf("%s: timeout = %v, want 0", name, client.Timeout)
		}
	}
}

func TestNewClientWithProxyCredentials(t *testing.T) {
	client, err := NewClient(time.Second, "http://hetzner:p%40ss%3Aword@89.124.80.99:3128")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatalf("transport type = %T, want *http.Transport", client.Transport)
	}

	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://openrouter.ai/api/v1", nil)
	proxyURL, err := transport.Proxy(req)
	if err != nil {
		t.Fatalf("proxy func error: %v", err)
	}
	if proxyURL.Host != "89.124.80.99:3128" {
		t.Fatalf("proxy host = %q", proxyURL.Host)
	}
	user := proxyURL.User.Username()
	pass, _ := proxyURL.User.Password()
	if user != "hetzner" || pass != "p@ss:word" {
		t.Fatalf("credentials = %q:%q, want hetzner with decoded password", user, pass)
	}
	if transport.DisableKeepAlives || transport.IdleConnTimeout != proxyIdleTimeout || transport.MaxIdleConnsPerHost != proxyIdleConnsPerHost {
		t.Fatalf("proxied connections should be reused: keepalive off=%v idle=%v per host=%d",
			transport.DisableKeepAlives, transport.IdleConnTimeout, transport.MaxIdleConnsPerHost)
	}
}

func TestNewClientInvalidProxyURL(t *testing.T) {
	for name, raw := range map[string]string{
		"unparseable":    "://bad",
		"no host":        "http://",
		"scheme missing": "89.124.80.99:3128",
	} {
		if _, err := NewClient(time.Second, raw); err == nil {
			t.Errorf("%s: expected error for %q", name, raw)
		}
	}
}
