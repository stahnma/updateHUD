package web

import (
	"log"
	"net/http"
	"server/api"
	"server/storage"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func StartWebServer(store storage.Storage, port string) {
	r := mux.NewRouter()

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
		defer conn.Close()

		// Stream updates to the WebSocket client
		for update := range store.SubscribeToUpdates() {
			if err := conn.WriteJSON(update); err != nil {
				log.Printf("[ERROR] Failed to send WebSocket message: %v", err)
				break
			}
		}
	})

	log.Printf("Web server starting on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}
