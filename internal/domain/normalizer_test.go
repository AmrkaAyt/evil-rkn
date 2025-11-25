package domain

import (
	"testing"
)

func TestNormalize_ValidURLs(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		want    NormalizedURL
		wantErr bool
	}{
		{
			name: "https simple",
			raw:  "https://Example.com",
			want: NormalizedURL{
				Scheme: "https",
				Host:   "example.com", // после idna.ToASCII и lowercase
				Path:   "/",
			},
		},
		{
			name: "http with path",
			raw:  "http://example.com/path/to/resource",
			want: NormalizedURL{
				Scheme: "http",
				Host:   "example.com",
				Path:   "/path/to/resource",
			},
		},
		{
			name: "https with port 443 (ignored)",
			raw:  "https://example.com:443/",
			want: NormalizedURL{
				Scheme: "https",
				Host:   "example.com",
				Path:   "/",
			},
		},
		{
			name: "IDN domain",
			raw:  "https://пример.рф/путь",
			// Host будет в ASCII (punycode), проверить только Scheme и Path,
			// Host проверим отдельно.
			want: NormalizedURL{
				Scheme: "https",
				Path:   "/путь",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Normalize(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got.Scheme != tt.want.Scheme {
				t.Errorf("Scheme = %q, want %q", got.Scheme, tt.want.Scheme)
			}
			if tt.want.Host != "" && got.Host != tt.want.Host {
				t.Errorf("Host = %q, want %q", got.Host, tt.want.Host)
			}
			if got.Path != tt.want.Path {
				t.Errorf("Path = %q, want %q", got.Path, tt.want.Path)
			}
		})
	}
}

func TestNormalize_MoreCases(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantURL string
	}{
		{
			name:    "https upper-case host and dirty path",
			raw:     "HTTPS://Example.COM:443/foo/../bar",
			wantURL: "https://example.com/bar",
		},
		{
			name:    "idn mixed case",
			raw:     "https://ПрИмер.Рф/",
			wantURL: "https://xn--e1afmkfd.xn--p1ai/",
		},
		{
			name:    "userinfo removed",
			raw:     "http://user:pass@example.com/path",
			wantURL: "http://example.com/path",
		},
		{
			name:    "ipv6 with port",
			raw:     "http://[2001:db8::1]:8080/path",
			wantURL: "http://2001:db8::1/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n, err := Normalize(tt.raw)
			if err != nil {
				t.Fatalf("Normalize() error = %v", err)
			}
			got := n.Scheme + "://" + n.Host + n.Path
			if got != tt.wantURL {
				t.Fatalf("got %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestNormalize_InvalidURLs(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		wantErr bool
	}{
		{name: "no scheme", raw: "example.com", wantErr: true},
		{name: "empty", raw: "", wantErr: true},
		{name: "invalid scheme", raw: "://example.com", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Normalize(tt.raw)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Normalize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
func TestIsBlocked_DomainAndSubdomain(t *testing.T) {
	reg := &Registry{
		DomainHashes: []uint64{
			HashString64("example.com"),
		},
		URLHashes: nil,
		IPs:       make(map[string]struct{}),
	}

	blockedURL := NormalizedURL{
		Scheme: "https",
		Host:   "sub.example.com",
		Path:   "/",
	}

	notBlockedURL := NormalizedURL{
		Scheme: "https",
		Host:   "anotherexample.com",
		Path:   "/",
	}

	if !IsBlocked(reg, blockedURL) {
		t.Errorf("expected sub.example.com to be blocked")
	}
	if IsBlocked(reg, notBlockedURL) {
		t.Errorf("expected anotherexample.com not to be blocked")
	}
}

func TestIsBlocked_IP_FromNormalizerFile(t *testing.T) {
	reg := &Registry{
		DomainHashes: nil,
		URLHashes:    nil,
		IPs:          map[string]struct{}{"203.0.113.5": {}},
	}

	n, err := Normalize("http://203.0.113.5/path")
	if err != nil {
		t.Fatalf("Normalize error: %v", err)
	}
	if !IsBlocked(reg, n) {
		t.Errorf("expected IP to be blocked")
	}
}

func BenchmarkNormalize(b *testing.B) {
	urls := []string{
		"https://example.com",
		"https://пример.рф/путь",
		"http://example.com/path/to/resource",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		raw := urls[i%len(urls)]
		if _, err := Normalize(raw); err != nil {
			b.Fatalf("Normalize error: %v", err)
		}
	}
}
