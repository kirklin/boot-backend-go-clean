package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		log.Println("Please provide a migration name")
		os.Exit(1)
	}

	migrationName := os.Args[1]
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d_%s", timestamp, migrationName)

	cmd := exec.Command("migrate", "create", "-ext", "sql", "-dir", "migrations", "-seq", fileName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error creating migration: %v\n", err)
		log.Println(string(output))
		os.Exit(1)
	}

	log.Println("Migration files created successfully")
	log.Println(string(output))
}
