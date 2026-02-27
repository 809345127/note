package main

import (
	"flag"
	"fmt"
	"os"

	"ddd/cmd"
	"ddd/config"
)

type cliOptions struct {
	configPath string
	debug      bool
}

func main() {
	opts := parseCLIOptions()
	if err := run(opts.configPath); err != nil {
		handleStartupError(err, opts.debug)
	}
}

func parseCLIOptions() cliOptions {
	var configPath string
	var debug bool
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.BoolVar(&debug, "debug", false, "Enable debug output with stack trace")
	flag.Parse()

	return cliOptions{
		configPath: configPath,
		debug:      debug,
	}
}

func run(configPath string) error {
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	app := cmd.NewBuilder(cfg).Build()

	if err := app.Run(); err != nil {
		return fmt.Errorf("application error: %w", err)
	}

	return nil
}

func handleStartupError(err error, debug bool) {
	fmt.Printf("%v\n", err)
	if debug {
		panic(err)
	}
	os.Exit(1)
}
