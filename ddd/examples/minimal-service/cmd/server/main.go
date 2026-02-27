package main

import (
	"fmt"
	"os"

	"ddd/examples/minimal-service/api"
	"ddd/examples/minimal-service/application"
	"ddd/examples/minimal-service/infrastructure/memory"
)

func main() {
	port := os.Getenv("MINIMAL_SERVICE_PORT")
	if port == "" {
		port = "8081"
	}

	repo := memory.NewTaskRepository()
	service := application.NewTaskService(repo)
	server := api.NewServer(service)

	if err := server.Run(":" + port); err != nil {
		fmt.Printf("minimal service failed: %v\n", err)
		os.Exit(1)
	}
}
