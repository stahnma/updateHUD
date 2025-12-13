package web

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"
	"server/api"
	"server/storage"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

//go:embed templates/*
var templateFiles embed.FS

//go:embed static/*
var staticFiles embed.FS

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// wsConn wraps a WebSocket connection with a mutex to serialize reads and writes
type wsConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

// writeJSON safely writes JSON to the connection with mutex protection
func (w *wsConn) writeJSON(v interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteJSON(v)
}

// writeControl safely writes control messages to the connection with mutex protection
func (w *wsConn) writeControl(messageType int, data []byte, deadline time.Time) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.WriteControl(messageType, data, deadline)
}

// readMessage safely reads messages from the connection with mutex protection
func (w *wsConn) readMessage() (messageType int, p []byte, err error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.ReadMessage()
}

// close safely closes the connection with mutex protection
func (w *wsConn) close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.conn.Close()
}

// WebSocket connection manager
type wsConnectionManager struct {
	sync.RWMutex
	connections map[*wsConn]bool
}

func newWSConnectionManager() *wsConnectionManager {
	return &wsConnectionManager{
		connections: make(map[*wsConn]bool),
	}
}

func (m *wsConnectionManager) add(conn *websocket.Conn) *wsConn {
	m.Lock()
	defer m.Unlock()
	wrapped := &wsConn{conn: conn}
	m.connections[wrapped] = true
	return wrapped
}

func (m *wsConnectionManager) remove(wrapped *wsConn) {
	m.Lock()
	defer m.Unlock()
	delete(m.connections, wrapped)
}

func (m *wsConnectionManager) broadcast(update interface{}) {
	// Create a snapshot of connections while holding the read lock
	m.RLock()
	conns := make([]*wsConn, 0, len(m.connections))
	for conn := range m.connections {
		conns = append(conns, conn)
	}
	m.RUnlock()

	// Now iterate over the snapshot without holding any lock
	// This allows goroutines to safely call remove() which acquires a write lock
	for _, wrapped := range conns {
		// Write in a non-blocking goroutine
		go func(c *wsConn) {
			if err := c.writeJSON(update); err != nil {
				log.Printf("[DEBUG] Failed to send WebSocket message to client: %v", err)
				c.close()
				m.remove(c)
			}
		}(wrapped)
	}
}

func StartWebServer(store storage.Storage, port string) {
	r := mux.NewRouter()
	connManager := newWSConnectionManager()

	// Serve static files (JS, CSS) from embedded filesystem
	// The embedded FS includes the "static/" prefix, so we need to use a subdirectory
	staticFS, err := fs.Sub(staticFiles, "static")
	if err != nil {
		log.Fatalf("[ERROR] Failed to create static files subdirectory: %v", err)
	}
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.FS(staticFS))))

	// API routes
	r.HandleFunc("/api/systems", api.GetSystemsHandler(store)).Methods("GET")
	r.HandleFunc("/api/systems/{hostname}", api.GetSystemHandler(store)).Methods("GET")

	// Serve the main page from embedded filesystem
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data, err := templateFiles.ReadFile("templates/index.html")
		if err != nil {
			log.Printf("[ERROR] Failed to read index.html: %v", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// WebSocket endpoint
	r.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			log.Printf("[ERROR] Failed to upgrade WebSocket connection: %v", err)
			return
		}

		// Configure WebSocket connection
		conn.SetReadLimit(512) // Limit size of incoming messages
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		conn.SetPongHandler(func(string) error {
			conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			return nil
		})

		// Add connection to manager and get wrapped connection
		wrappedConn := connManager.add(conn)

		// Create a context for the ping ticker to allow graceful shutdown
		pingCtx, pingCancel := context.WithCancel(context.Background())
		defer pingCancel()

		// Start ping ticker
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-pingCtx.Done():
					// Connection closed, stop the ticker
					return
				case <-ticker.C:
					// Use wrapped connection's writeControl which is thread-safe
					if err := wrappedConn.writeControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
						log.Printf("[DEBUG] Failed to send ping to client: %v", err)
						return
					}
				}
			}
		}()

		// Cleanup on exit
		defer func() {
			pingCancel() // Signal ping ticker to stop
			connManager.remove(wrappedConn)
			wrappedConn.close()
		}()

		// Keep connection alive and handle incoming messages
		for {
			// Use wrapped connection's readMessage which is thread-safe and serializes with writes
			messageType, _, err := wrappedConn.readMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[DEBUG] WebSocket read error: %v", err)
				}
				break
			}
			if messageType == websocket.PingMessage {
				// Use wrapped connection's writeControl which is thread-safe
				if err := wrappedConn.writeControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
					log.Printf("[DEBUG] Failed to send pong to client: %v", err)
					break
				}
			}
		}
	})

	// Subscribe to system updates and broadcast to all WebSocket clients
	go func() {
		for update := range store.SubscribeToUpdates() {
			connManager.broadcast(update)
		}
	}()

	log.Printf("[INFO] Web server starting on :%s", port)
	if err := http.ListenAndServe(":"+port, r); err != nil {
		log.Printf("[ERROR] Web server failed: %v", err)
	}
}
