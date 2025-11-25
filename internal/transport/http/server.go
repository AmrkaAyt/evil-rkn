package http

import (
	"context"
	"evil-rkn/internal/registry"
	"log"
	"net/http"
	"time"

	pb "evil-rkn/proto/gen"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func RunHTTPGatewayServer(ctx context.Context, httpAddr, grpcEndpoint string, holder *registry.Holder) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// Initialize gRPC-Gateway mux
	gwMux := runtime.NewServeMux()
	opts := []grpc.DialOption{grpc.WithTransportCredentials(insecure.NewCredentials())}
	if err := pb.RegisterBlockCheckerHandlerFromEndpoint(ctx, gwMux, grpcEndpoint, opts); err != nil {
		return err
	}

	// Main HTTP mux, routing all requests through the gRPC-Gateway
	mux := http.NewServeMux()
	mux.Handle("/", gwMux)

	// /healthz — basic liveness check
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// /readyz — readiness check; in production it can be replaced with real gRPC health probing
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, r *http.Request) {
		reg := holder.Get()
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

	srv := &http.Server{
		Addr:         httpAddr,
		Handler:      mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown of the HTTP server when the parent context is canceled
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			log.Printf("http gateway: graceful shutdown error: %v", err)
		}
	}()

	log.Printf("HTTP gateway listening on %s, proxying to gRPC %s", httpAddr, grpcEndpoint)
	return srv.ListenAndServe()
}
