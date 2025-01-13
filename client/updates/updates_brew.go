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
	out, err := exec.Command("brew", "outdated").Output()
	if err != nil {
		log.Printf("Error checking updates with brew: %v", err)
		return updates
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		updates = append(updates, Update{
			Name:   line,
			Source: "brew",
		})
	}
	return updates
}
