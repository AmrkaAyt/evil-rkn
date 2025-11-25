package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"evil-rkn/internal/app"
	"evil-rkn/internal/config"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	if err := app.Run(ctx, cfg); err != nil {
		log.Fatalf("app error: %v", err)
	}
}
