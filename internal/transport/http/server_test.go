package http

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"evil-rkn/internal/domain"
	reginfra "evil-rkn/internal/registry"
	grpcTransport "evil-rkn/internal/transport/grpc"
	pb "evil-rkn/proto/gen"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
)

func newTestHolder() *reginfra.Holder {
	h := reginfra.NewHolder()
	reg := &domain.Registry{
		DomainHashes: []uint64{domain.HashString64("blocked.com")},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	h.Set(reg)
	return h
}

func newTestGatewayMux(tb testing.TB, holder *reginfra.Holder) http.Handler {
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
