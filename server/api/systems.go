package api

import (
	"encoding/json"
	"net/http"
	"strings"
	"server/storage"
)

func GetAllSystemsHandler(store storage.Storage, w http.ResponseWriter, r *http.Request) {
	systems, err := store.GetAllSystems()
	if err != nil {
		http.Error(w, "Failed to fetch systems", http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(systems)
}

func GetSystemHandler(store storage.Storage, w http.ResponseWriter, r *http.Request) {
	hostname := strings.TrimPrefix(r.URL.Path, "/api/systems/")
	system, err := store.GetSystem(hostname)
	if err != nil {
		http.Error(w, "System not found", http.StatusNotFound)
		return
	}
	json.NewEncoder(w).Encode(system)
}

