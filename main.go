package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kiquetal/go-idp-caller/internal/config"
	"github.com/kiquetal/go-idp-caller/internal/jwks"
	"github.com/kiquetal/go-idp-caller/internal/server"
)

func main() {
	// Load configuration from environment variable or default
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger := config.InitLogger(cfg.Logging)
	logger.Info("Starting IDP JWS caller service")

	// Create JWKS manager
	manager := jwks.NewManager(logger)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start JWKS updaters for each IDP
	for _, idp := range cfg.IDPs {
		logger.Info("Starting updater for IDP", "name", idp.Name, "url", idp.URL, "interval", idp.RefreshInterval)
		updater := jwks.NewUpdater(idp, manager, logger)
		go updater.Start(ctx)
	}

	// Create and start HTTP server
	srv := server.New(cfg.Server, manager, logger)
	go func() {
		if err := srv.Start(); err != nil {
			logger.Error("Server failed", "error", err)
			cancel()
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	<-sigChan
	logger.Info("Received shutdown signal")

	// Graceful shutdown
	cancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("Server shutdown failed", "error", err)
	}

	logger.Info("Service stopped")
}
