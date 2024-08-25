package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Please provide a migration name")
		os.Exit(1)
	}

	migrationName := os.Args[1]
	timestamp := time.Now().Unix()
	fileName := fmt.Sprintf("%d_%s", timestamp, migrationName)

	cmd := exec.Command("migrate", "create", "-ext", "sql", "-dir", "migrations", "-seq", fileName)

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Error creating migration: %v\n", err)
		fmt.Println(string(output))
		os.Exit(1)
	}

	fmt.Println("Migration files created successfully")
	fmt.Println(string(output))
}
