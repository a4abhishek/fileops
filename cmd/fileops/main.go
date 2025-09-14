package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/a4abhishek/fileops/internal/cli"
	"github.com/a4abhishek/fileops/internal/config"
	"github.com/a4abhishek/fileops/internal/logger"
)

func main() {
	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\n🛑 Gracefully shutting down...")
		cancel()
	}()

	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to load configuration: %v\n", err)
		os.Exit(1)
	}

	// Initialize logger
	logConfig := logger.LoggingConfig{
		Level:  cfg.Logging.Level,
		Format: cfg.Logging.Format,
	}
	log, err := logger.New(logConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "❌ Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	// Create and execute CLI
	rootCmd := cli.NewRootCommand(ctx, cfg, log)
	if err := rootCmd.Execute(); err != nil {
		log.Error("Command execution failed", "error", err)
		os.Exit(1)
	}
}
