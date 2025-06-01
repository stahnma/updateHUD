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
		debugLog("apt not found at /usr/bin/apt")
		return updates
	}

	debugLog("Checking for apt updates...")
	out, err := exec.Command("/usr/bin/apt", "list", "--upgradable").Output()
	if err != nil {
		log.Printf("[ERROR] Error checking updates with apt: %v", err)
		return updates
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 && !strings.HasPrefix(line, "Listing") {
			// Split package name into name and source
			parts := strings.Split(fields[0], "/")
			packageName := parts[0]
			source := "unknown"
			if len(parts) > 1 {
				source = parts[1]
			}
			version := fields[1]
			debugLog("Found update: package=%s version=%s source=%s", packageName, version, source)
			updates = append(updates, Update{
				Name:    packageName,
				Version: version,
				Source:  source,
			})
		}
	}
	debugLog("Found %d apt updates", len(updates))
	return updates
}

// getAptSource determines the source repository for a package
func getAptSource(packageName string) string {
	debugLog("Getting source for package: %s", packageName)
	out, err := exec.Command("/usr/bin/apt-cache", "policy", packageName).Output()
	if err != nil {
		log.Printf("[ERROR] Error checking repository source for package %s: %v", packageName, err)
		return "unknown"
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, " 500 ") {
			fields := strings.Fields(line)
			if len(fields) > 1 {
				source := fields[len(fields)-1]
				debugLog("Found source %s for package %s", source, packageName)
				return source
			}
		}
	}
	debugLog("No source found for package %s, using 'unknown'", packageName)
	return "unknown"
}
