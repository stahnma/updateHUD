package updates

import "runtime"

type Update struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Source  string `json:"source"`
}

// UpdateResult contains the list of updates and whether a package manager was detected
type UpdateResult struct {
	Updates         []Update
	ManagerDetected bool
}

// GetPendingUpdates determines the OS and delegates to the appropriate function
func GetPendingUpdates() UpdateResult {
	switch runtime.GOOS {
	case "linux":
		return getLinuxUpdates()
	case "darwin":
		return getBrewUpdates()
	default:
		return UpdateResult{
			Updates:         nil,
			ManagerDetected: false,
		}
	}
}

// Helper for Linux systems to gather updates from various package managers
func getLinuxUpdates() UpdateResult {
	var allUpdates []Update
	managerDetected := false

	// Try each package manager and track if any were found
	aptResult := getAptUpdates()
	if aptResult.ManagerDetected {
		managerDetected = true
		allUpdates = append(allUpdates, aptResult.Updates...)
	}

	dnfResult := getDnfUpdates()
	if dnfResult.ManagerDetected {
		managerDetected = true
		allUpdates = append(allUpdates, dnfResult.Updates...)
	}

	yumResult := getYumUpdates()
	if yumResult.ManagerDetected {
		managerDetected = true
		allUpdates = append(allUpdates, yumResult.Updates...)
	}

	return UpdateResult{
		Updates:         allUpdates,
		ManagerDetected: managerDetected,
	}
}
