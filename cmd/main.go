package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/kirklin/boot-backend-go-clean/internal/app/server"
	"github.com/kirklin/boot-backend-go-clean/pkg/banner"
	"github.com/kirklin/boot-backend-go-clean/pkg/logger"
)

func main() {

	config := logger.NewDefaultConfig()

	if config.FileConfig.Environment == "" {
		config.FileConfig.Environment = "development"
	}

	if err := logger.InitLogger(config); err != nil {
		panic(err)
	}
	log := logger.GetLogger()

	// Print startup banner (Spring Boot style)
	banner.Print(os.Stdout, config.FileConfig.Environment)

	// Create a new application instance
	app, err := server.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Initialize the application
	if err := app.Initialize(); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Create a context that is cancelled on SIGINT or SIGTERM.
	// When a signal is received, ctx.Done() fires and app.Run
	// automatically drains in-flight requests and shuts down cleanly.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Run blocks until the context is cancelled, then performs graceful shutdown.
	if err := app.Run(ctx); err != nil {
		log.Fatalf("Application error: %v", err)
	}

	if err := log.Sync(); err != nil {
		// Ignore sync errors on stdout/stderr — these are not real files
		// and produce "inappropriate ioctl for device" on macOS.
		// This is a known Zap issue: https://github.com/uber-go/zap/issues/991
	}
}

