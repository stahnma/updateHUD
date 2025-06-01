package updates

import (
	"bufio"
	"log"
	"os/exec"
	"strings"
)

// getBrewUpdates fetches updates from brew package manager
func getBrewUpdates() []Update {
	var updates []Update
	debugLog("Checking for brew updates...")
	out, err := exec.Command("brew", "outdated").Output()
	if err != nil {
		log.Printf("[ERROR] Error checking updates with brew: %v", err)
		return updates
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		debugLog("Found brew update: %s", line)
		updates = append(updates, Update{
			Name:   line,
			Source: "brew",
		})
	}
	debugLog("Found %d brew updates", len(updates))
	return updates
}
