package main

import (
	"flag" // to accept parameters from Cmd line arguments
	"log"

	"key-value-store/internal/config"
	"key-value-store/internal/server"
	"key-value-store/internal/store"
)

func main() {
	// Define and parse the config file path from a command-line flag
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to the configuration file") // StringVar tells go where to store the flag
	flag.Parse()                                                                // reads cmd line arguments and store it in actual path

	if configPath == "" {
		log.Fatal("Configuration file path is required. Use the -config flag.")
	}

	// Load the configuration
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}
	log.Printf("Starting node %s", cfg.NodeID)

	// Create the kv store
	kvStore := store.CreateKVStore()
	log.Println("KVStore initialized successfully")

	server := server.New(kvStore)
	log.Printf("HTTP server starting on %s", cfg.HTTPAddr)
	log.Fatal(server.Start(cfg.HTTPAddr))
}
