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
	app := cmd.NewBuilder(cfg).Build()

	if err := app.Run(); err != nil {
		fmt.Printf("Application error: %v\n", err)
		if debug {
			panic(err)
		}
		os.Exit(1)
	}
}
