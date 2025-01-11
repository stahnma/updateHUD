package storage

import (
	"encoding/json"
	"errors"
	"go.etcd.io/bbolt"
	"server/models"
)

type BboltStorage struct {
	db *bbolt.DB
}

func NewBboltStorage(dbPath string) (*BboltStorage, error) {
	db, err := bbolt.Open(dbPath, 0666, nil)
	if err != nil {
		return nil, err
	}
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("Systems"))
		return err
	})
	return &BboltStorage{db: db}, nil
}

func (s *BboltStorage) SaveSystem(hostname string, system models.System) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("Systems"))
		data, err := json.Marshal(system)
		if err != nil {
			return err
		}
		return bucket.Put([]byte(hostname), data)
	})
}

func (s *BboltStorage) GetSystem(hostname string) (models.System, error) {
	var system models.System
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("Systems"))
		data := bucket.Get([]byte(hostname))
		if data == nil {
			return errors.New("system not found")
		}
		return json.Unmarshal(data, &system)
	})
	return system, err
}

func (s *BboltStorage) GetAllSystems() ([]models.System, error) {
	var systems []models.System
	err := s.db.View(func(tx *bbolt.Tx) error {
		bucket := tx.Bucket([]byte("Systems"))
		return bucket.ForEach(func(k, v []byte) error {
			var system models.System
			if err := json.Unmarshal(v, &system); err != nil {
				return err
			}
			systems = append(systems, system)
			return nil
		})
	})
	return systems, err
}

func (s *BboltStorage) Close() error {
	return s.db.Close()
}

