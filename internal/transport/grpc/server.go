package grpc

import (
	"context"
	"log"
	"net"
	"strings"

	"evil-rkn/internal/domain"
	"evil-rkn/internal/registry"
	pb "evil-rkn/proto/gen"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

type Server struct {
	pb.UnimplementedBlockCheckerServer
	holder *registry.Holder
}

func NewServer(holder *registry.Holder) *Server {
	return &Server{holder: holder}
}

const maxURLLen = 2048

func (s *Server) Check(ctx context.Context, req *pb.CheckRequest) (*pb.CheckResponse, error) {
	rawURL := strings.TrimSpace(req.GetUrl())
	if rawURL == "" {
		return nil, status.Error(codes.InvalidArgument, "url is required")
	}
	if len(rawURL) > maxURLLen {
		return nil, status.Error(codes.InvalidArgument, "url is too long")
	}

	n, err := domain.Normalize(rawURL)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid url: %v", err)
	}

	reg := s.holder.Get()
	if reg == nil {
		return nil, status.Error(codes.Unavailable, "registry not initialized")
	}

	blocked := domain.IsBlocked(reg, n)

	return &pb.CheckResponse{Blocked: blocked}, nil
}

// RunGRPCServer starts a gRPC server on the given address and
// shuts it down gracefully when the context is canceled.
func RunGRPCServer(ctx context.Context, addr string, holder *registry.Holder) error {
	if addr == "" {
		// Reasonable default if nothing is provided.
		addr = ":9090"
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	s := grpc.NewServer()
	pb.RegisterBlockCheckerServer(s, NewServer(holder))
	reflection.Register(s)

	// Stop the server once the context is done (SIGTERM, timeout, etc.).
	go func() {
		<-ctx.Done()
		s.GracefulStop()
	}()

	log.Printf("gRPC server listening on %s", lis.Addr().String())
	return s.Serve(lis)
}
