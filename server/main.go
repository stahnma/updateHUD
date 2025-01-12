package main

import (
	"log"
	"server/nats"
	"server/storage"
	"server/web"
)

func main() {
	// Load configuration
	config := LoadConfig()

	// Initialize storage
	store, err := storage.NewBboltStorage(config.DBPath)
	if err != nil {
		log.Fatalf("[ERROR] Failed to initialize storage: %v", err)
	}
	defer store.Close()

	// Start NATS subscriber in a separate goroutine
	go func() {
		log.Println("[INFO] Starting NATS subscriber...")
		nats.StartSubscriber(store, config.NATSURL)
	}()

	// Start the web server
	log.Println("[INFO] Starting web server...")
	web.StartWebServer(store, config.HTTPPort)
}
