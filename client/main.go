package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	nats "github.com/nats-io/nats.go"
)

type Update struct {
	Name    string `json:"name"`
	Version string `json:"version,omitempty"`
	Source  string `json:"source"`
}

type System struct {
	Hostname         string   `json:"hostname"`
	Architecture     string   `json:"architecture"`
	Ip               string   `json:"ip"`
	OS               string   `json:"os"`
	OSVersion        string   `json:"os_version"`
	UpdatesAvailable bool     `json:"updates_available"`
	PendingUpdates   []Update `json:"pending_updates"`
	Timestamp        string   `json:"timestamp"`
}

func collectSystemData() (System, error) {
	var system System

	// Get hostname
	hostname, err := os.Hostname()
	if err != nil {
		return system, err
	}
	system.Hostname = hostname

	// Get architecture
	arch, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return system, err
	}
	system.Architecture = strings.TrimSpace(string(arch))

	// Get OS and version
	system.OS = runtime.GOOS
	if runtime.GOOS == "darwin" {
		// Get macOS version using sw_vers
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			log.Printf("[ERROR] Failed to get macOS version: %v", err)
		} else {
			system.OSVersion = strings.TrimSpace(string(out))
		}
	} else if runtime.GOOS == "linux" {
		// Get Linux OS version
		if _, err := os.Stat("/etc/os-release"); err == nil {
			content, _ := os.ReadFile("/etc/os-release")
			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				if strings.HasPrefix(line, "PRETTY_NAME=") {
					system.OS = strings.Trim(strings.TrimPrefix(line, "PRETTY_NAME="), `"`)
				}
			}
		}
	}

	// Get IP address
	ip, err := getIPAddress()
	if err != nil {
		log.Printf("[ERROR] Failed to get IP address: %v", err)
	} else {
		system.Ip = ip
	}

	// Get pending updates
	system.PendingUpdates = getPendingUpdates()
	system.UpdatesAvailable = len(system.PendingUpdates) > 0

	// Timestamp
	system.Timestamp = time.Now().Format(time.RFC3339)

	return system, nil
}

func getPendingUpdates() []Update {
	var updates []Update

	// Check for updates on Linux with apt
	if runtime.GOOS == "linux" {
		out, err := exec.Command("apt", "list", "--upgradable").Output()
		if err != nil {
			log.Printf("[ERROR] Failed to check apt updates: %v", err)
			return updates
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				updates = append(updates, Update{
					Name:    fields[0],
					Version: fields[1],
					Source:  "apt",
				})
			}
		}
	}

	// Check for updates on macOS with Homebrew
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("brew", "outdated").Output()
		if err != nil {
			log.Printf("[ERROR] Failed to check Homebrew updates: %v", err)
			return updates
		}
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if len(strings.TrimSpace(line)) > 0 {
				updates = append(updates, Update{
					Name:   line,
					Source: "brew",
				})
			}
		}
	}

	return updates
}

func main() {
	// NATS server URL with authentication
	natsURL := "nats://admin:password@localhost:4222"

	log.Printf("[DEBUG] Attempting to connect to NATS at %s...", natsURL)
	nc, err := nats.Connect(natsURL,
		nats.Name("System Updates Publisher"),
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(5),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("[ERROR] Disconnected from NATS: %v", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[INFO] Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Printf("[INFO] Connection to NATS closed: %v", nc.LastError())
		}),
	)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to NATS: %v", err)
	}
	defer nc.Close()
	log.Println("[INFO] Successfully connected to NATS")

	// Run the client as a long-running daemon
	ticker := time.NewTicker(1 * time.Minute) // Set the interval to 5 minutes
	defer ticker.Stop()

	// Send an immediate update
	sendSystemUpdate(nc)

	// Periodically send updates
	for {
		select {
		case <-ticker.C:
			sendSystemUpdate(nc)
		}
	}
}

func sendSystemUpdate(nc *nats.Conn) {
	// Collect system data
	system, err := collectSystemData()
	if err != nil {
		log.Printf("[ERROR] Failed to collect system data: %v", err)
		return
	}

	// Marshal system data to JSON
	data, err := json.Marshal(system)
	if err != nil {
		log.Printf("[ERROR] Failed to marshal system data: %v", err)
		return
	}

	// Publish the message
	subject := "systems.updates." + system.Hostname
	log.Printf("[DEBUG] Publishing to subject: %s", subject)
	log.Printf("[DEBUG] Message: %s", string(data))

	if err := nc.Publish(subject, data); err != nil {
		log.Printf("[ERROR] Failed to publish message to NATS: %v", err)
	} else {
		log.Printf("[INFO] Successfully published to subject: %s", subject)
	}
}

func getIPAddress() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", err
	}

	for _, addr := range addrs {
		// Check if the address is an IP network
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ip := ipNet.IP
			// Ensure it's IPv4 and not a link-local address
			if ip.To4() != nil && !ip.IsLinkLocalUnicast() {
				return ip.String(), nil
			}
		}
	}

	return "", fmt.Errorf("no valid external IP address found")
}
