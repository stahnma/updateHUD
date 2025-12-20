package nats

import (
	"encoding/json"
	"log/slog"
	"os"
	"server/models"
	"server/storage"
	"time"

	nats "github.com/nats-io/nats.go"
)

func StartSubscriber(store storage.Storage, natsURL string) {
	// Connect to NATS
	slog.Debug("NATS Subscriber Details")
	slog.Debug("Attempting to connect to NATS", "url", natsURL)
	nc, err := nats.Connect(natsURL,
		nats.Name("System Updates Subscriber"),
		nats.Timeout(10*time.Second),    // Set a 10-second timeout for the connection
		nats.RetryOnFailedConnect(true), // Retry if initial connection fails
		nats.MaxReconnects(5),           // Attempt to reconnect up to 5 times
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			slog.Error("Disconnected from NATS", "error", err)
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			slog.Info("Reconnected to NATS", "url", nc.ConnectedUrl())
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			slog.Info("Connection to NATS closed", "reason", nc.LastError())
		}),
	)
	if err != nil {
		slog.Error("Failed to connect to NATS", "error", err)
		os.Exit(1)
	}
	defer nc.Close()
	slog.Info("Successfully connected to NATS", "url", nc.ConnectedUrl())
	slog.Debug("Server ID", "id", nc.ConnectedServerId())
	slog.Debug("Client ID", "id", nc.ConnectedClusterName())

	// Subscribe to the subject
	subject := "systems.updates.>"
	slog.Debug("Subscribing to subject pattern", "subject", subject)
	sub, err := nc.Subscribe(subject, func(m *nats.Msg) {
		slog.Debug("Received NATS Message", "subject", m.Subject, "reply", m.Reply, "size", len(m.Data))
		slog.Debug("Raw message", "data", string(m.Data))

		// Parse the message into a System struct
		var system models.System
		if err := json.Unmarshal(m.Data, &system); err != nil {
			slog.Error("Failed to unmarshal message", "error", err)
			return
		}

		// Log parsed system data
		slog.Debug("Parsed system data",
			"hostname", system.Hostname,
			"ip", system.Ip,
			"os", system.OS,
			"os_version", system.OSVersion,
			"updates_available", system.UpdatesAvailable,
			"update_count", len(system.PendingUpdates))

		if system.UpdatesAvailable {
			for _, update := range system.PendingUpdates {
				slog.Debug("Update", "name", update.Name, "version", update.Version, "source", update.Source)
			}
		}

		// Store the system data
		slog.Debug("Calling SaveSystem", "hostname", system.Hostname)
		startTime := time.Now()
		if err := store.SaveSystem(system.Hostname, system); err != nil {
			slog.Error("Failed to save system data", "hostname", system.Hostname, "error", err, "duration_ms", time.Since(startTime).Milliseconds())
		} else {
			duration := time.Since(startTime)
			slog.Info("Successfully saved system data", "hostname", system.Hostname, "duration_ms", duration.Milliseconds())
		}
	})
	if err != nil {
		slog.Error("Failed to subscribe to subject", "subject", subject, "error", err)
		os.Exit(1)
	}

	slog.Info("Successfully subscribed to subject", "subject", subject, "subscription_id", sub)
	slog.Debug("End NATS Subscriber Details")

	// Keep the subscriber running
	slog.Info("NATS subscriber is now running and listening for messages...")
	select {} // Block forever to keep the subscriber running
}
