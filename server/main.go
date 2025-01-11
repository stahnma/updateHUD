package main

import (
	"log"
	"server/api"
	"server/nats"
	"server/storage"
)

func main() {
	// Load configuration
	config := LoadConfig()

	// Initialize storage
	store, err := storage.NewBboltStorage(config.DBPath)
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Start NATS subscriber
	go nats.StartSubscriber(store, config.NATSURL)

	// Start HTTP API
	api.StartServer(store, config.HTTPPort)
}
