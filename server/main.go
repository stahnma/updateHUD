package main

import (
	"log"
	"os"
	"os/signal"
	"server/nats"
	"server/storage"
	"server/web"
	"syscall"

	natsServer "github.com/nats-io/nats-server/v2/server"
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

	// Determine NATS URL - use embedded server if no external URL is provided
	var natsURL string
	var embeddedServer *natsServer.Server

	if config.NATSURL == "" || config.NATSURL == "embedded" {
		// Start embedded NATS server
		log.Println("[INFO] Starting embedded NATS server...")
		embeddedServer, natsURL, err = nats.StartEmbeddedServer(config.NATSPort)
		if err != nil {
			log.Fatalf("[ERROR] Failed to start embedded NATS server: %v", err)
		}
		defer func() {
			log.Println("[INFO] Shutting down embedded NATS server...")
			embeddedServer.Shutdown()
		}()
	} else {
		// Use external NATS server
		natsURL = config.NATSURL
		log.Printf("[INFO] Using external NATS server at %s", natsURL)
	}

	// Start NATS subscriber in a separate goroutine
	go func() {
		log.Println("[INFO] Starting NATS subscriber...")
		nats.StartSubscriber(store, natsURL)
	}()

	// Start the web server
	log.Println("[INFO] Starting web server...")
	go web.StartWebServer(store, config.HTTPPort)

	// Wait for interrupt signal to gracefully shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan
	log.Println("[INFO] Shutting down...")
}
