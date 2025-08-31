package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/kirklin/boot-backend-go-clean/internal/app/server"
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

	// Create a new application instance
	app, err := server.NewApplication()
	if err != nil {
		log.Fatalf("Failed to create application: %v", err)
	}

	// Initialize the application
	if err := app.Initialize(); err != nil {
		log.Fatalf("Failed to initialize application: %v", err)
	}

	// Set up a channel to listen for interrupt signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start the application in a separate goroutine
	go func() {
		if err := app.Run(); err != nil {
			log.Fatalf("Failed to run application: %v", err)
		}
	}()

	log.Infof("Application is running on %s. Press CTRL+C to stop.", app.Config.ServerAddress)

	// Wait for interrupt signal
	<-stop

	log.Info("Shutting down gracefully...")

	// Perform cleanup
	app.Shutdown()

	if err := logger.GetLogger().Sync(); err != nil {
		log.Errorf("Failed to sync logger: %v", err)
	}

	log.Info("Application stopped")
}
