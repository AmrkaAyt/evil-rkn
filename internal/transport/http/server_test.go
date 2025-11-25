package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"evil-rkn/internal/domain"
	"evil-rkn/internal/registry"
	grpcTransport "evil-rkn/internal/transport/grpc"
	pb "evil-rkn/proto/gen"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func newTestHolder() *registry.Holder {
	h := registry.NewHolder()
	reg := &domain.Registry{
		DomainHashes: []uint64{domain.HashString64("blocked.com")},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	h.Set(reg)
	return h
}

func newTestGatewayMux(tb testing.TB, holder *registry.Holder) http.Handler {
	tb.Helper()

	srv := grpcTransport.NewServer(holder)

	mux := runtime.NewServeMux()
	if err := pb.RegisterBlockCheckerHandlerServer(context.Background(), mux, srv); err != nil {
		tb.Fatalf("failed to register gateway handler: %v", err)
	}

	return mux
}

func TestHTTPGateway_Blocked(t *testing.T) {
	holder := newTestHolder()
	h := newTestGatewayMux(t, holder)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/check?url=https://blocked.com", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "{\"blocked\":true}\n" && body != "{\"blocked\":true}" {
		t.Fatalf("body = %q, want blocked:true", body)
	}
}

func TestHTTPGateway_NotBlocked(t *testing.T) {
	holder := newTestHolder()
	h := newTestGatewayMux(t, holder)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/check?url=https://example.com", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}

	body := w.Body.String()
	if body != "{\"blocked\":false}\n" && body != "{\"blocked\":false}" {
		t.Fatalf("body = %q, want blocked:false", body)
	}
}

func TestHTTPGateway_InvalidURL(t *testing.T) {
	holder := newTestHolder()
	h := newTestGatewayMux(t, holder)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/check?url=example.com", nil)
	w := httptest.NewRecorder()

	h.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}
func newReadyzMux(h *registry.Holder) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		reg := h.Get()
		if reg == nil || reg.LastUpdated.IsZero() || len(reg.DomainHashes) == 0 {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("stale"))
			return
		}

		age := time.Since(reg.LastUpdated)
		if age < 0 || age > 48*time.Hour {
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte("stale"))
			return
		}

		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready"))
	})

	return mux
}

func TestReadyz_NotReady_WhenRegistryNotInitialized(t *testing.T) {
	h := registry.NewHolder() // LastUpdated = zero, DomainHashes = empty
	mux := newReadyzMux(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyz_NotReady_WhenRegistryTooOld(t *testing.T) {
	h := registry.NewHolder()

	reg := &domain.Registry{
		DomainHashes: []uint64{domain.HashString64("blocked.com")},
		IPs:          make(map[string]struct{}),
		LastUpdated:  time.Now().Add(-72 * time.Hour), // 3 дня назад
	}
	h.Set(reg)

	mux := newReadyzMux(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusServiceUnavailable)
	}
}

func TestReadyz_Ready_WhenFresh(t *testing.T) {
	h := registry.NewHolder()

	reg := &domain.Registry{
		DomainHashes: []uint64{domain.HashString64("blocked.com")},
		IPs:          make(map[string]struct{}),
		LastUpdated:  time.Now().Add(-time.Hour), // 1 час назад
	}
	h.Set(reg)

	mux := newReadyzMux(h)

	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	w := httptest.NewRecorder()

	mux.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func BenchmarkHTTPGateway_Check(b *testing.B) {
	holder := newTestHolder()
	h := newTestGatewayMux(b, holder)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/check?url=https://blocked.com", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			b.Fatalf("status = %d, want %d", w.Code, http.StatusOK)
		}
	}
}
