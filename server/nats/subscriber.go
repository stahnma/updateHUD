package nats

import (
	"encoding/json"
	"log"
	"time"

	nats "github.com/nats-io/nats.go"
	"server/storage"
	"server/models"
)

func StartSubscriber(store storage.Storage, natsURL string) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("Failed to connect to NATS: %v", err)
	}
	defer nc.Close()

	_, err = nc.Subscribe("systems.updates.*", func(m *nats.Msg) {
		var system models.System
		if err := json.Unmarshal(m.Data, &system); err != nil {
			log.Printf("Failed to parse system update: %v", err)
			return
		}
		system.LastSeen = time.Now().Format(time.RFC3339)
		if err := store.SaveSystem(system.Hostname, system); err != nil {
			log.Printf("Failed to save system: %v", err)
		}
	})
	if err != nil {
		log.Fatalf("Failed to subscribe to NATS topic: %v", err)
	}

	log.Println("NATS subscriber started")
	select {} // Keep the subscriber running
}

