package grpc

import (
	"context"
	"net"
	"testing"
	"time"

	"evil-rkn/internal/domain"
	reginfra "evil-rkn/internal/registry"
	pb "evil-rkn/proto/gen"

	"google.golang.org/grpc"
)

func newTestGRPCHolder() *reginfra.Holder {
	h := reginfra.NewHolder()
	reg := &domain.Registry{
		DomainHashes: []uint64{domain.HashString64("blocked.com")},
		URLHashes:    nil,
		IPs:          make(map[string]struct{}),
	}
	h.Set(reg)
	return h
}

func startTestGRPCServer(t *testing.T, holder *reginfra.Holder) (addr string, stop func()) {
	t.Helper()

	lis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to listen: %v", err)
	}

	s := grpc.NewServer()
	pb.RegisterBlockCheckerServer(s, NewServer(holder))

	go func() {
		_ = s.Serve(lis)
	}()

	return lis.Addr().String(), s.GracefulStop
}

func TestGRPCCheck_Blocked(t *testing.T) {
	holder := newTestGRPCHolder()
	addr, stop := startTestGRPCServer(t, holder)
	defer stop()

	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		t.Fatalf("failed to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewBlockCheckerClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	resp, err := client.Check(ctx, &pb.CheckRequest{Url: "https://blocked.com"})
	if err != nil {
		t.Fatalf("Check error: %v", err)
	}

	if !resp.Blocked {
		t.Fatalf("expected blocked=true, got false")
	}
}
