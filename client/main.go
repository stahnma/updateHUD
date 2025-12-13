package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/cenkalti/backoff/v4"
	"github.com/nats-io/nats.go"
	"github.com/stahnma/mqttfun/client/updates"
)

// Global debug flag
var isDebugEnabled bool

func init() {
	// Check DEBUG environment variable
	debugEnv := strings.ToLower(os.Getenv("DEBUG"))
	isDebugEnabled = debugEnv == "1" || debugEnv == "true"
}

// debugLog prints a message only if debug logging is enabled
func debugLog(format string, v ...interface{}) {
	if isDebugEnabled {
		log.Printf("[DEBUG] "+format, v...)
	}
}

// getEnv gets an environment variable or returns a default value
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

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

	// Enhanced logging for NATS publishing
	debugLog("---- NATS Publishing Details ----")
	debugLog("Connected to NATS server: %s", nc.ConnectedUrl())
	debugLog("Client ID: %s", nc.ConnectedClusterName())
	debugLog("Connection Statistics:")
	debugLog("- Reconnects: %d", nc.Stats().Reconnects)
	debugLog("- Messages In: %d", nc.Stats().InMsgs)
	debugLog("- Messages Out: %d", nc.Stats().OutMsgs)
	debugLog("- Bytes In: %d", nc.Stats().InBytes)
	debugLog("- Bytes Out: %d", nc.Stats().OutBytes)
	debugLog("Message size: %d bytes", len(data))
	debugLog("System hostname: %s", system.Hostname)
	debugLog("System IP: %s", system.Ip)
	debugLog("Updates available: %v", system.UpdatesAvailable)
	if system.UpdatesAvailable {
		debugLog("Number of pending updates: %d", len(system.PendingUpdates))
	}

	// Publish the message
	subject := "systems.updates." + system.Hostname
	debugLog("Publishing to subject: %s", subject)
	debugLog("Message payload: %s", string(data))

	// Set a context with timeout for the publish operation
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to publish with timeout
	// Use buffered channel and ensure goroutine always completes to prevent leaks
	publishChan := make(chan error, 1)
	go func() {
		// Check if context is already cancelled before publishing
		select {
		case <-ctx.Done():
			publishChan <- ctx.Err()
			return
		default:
		}
		publishChan <- nc.Publish(subject, data)
	}()

	select {
	case err := <-publishChan:
		if err != nil {
			if err == ctx.Err() {
				log.Printf("[ERROR] Publish cancelled due to timeout")
			} else {
				log.Printf("[ERROR] Failed to publish message to NATS: %v", err)
			}
			return
		}
		log.Printf("[INFO] Successfully published to subject: %s", subject)

		// Try to flush with timeout using context
		flushCtx, flushCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer flushCancel()

		flushChan := make(chan error, 1)
		go func() {
			// Check if context is already cancelled before flushing
			select {
			case <-flushCtx.Done():
				flushChan <- flushCtx.Err()
				return
			default:
			}
			flushChan <- nc.Flush()
		}()

		select {
		case err := <-flushChan:
			if err != nil {
				if err == flushCtx.Err() {
					log.Printf("[ERROR] Flush timeout after 5 seconds")
				} else {
					log.Printf("[ERROR] Failed to flush NATS connection: %v", err)
				}
			} else {
				debugLog("Successfully flushed NATS connection")
			}
		case <-flushCtx.Done():
			log.Printf("[ERROR] Flush timeout after 5 seconds")
			// Drain the channel in a non-blocking way to prevent goroutine leak
			select {
			case <-flushChan:
			default:
			}
		}

	case <-ctx.Done():
		log.Printf("[ERROR] Publish timeout after 10 seconds")
		// Drain the channel in a non-blocking way to prevent goroutine leak
		// The goroutine will complete when nc.Publish() returns, but we don't wait
		select {
		case <-publishChan:
		default:
		}
	}

	debugLog("---- End NATS Publishing Details ----")
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
	// NATS server URL - configurable via environment variable
	// If not set, try to get server IP from NATS_SERVER_IP, or default to 192.168.1.157
	natsURL := getEnv("NATS_URL", "")
	if natsURL == "" {
		// Try to get server IP from environment or use 192.168.1.157
		serverIP := getEnv("NATS_SERVER_IP", "192.168.1.157")
		natsURL = fmt.Sprintf("nats://%s:4222", serverIP)
	}

	// Configure exponential backoff: start at 5s, max 10 minutes
	// Pattern: 5s, 10s, 20s, 40s, 80s, 160s, 320s, 600s (capped)
	backoffConfig := backoff.NewExponentialBackOff()
	backoffConfig.InitialInterval = 5 * time.Second
	backoffConfig.MaxInterval = 10 * time.Minute
	backoffConfig.Multiplier = 2.0
	backoffConfig.MaxElapsedTime = 10 * time.Minute
	backoffConfig.RandomizationFactor = 0

	var nc *nats.Conn
	connectFunc := func() error {
		log.Printf("[INFO] Attempting to connect to NATS at %s...", natsURL)
		var err error
		nc, err = nats.Connect(natsURL,
			nats.Name("System Updates Publisher"),
			nats.Timeout(30*time.Second),      // Increased timeout
			nats.PingInterval(20*time.Second), // Add periodic ping
			nats.MaxPingsOutstanding(5),       // Allow 5 outstanding pings
			nats.RetryOnFailedConnect(true),   // Enable automatic retry on initial connection failure
			nats.MaxReconnects(-1),            // Unlimited reconnections
			nats.ReconnectWait(5*time.Second), // Wait 5 seconds between reconnection attempts
			nats.ReconnectHandler(func(nc *nats.Conn) {
				log.Printf("[INFO] Reconnected to NATS at %s", nc.ConnectedUrl())
				debugLog("Connection statistics - Reconnects: %d, Messages In: %d, Messages Out: %d",
					nc.Stats().Reconnects, nc.Stats().InMsgs, nc.Stats().OutMsgs)
			}),
			nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
				if err != nil {
					log.Printf("[ERROR] Disconnected from NATS due to error: %v", err)
				} else {
					log.Printf("[INFO] Disconnected from NATS")
				}
			}),
			nats.ClosedHandler(func(nc *nats.Conn) {
				log.Printf("[INFO] Connection to NATS closed: %v", nc.LastError())
				debugLog("Final connection statistics - Reconnects: %d, Messages In: %d, Messages Out: %d",
					nc.Stats().Reconnects, nc.Stats().InMsgs, nc.Stats().OutMsgs)
			}),
		)
		if err != nil {
			return err
		}

		// Verify connection is actually ready by flushing
		// This ensures the connection handshake is complete
		if err := nc.FlushTimeout(5 * time.Second); err != nil {
			nc.Close()
			return fmt.Errorf("connection established but flush failed (connection not ready): %w", err)
		}

		return nil
	}

	// Retry connection with exponential backoff
	notifyFunc := func(err error, duration time.Duration) {
		log.Printf("[WARN] Connection attempt failed: %v. Retrying in %v...", err, duration)
	}

	err := backoff.RetryNotify(connectFunc, backoffConfig, notifyFunc)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to NATS after 10 minutes of retrying: %v", err)
	}
	defer nc.Close()

	log.Printf("[INFO] Successfully connected to NATS at %s", nc.ConnectedUrl())
	debugLog("Server ID: %s", nc.ConnectedServerId())

	// Function to check NATS connection health
	checkConnection := func() bool {
		if !nc.IsConnected() {
			// Log additional diagnostic info
			debugLog("Connection check failed - IsConnected: %v, LastError: %v, ConnectedUrl: %v",
				nc.IsConnected(), nc.LastError(), nc.ConnectedUrl())
			log.Printf("[WARN] NATS connection is not active")
			return false
		}
		return true
	}

	// Send the first update immediately
	if checkConnection() {
		sendSystemUpdate(nc)
	}

	// Run the client as a long-running daemon
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	// Add a separate ticker for connection health checks
	healthTicker := time.NewTicker(30 * time.Second)
	defer healthTicker.Stop()

	for {
		select {
		case <-ticker.C:
			if checkConnection() {
				sendSystemUpdate(nc)
			}
		case <-healthTicker.C:
			checkConnection()
		}
	}
}
