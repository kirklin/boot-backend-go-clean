package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/kirklin/boot-backend-go-clean/bootstrap"
)

func main() {
	// Create a new application instance
	app, err := bootstrap.NewApplication()
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

	// Start your application here
	// For example, you might start an HTTP server:
	// go startHTTPServer(app)

	log.Printf("Application is running. Press CTRL+C to stop.")

	// Wait for interrupt signal
	<-stop

	log.Println("Shutting down gracefully...")

	// Perform cleanup
	app.Shutdown()

	log.Println("Application stopped")
}

// You might have additional functions here, such as:
// func startHTTPServer(app *bootstrap.Application) {
//     // HTTP server setup and start logic
// }
