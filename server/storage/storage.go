package storage

import "server/models"

type Storage interface {
	SaveSystem(hostname string, system models.System) error
	GetSystem(hostname string) (models.System, error)
	GetAllSystems() ([]models.System, error)
	SubscribeToUpdates() <-chan models.System // This should be declared
}
