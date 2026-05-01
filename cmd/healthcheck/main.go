// Package main provides a lightweight health check binary for Docker HEALTHCHECK.
// It is compiled as a static binary and copied into the distroless production image.
package main

import (
	"fmt"
	"net/http"
	"os"
	"time"
)

func main() {
	port := os.Getenv("SERVER_PORT")
	if port == "" {
		port = "8888"
	}

	url := fmt.Sprintf("http://localhost:%s/v1/api/health", port)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "health check failed: status %d\n", resp.StatusCode)
		os.Exit(1)
	}

	os.Exit(0)
}
