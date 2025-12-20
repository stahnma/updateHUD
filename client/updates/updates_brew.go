package updates

import (
	"bufio"
	"log/slog"
	"os/exec"
	"strings"
)

// getBrewUpdates fetches updates from brew package manager
func getBrewUpdates() []Update {
	var updates []Update
	debugLog("Checking for brew updates")
	out, err := exec.Command("brew", "outdated").Output()
	if err != nil {
		slog.Error("Error checking updates with brew", "error", err)
		return updates
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		debugLog("Found brew update", "package", line)
		updates = append(updates, Update{
			Name:   line,
			Source: "brew",
		})
	}
	debugLog("Found brew updates", "count", len(updates))
	return updates
}
