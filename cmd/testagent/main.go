package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"fulcrumproject.org/core/cmd/testagent/agent"
	"fulcrumproject.org/core/cmd/testagent/config"
)

func main() {
	// Parse command line flags
	configPath := flag.String("config", "", "Path to configuration file")
	flag.Parse()

	// Load configuration
	var cfg *config.Config
	var err error

	if *configPath != "" {
		// Load from file if specified
		cfg, err = config.LoadFromFile(*configPath)
		if err != nil {
			log.Fatalf("Failed to load configuration from file: %v", err)
		}
		log.Printf("Loaded configuration from %s", *configPath)
	} else {
		// Use default configuration
		cfg = config.DefaultConfig()
		log.Printf("Using default configuration")
	}

	// Override with environment variables
	if err := cfg.LoadFromEnv(); err != nil {
		log.Fatalf("Failed to load configuration from environment: %v", err)
	} else {
		log.Printf("Applied environment variable overrides")
	}

	// Print configuration
	log.Printf("Config %#v", cfg)

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	log.Printf("Starting test agent with Fulcrum API at %s", cfg.FulcrumAPIURL)

	// Create and start the agent
	testAgent, err := agent.New(cfg)
	if err != nil {
		log.Fatalf("Failed to create agent: %v", err)
	}

	// Handle graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	// Start the agent
	if err := testAgent.Start(ctx); err != nil {
		log.Fatalf("Failed to start agent: %v", err)
	}

	log.Printf("Test agent started successfully (Agent ID: %s)", testAgent.GetAgentID())
	log.Printf("Press Ctrl+C to stop the agent")

	// Start a background goroutine to periodically display VM state counts
	go func() {
		displayTicker := time.NewTicker(30 * time.Second)
		defer displayTicker.Stop()
		for {
			select {
			case <-displayTicker.C:
				// Display VM state counts
				stateCounts := testAgent.GetVMStateCounts()
				log.Printf("VM States: %v", stateCounts)

				// Display job statistics
				processed, succeeded, failed := testAgent.GetJobStats()
				log.Printf("Jobs: Processed: %d, Succeeded: %d, Failed: %d",
					processed, succeeded, failed)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for termination signal
	<-sigCh
	log.Println("Received shutdown signal")

	// Create a context with timeout for graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shut down the agent
	if err := testAgent.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Error during shutdown: %v", err)
	}

	// Display final job statistics
	processed, succeeded, failed := testAgent.GetJobStats()
	log.Printf("Final Job Statistics: Processed: %d, Succeeded: %d, Failed: %d", processed, succeeded, failed)

	// Display final VM state counts
	stateCounts := testAgent.GetVMStateCounts()
	if len(stateCounts) > 0 {
		log.Printf("Final VM States: %v", stateCounts)
	}
	log.Printf("Agent uptime: %s", testAgent.GetUptime().Round(time.Second))
}
