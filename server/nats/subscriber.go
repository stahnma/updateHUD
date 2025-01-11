package nats

import (
	"encoding/json"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
	"server/models"
	"server/storage"
)

func StartSubscriber(store storage.Storage, natsURL string) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS at %s: %v", natsURL, err)
	}
	defer nc.Close()

	log.Println("Connected to NATS at", natsURL)

	_, err = nc.Subscribe("systems.updates.*", func(m *nats.Msg) {
		log.Printf("Received message on subject %s: %s", m.Subject, string(m.Data))

		// Parse and store the message
		var system models.System
		if err := json.Unmarshal(m.Data, &system); err != nil {
			log.Printf("Failed to parse message: %v", err)
			return
		}
		system.LastSeen = time.Now().Format(time.RFC3339)
		if err := store.SaveSystem(system.Hostname, system); err != nil {
			log.Printf("Failed to save system: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to subject: %v", err)
	}

	log.Println("Successfully subscribed to systems.updates.*")
	select {} // Keep the subscriber running
}
