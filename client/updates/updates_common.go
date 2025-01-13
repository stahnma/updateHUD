package updates

import "runtime"

type Update struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// GetPendingUpdates determines the OS and delegates to the appropriate function
func GetPendingUpdates() []Update {
	switch runtime.GOOS {
	case "linux":
		return getLinuxUpdates()
	case "darwin":
		return getBrewUpdates()
	default:
		return nil
	}
}

// Helper for Linux systems to gather updates from various package managers
func getLinuxUpdates() []Update {
	var updates []Update
	updates = append(updates, getAptUpdates()...)
	updates = append(updates, getDnfUpdates()...)
	updates = append(updates, getYumUpdates()...)
	return updates
}
