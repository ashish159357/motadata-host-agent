package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/motadata/motadata-host-agent/internal/agent"
	"github.com/motadata/motadata-host-agent/internal/config"
)

func main() {
	cfg := config.Load()

	svc := agent.NewService(cfg)

	httpServer := &http.Server{Addr: cfg.ListenAddr}

	go func() {
		log.Printf("HTTP server listening on %s", cfg.ListenAddr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("HTTP server error: %v", err)
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := svc.Run(); err != nil {
			log.Fatalf("agent service error: %v", err)
		}
	}()

	<-sigCh
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpServer.Shutdown(ctx)
}
