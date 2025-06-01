package nats

import (
	"encoding/json"
	"log"
	"server/models"
	"server/storage"
	"time"

	nats "github.com/nats-io/nats.go"
)

func StartSubscriber(store storage.Storage, natsURL string) {
	// Connect to NATS
	log.Printf("[DEBUG] ---- NATS Subscriber Details ----")
	log.Printf("[DEBUG] Attempting to connect to NATS at %s...", natsURL)
	nc, err := nats.Connect(natsURL,
		nats.Name("System Updates Subscriber"),
		nats.Timeout(10*time.Second),    // Set a 10-second timeout for the connection
		nats.RetryOnFailedConnect(true), // Retry if initial connection fails
		nats.MaxReconnects(5),           // Attempt to reconnect up to 5 times
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("[ERROR] Disconnected from NATS: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[INFO] Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("[INFO] Connection to NATS closed. Reason: %v", nc.LastError())
		}),
	)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Printf("[INFO] Successfully connected to NATS at %s", nc.ConnectedUrl())
	log.Printf("[DEBUG] Server ID: %s", nc.ConnectedServerId())
	log.Printf("[DEBUG] Client ID: %s", nc.ConnectedClusterName())

	// Subscribe to the subject
	subject := "systems.updates.>"
	log.Printf("[DEBUG] Subscribing to subject pattern: %s", subject)
	sub, err := nc.Subscribe(subject, func(m *nats.Msg) {
		log.Printf("[DEBUG] ---- Received NATS Message ----")
		log.Printf("[DEBUG] Subject: %s", m.Subject)
		log.Printf("[DEBUG] Reply: %s", m.Reply)
		log.Printf("[DEBUG] Message size: %d bytes", len(m.Data))
		log.Printf("[DEBUG] Raw message: %s", string(m.Data))

		// Parse the message into a System struct
		var system models.System
		if err := json.Unmarshal(m.Data, &system); err != nil {
			log.Printf("[ERROR] Failed to unmarshal message: %v", err)
			return
		}

		// Log parsed system data
		log.Printf("[DEBUG] Parsed system data:")
		log.Printf("[DEBUG] - Hostname: %s", system.Hostname)
		log.Printf("[DEBUG] - IP: %s", system.Ip)
		log.Printf("[DEBUG] - OS: %s %s", system.OS, system.OSVersion)
		log.Printf("[DEBUG] - Updates Available: %v", system.UpdatesAvailable)
		if system.UpdatesAvailable {
			log.Printf("[DEBUG] - Number of Updates: %d", len(system.PendingUpdates))
			for _, update := range system.PendingUpdates {
				log.Printf("[DEBUG] - Update: %s %s from %s", update.Name, update.Version, update.Source)
			}
		}

		// Store the system data
		if err := store.SaveSystem(system.Hostname, system); err != nil {
			log.Printf("[ERROR] Failed to save system data for %s: %v", system.Hostname, err)
		} else {
			log.Printf("[INFO] Successfully saved system data for %s", system.Hostname)
		}
		log.Printf("[DEBUG] ---- End NATS Message Processing ----")
	})
	if err != nil {
		log.Fatalf("[ERROR] Failed to subscribe to subject %s: %v", subject, err)
	}

	log.Printf("[INFO] Successfully subscribed to subject %s. Subscription ID: %v", subject, sub)
	log.Printf("[DEBUG] ---- End NATS Subscriber Details ----")

	// Keep the subscriber running
	log.Println("[INFO] NATS subscriber is now running and listening for messages...")
	select {} // Block forever to keep the subscriber running
}
