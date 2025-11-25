package domain

import "testing"

func TestIsBlocked(t *testing.T) {
	reg := &Registry{
		DomainHashes: []uint64{
			HashString64("blocked.com"),
		},
		URLHashes: []uint64{
			HashString64("http://blocked.com/path"),
		},
		IPs: make(map[string]struct{}),
	}

	tests := []struct {
		name string
		n    NormalizedURL
		want bool
	}{
		{
			name: "https blocked by domain",
			n: NormalizedURL{
				Scheme: "https",
				Host:   "blocked.com",
				Path:   "/any",
			},
			want: true,
		},
		{
			name: "subdomain blocked by parent domain",
			n: NormalizedURL{
				Scheme: "https",
				Host:   "sub.blocked.com",
				Path:   "/",
			},
			want: true,
		},
		{
			name: "http blocked by full url",
			n: NormalizedURL{
				Scheme: "http",
				Host:   "blocked.com",
				Path:   "/path",
			},
			want: true,
		},
		{
			name: "http not blocked if only url differs",
			n: NormalizedURL{
				Scheme: "http",
				Host:   "other.com",
				Path:   "/path",
			},
			want: false,
		},
		{
			name: "https not blocked if domain not present",
			n: NormalizedURL{
				Scheme: "https",
				Host:   "other.com",
				Path:   "/",
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsBlocked(reg, tt.n)
			if got != tt.want {
				t.Errorf("IsBlocked() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsBlocked_IP(t *testing.T) {
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

func BenchmarkIsBlocked_Hit(b *testing.B) {
	reg := &Registry{
		DomainHashes: []uint64{HashString64("blocked.com")},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	n := NormalizedURL{Scheme: "https", Host: "blocked.com", Path: "/any"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if !IsBlocked(reg, n) {
			b.Fatalf("expected blocked")
		}
	}
}

func BenchmarkIsBlocked_Miss(b *testing.B) {
	reg := &Registry{
		DomainHashes: []uint64{HashString64("blocked.com")},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	n := NormalizedURL{Scheme: "https", Host: "other.com", Path: "/any"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if IsBlocked(reg, n) {
			b.Fatalf("expected not blocked")
		}
	}
}
