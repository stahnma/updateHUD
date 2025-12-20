package nats

import (
	"log/slog"
	"time"

	natsServer "github.com/nats-io/nats-server/v2/server"
)

// StartEmbeddedServer starts an embedded NATS server and returns the server instance
// and the connection URL. The server runs in a goroutine.
func StartEmbeddedServer(port int) (*natsServer.Server, string, error) {
	opts := &natsServer.Options{
		Host:       "0.0.0.0", // Listen on all interfaces to allow remote clients
		Port:       port,
		MaxPayload: 1024 * 1024, // 1MB
		LogFile:    "",
		Logtime:    true,
		Debug:      false,
		Trace:      false,
	}

	// Create and start the NATS server
	ns, err := natsServer.NewServer(opts)
	if err != nil {
		return nil, "", err
	}

	// Start the server in a goroutine
	go ns.Start()

	// Wait for the server to be ready
	if !ns.ReadyForConnections(10 * time.Second) {
		ns.Shutdown()
		return nil, "", natsServer.ErrServerNotRunning
	}

	// Get the server URL
	serverURL := ns.ClientURL()
	slog.Info("Embedded NATS server started", "url", serverURL, "id", ns.ID())

	return ns, serverURL, nil
}
