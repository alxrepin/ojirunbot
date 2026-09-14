package httpx

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

const (
	proxyIdleTimeout      = 30 * time.Second
	proxyIdleConnsPerHost = 8
)

func NewClient(timeout time.Duration, proxyURL string) (*http.Client, error) {
	client := &http.Client{Timeout: timeout}
	if proxyURL == "" {
		direct := http.DefaultTransport.(*http.Transport).Clone()
		direct.Proxy = nil
		client.Transport = direct
		return client, nil
	}

	parsed, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("parse proxy url: %w", err)
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("proxy url has no host (expected scheme://[user:password@]host:port)")
	}

	headerTimeout := timeout - 5*time.Second
	if headerTimeout <= 0 {
		headerTimeout = timeout
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.Proxy = http.ProxyURL(parsed)
	transport.TLSHandshakeTimeout = 10 * time.Second
	transport.ResponseHeaderTimeout = headerTimeout
	transport.IdleConnTimeout = proxyIdleTimeout
	transport.MaxIdleConnsPerHost = proxyIdleConnsPerHost
	transport.ForceAttemptHTTP2 = false
	transport.TLSClientConfig = &tls.Config{NextProtos: []string{"http/1.1"}}
	client.Transport = transport
	return client, nil
}
