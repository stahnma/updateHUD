package web

import (
	"log"
	"net/http"
	"server/api"
	"server/storage"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// WebSocket connection manager
type wsConnectionManager struct {
	sync.RWMutex
	connections map[*websocket.Conn]bool
}

func newWSConnectionManager() *wsConnectionManager {
	return &wsConnectionManager{
		connections: make(map[*websocket.Conn]bool),
	}
}

func (m *wsConnectionManager) add(conn *websocket.Conn) {
	m.Lock()
	m.connections[conn] = true
	m.Unlock()
}

func (m *wsConnectionManager) remove(conn *websocket.Conn) {
	m.Lock()
	delete(m.connections, conn)
	m.Unlock()
}

func (m *wsConnectionManager) broadcast(update interface{}) {
	m.RLock()
	defer m.RUnlock()

	for conn := range m.connections {
		// Write in a non-blocking goroutine
		go func(c *websocket.Conn) {
			if err := c.WriteJSON(update); err != nil {
				log.Printf("[DEBUG] Failed to send WebSocket message to client: %v", err)
				c.Close()
				m.remove(c)
			}
		}(conn)
	}
}

func StartWebServer(store storage.Storage, port string) {
	r := mux.NewRouter()
	connManager := newWSConnectionManager()

	// Serve static files (JS, CSS)
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", http.FileServer(http.Dir("./web/static"))))

	// API routes
	r.HandleFunc("/api/systems", api.GetSystemsHandler(store)).Methods("GET")
	r.HandleFunc("/api/systems/{hostname}", api.GetSystemHandler(store)).Methods("GET")

	// Serve the main page
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "./web/templates/index.html")
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

		// Add connection to manager
		connManager.add(conn)

		// Start ping ticker
		go func() {
			ticker := time.NewTicker(30 * time.Second)
			defer ticker.Stop()

			for {
				select {
				case <-ticker.C:
					if err := conn.WriteControl(websocket.PingMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
						log.Printf("[DEBUG] Failed to send ping to client: %v", err)
						return
					}
				}
			}
		}()

		// Cleanup on exit
		defer func() {
			connManager.remove(conn)
			conn.Close()
		}()

		// Keep connection alive and handle incoming messages
		for {
			messageType, _, err := conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("[DEBUG] WebSocket read error: %v", err)
				}
				break
			}
			if messageType == websocket.PingMessage {
				if err := conn.WriteControl(websocket.PongMessage, []byte{}, time.Now().Add(10*time.Second)); err != nil {
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
	log.Fatal(http.ListenAndServe(":"+port, r))
}
