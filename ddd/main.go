package main

import (
	"flag"
	"fmt"
	"os"

	"ddd/cmd"
	"ddd/config"
)

func main() {
	// Parse command line arguments
	var configPath string
	var debug bool
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&debug, "debug", false, "Enable debug output with stack trace")
	flag.Parse()

	// Load configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %v\n", err)
		if debug {
			panic(err)
		}
		os.Exit(1)
	}

	// Create and run application using builder
	//
	// Example: Add custom routes
	//
	// // 1. Create a handler function
	// func CreateUserHandler(svc *userapp.ApplicationService) gin.HandlerFunc {
	//     return func(ctx *gin.Context) {
	//         var req userapp.CreateUserRequest
	//         if err := ctx.ShouldBindJSON(&req); err != nil {
	//             ctx.JSON(400, gin.H{"error": err.Error()})
	//             return
	//         }
	//         user, err := svc.CreateUser(ctx.Request.Context(), req)
	//         if err != nil {
	//             ctx.JSON(500, gin.H{"error": err.Error()})
	//             return
	//         }
	//         ctx.JSON(200, user)
	//     }
	// }
	//
	// // 2. Register routes directly on the router
	// app := cmd.NewBuilder(cfg).
	//     WithController(userController).
	//     WithRoute("POST", "/api/v1/hello", func(ctx *gin.Context) {
	//         ctx.JSON(200, gin.H{"message": "hello"})
	//     }).
	//     Build()
	//
	app := cmd.NewBuilder(cfg).Build()

	if err := app.Run(); err != nil {
		fmt.Printf("Application error: %v\n", err)
		if debug {
			panic(err)
		}
		os.Exit(1)
	}
}
