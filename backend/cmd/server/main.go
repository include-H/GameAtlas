package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/hao/game/internal/app"
	"github.com/hao/game/internal/config"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid configuration: %v", err)
	}
	if cfg.Proxy == "" {
		log.Printf("outbound proxy: disabled")
	} else {
		log.Printf("outbound proxy: %s", cfg.ProxyLogValue())
	}

	application, err := app.New(cfg)
	if err != nil {
		log.Fatalf("failed to initialize application: %v", err)
	}
	defer func() {
		if closeErr := application.Close(); closeErr != nil {
			log.Printf("failed to close application cleanly: %v", closeErr)
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	errCh := make(chan error, 1)
	go func() {
		errCh <- application.Run()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := application.Shutdown(shutdownCtx); err != nil {
			log.Fatalf("shutdown failed: %v", err)
		}
	case err := <-errCh:
		if err != nil {
			log.Fatalf("server failed: %v", err)
		}
	}
}
