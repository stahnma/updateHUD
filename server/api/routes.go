package api

import (
	"encoding/json"
	"log"
	"net/http"
	"server/storage"
	"strings"

	"github.com/gorilla/mux"
)

type SystemSummary struct {
	Hostname         string `json:"hostname"`
	Architecture     string `json:"architecture"`
	Ip               string `json:"ip"`
	OS               string `json:"os"`
	OSVersion        string `json:"os_version"`
	UpdatesAvailable bool   `json:"updates_available"`
	LastSeen         string `json:"last_seen"`
}

// GetSystemsHandler returns a JSON list of systems without pending updates details
func GetSystemsHandler(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Retrieve all systems from storage
		systems, err := store.GetAllSystems()
		if err != nil {
			log.Printf("[ERROR] Failed to get all systems: %v", err)
			http.Error(w, "Failed to fetch systems", http.StatusInternalServerError)
			return
		}

		// Generate summarized system data
		summaries := make([]SystemSummary, 0, len(systems))
		for _, system := range systems {
			summaries = append(summaries, SystemSummary{
				Hostname:         system.Hostname,
				Architecture:     system.Architecture,
				Ip:               system.Ip,
				OS:               system.OS,
				OSVersion:        system.OSVersion,
				UpdatesAvailable: system.UpdatesAvailable,
				LastSeen:         system.LastSeen,
			})
		}

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(summaries); err != nil {
			log.Printf("[ERROR] Failed to encode systems summary: %v", err)
			http.Error(w, "Failed to encode systems summary", http.StatusInternalServerError)
		}
	}
}

// GetSystemHandler returns detailed information for a single system as JSON
func GetSystemHandler(store storage.Storage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Get the hostname from the URL
		vars := mux.Vars(r)
		hostname := strings.TrimSpace(vars["hostname"])

		// Retrieve the specific system from storage
		system, err := store.GetSystem(hostname)
		if err != nil {
			log.Printf("[ERROR] Failed to get system %s: %v", hostname, err)
			http.Error(w, "System not found", http.StatusNotFound)
			return
		}

		// Respond with JSON
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(system); err != nil {
			log.Printf("[ERROR] Failed to encode system details: %v", err)
			http.Error(w, "Failed to encode system details", http.StatusInternalServerError)
		}
	}
}
