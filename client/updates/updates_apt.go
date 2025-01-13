package updates

import (
	"log"
	"os"
	"os/exec"
	"strings"
)

// getAptUpdates fetches updates from apt package manager
func getAptUpdates() []Update {
	var updates []Update
	if _, err := os.Stat("/usr/bin/apt"); err != nil {
		return updates
	}

	out, err := exec.Command("/usr/bin/apt", "list", "--upgradable").Output()
	if err != nil {
		log.Printf("Error checking updates with apt: %v", err)
		return updates
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && !strings.HasPrefix(line, "Listing") {
			packageName := fields[0]
			version := fields[1]
			source := getAptSource(packageName)
			updates = append(updates, Update{
				Name:    packageName,
				Version: version,
				Source:  source,
			})
		}
	}
	return updates
}

// getAptSource determines the source repository for a package
func getAptSource(packageName string) string {
	out, err := exec.Command("/usr/bin/apt-cache", "policy", packageName).Output()
	if err != nil {
		log.Printf("Error checking repository source for package %s: %v", packageName, err)
		return "unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, " 500 ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				return fields[len(fields)-1]
			}
		}
	}
	return "unknown"
}
