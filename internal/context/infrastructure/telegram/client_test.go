package telegram

import (
	"net/http"
	"testing"
)

func TestNewClientBaseURL(t *testing.T) {
	cases := map[string]struct {
		base         string
		wantBase     string
		wantFileBase string
	}{
		"default on empty": {
			base:         "",
			wantBase:     "https://api.telegram.org/botTOKEN",
			wantFileBase: "https://api.telegram.org/file/botTOKEN",
		},
		"custom reverse proxy": {
			base:         "https://tg.example.com",
			wantBase:     "https://tg.example.com/botTOKEN",
			wantFileBase: "https://tg.example.com/file/botTOKEN",
		},
		"trailing slash trimmed": {
			base:         "https://tg.example.com/",
			wantBase:     "https://tg.example.com/botTOKEN",
			wantFileBase: "https://tg.example.com/file/botTOKEN",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			c, err := NewClient(tc.base, "TOKEN", "")
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if c.baseURL != tc.wantBase {
				t.Errorf("baseURL = %q, want %q", c.baseURL, tc.wantBase)
			}
			if c.fileBaseURL != tc.wantFileBase {
				t.Errorf("fileBaseURL = %q, want %q", c.fileBaseURL, tc.wantFileBase)
			}
		})
	}
}

func TestNewClientProxy(t *testing.T) {
	direct, err := NewClient("", "TOKEN", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for name, client := range map[string]*http.Client{"rpc": direct.http, "download": direct.download} {
		transport, ok := client.Transport.(*http.Transport)
		if !ok {
			t.Fatalf("%s transport type = %T, want *http.Transport", name, client.Transport)
		}
		if transport.Proxy != nil {
			t.Errorf("%s: proxy set without configuration, want direct", name)
		}
	}

	proxied, err := NewClient("", "TOKEN", "http://proxy.example.com:3128")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodGet, "https://api.telegram.org/botTOKEN/getMe", nil)
	for name, client := range map[string]*http.Client{"rpc": proxied.http, "download": proxied.download} {
		proxyURL, err := client.Transport.(*http.Transport).Proxy(req)
		if err != nil {
			t.Fatalf("%s: proxy func error: %v", name, err)
		}
		if proxyURL == nil || proxyURL.Host != "proxy.example.com:3128" {
			t.Errorf("%s: proxy = %v, want proxy.example.com:3128", name, proxyURL)
		}
	}

	if _, err := NewClient("", "TOKEN", "://bad"); err == nil {
		t.Error("expected error for invalid proxy url")
	}
}
