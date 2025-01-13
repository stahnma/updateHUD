package updates

import (
	"bufio"
	"log"
	"os"
	"os/exec"
	"strings"
)

// getYumUpdates fetches updates from yum package manager
func getYumUpdates() []Update {
	var updates []Update
	if _, err := os.Stat("/usr/bin/yum"); err != nil {
		return updates
	}

	out, err := exec.Command("/usr/bin/yum", "check-update").Output()
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok && exitError.ExitCode() != 100 {
			log.Printf("Error checking updates with yum: %v", err)
		}
	}

	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) >= 3 && !strings.HasPrefix(line, "Loaded plugins") {
			updates = append(updates, Update{
				Name:    fields[0],
				Version: fields[1],
				Source:  fields[2],
			})
		}
	}
	return updates
}
