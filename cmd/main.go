package main

import (
	"github.com/kirklin/boot-backend-go-clean/internal/app/server"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
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

	log.Printf("Application is running on %s. Press CTRL+C to stop.", app.Config.ServerAddress)

	// Wait for interrupt signal
	<-stop

	log.Println("Shutting down gracefully...")

	// Perform cleanup
	app.Shutdown()

	log.Println("Application stopped")
}
