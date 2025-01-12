package storage

import (
	"encoding/json"
	"fmt"
	"log"
	"server/models"
	"sync"
	"time"

	bolt "go.etcd.io/bbolt"
)

type BboltStorage struct {
	sync.Mutex
	db      *bolt.DB
	updates chan models.System
}

func NewBboltStorage(dbPath string) (*BboltStorage, error) {
	db, err := bolt.Open(dbPath, 0666, nil)
	if err != nil {
		return nil, err
	}

	return &BboltStorage{
		db:      db,
		updates: make(chan models.System, 100), // Buffered channel for updates
	}, nil
}

func (s *BboltStorage) SaveSystem(hostname string, system models.System) error {
	s.Lock()
	defer s.Unlock()
	system.LastSeen = time.Now().Format(time.RFC3339) // Add current timestamp

	err := s.db.Update(func(tx *bolt.Tx) error {
		bucket, err := tx.CreateBucketIfNotExists([]byte("systems"))
		if err != nil {
			return err
		}
		data, err := json.Marshal(system)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(hostname), data)
	})
	if err != nil {
		log.Printf("[ERROR] Failed to save system: %v", err)
		return err
	}

	// Send update to the updates channel
	select {
	case s.updates <- system:
	default:
		log.Printf("[WARNING] Updates channel is full; dropping update for %s", hostname)
	}

	return nil
}

func (s *BboltStorage) GetSystem(hostname string) (models.System, error) {
	var system models.System

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("systems"))
		if bucket == nil {
			return fmt.Errorf("bucket 'systems' not found")
		}

		data := bucket.Get([]byte(hostname))
		if data == nil {
			return fmt.Errorf("system with hostname '%s' not found", hostname)
		}

		if err := json.Unmarshal(data, &system); err != nil {
			return fmt.Errorf("failed to unmarshal system data: %v", err)
		}

		return nil
	})

	if err != nil {
		return models.System{}, err
	}

	return system, nil
}

func (s *BboltStorage) GetAllSystems() ([]models.System, error) {
	var systems []models.System

	err := s.db.View(func(tx *bolt.Tx) error {
		bucket := tx.Bucket([]byte("systems"))
		if bucket == nil {
			return fmt.Errorf("bucket 'systems' not found")
		}

		return bucket.ForEach(func(k, v []byte) error {
			var system models.System
			if err := json.Unmarshal(v, &system); err != nil {
				log.Printf("[ERROR] Failed to unmarshal system data for key %s: %v", k, err)
				return err
			}
			systems = append(systems, system)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return systems, nil
}

func (s *BboltStorage) SubscribeToUpdates() <-chan models.System {
	return s.updates
}

func (s *BboltStorage) Close() error {
	close(s.updates)
	return s.db.Close()
}
