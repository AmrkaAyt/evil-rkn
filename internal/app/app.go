package app

import (
	"context"
	"log"
	"time"

	"evil-rkn/internal/config"
	"evil-rkn/internal/registry"
	"evil-rkn/internal/transport/grpc"
	httpgw "evil-rkn/internal/transport/http"

	"golang.org/x/sync/errgroup"
)

func Run(ctx context.Context, cfg config.Config) error {
	holder := registry.NewHolder()
	client := registry.NewClient(cfg.RKNAPIBaseURL)

	updCfg := registry.Config{
		Interval:       cfg.UpdateInterval,
		InitialBackoff: 30 * time.Second,
		MaxBackoff:     30 * time.Minute,
	}

	g, ctx := errgroup.WithContext(ctx)

	g.Go(func() error {
		return registry.Start(ctx, updCfg, client, holder)
	})

	g.Go(func() error {
		return grpc.RunGRPCServer(ctx, cfg.GRPCAddr, holder)
	})

	g.Go(func() error {
		return httpgw.RunHTTPGatewayServer(ctx, cfg.HTTPAddr, cfg.GRPCAddr)
	})

	if err := g.Wait(); err != nil {
		log.Printf("app: servers stopped with error: %v", err)
		return err
	}

	log.Printf("app: servers stopped gracefully")
	return nil
}
