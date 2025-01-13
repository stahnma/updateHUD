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

	"github.com/nats-io/nats.go"
	"github.com/stahnma/mqttfun/client/updates"
)

type System struct {
	Hostname         string           `json:"hostname"`
	Architecture     string           `json:"architecture"`
	Ip               string           `json:"ip"`
	OS               string           `json:"os"`
	OSVersion        string           `json:"os_version"`
	UpdatesAvailable bool             `json:"updates_available"`
	PendingUpdates   []updates.Update `json:"pending_updates"`
	Timestamp        string           `json:"timestamp"`
}

// Collects all system data to prepare for publishing
func collectSystemData() (System, error) {
	var system System

	// Hostname
	hostname, err := os.Hostname()
	if err != nil {
		return system, err
	}
	system.Hostname = hostname

	// Architecture
	arch, err := exec.Command("uname", "-m").Output()
	if err != nil {
		return system, err
	}
	system.Architecture = strings.TrimSpace(string(arch))

	// OS and Version
	system.OS = runtime.GOOS
	if runtime.GOOS == "darwin" {
		// macOS version
		out, err := exec.Command("sw_vers", "-productVersion").Output()
		if err != nil {
			log.Printf("[ERROR] Failed to get macOS version: %v", err)
		} else {
			system.OSVersion = strings.TrimSpace(string(out))
		}
	} else if runtime.GOOS == "linux" {
		// Linux OS version
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

	// IP Address
	ip, err := getIPAddress()
	if err != nil {
		log.Printf("[ERROR] Failed to get IP address: %v", err)
	} else {
		system.Ip = ip
	}

	// Pending Updates
	system.PendingUpdates = updates.GetPendingUpdates()
	system.UpdatesAvailable = len(system.PendingUpdates) > 0

	// Timestamp
	system.Timestamp = time.Now().Format(time.RFC3339)

	return system, nil
}

// Publishes system data to NATS
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

	// Log message size
	log.Printf("[DEBUG] Message size: %d bytes", len(data))

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

// Finds the external IP address of the system
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

func main() {
	// NATS server URL with authentication
	natsURL := "nats://admin:password@192.168.1.206:4222"

	log.Printf("[DEBUG] Connecting to NATS at %s...", natsURL)
	nc, err := nats.Connect(natsURL,
		nats.Name("System Updates Publisher"),
		nats.Timeout(10*time.Second),
		nats.RetryOnFailedConnect(true),
		nats.MaxReconnects(-1), // Unlimited reconnections
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Printf("[INFO] Reconnected to NATS at %s", nc.ConnectedUrl())
		}),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			log.Printf("[ERROR] Disconnected from NATS: %v", err)
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

	// Send the first update immediately
	sendSystemUpdate(nc)

	// Run the client as a long-running daemon
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sendSystemUpdate(nc)
		}
	}
}
