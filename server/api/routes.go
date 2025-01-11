package api

import (
	"log"
	"net/http"
	"server/storage"
)

func StartServer(store storage.Storage, port string) {
	http.HandleFunc("/api/systems", func(w http.ResponseWriter, r *http.Request) {
		GetAllSystemsHandler(store, w, r)
	})
	http.HandleFunc("/api/systems/", func(w http.ResponseWriter, r *http.Request) {
		GetSystemHandler(store, w, r)
	})

	http.Handle("/", http.FileServer(http.Dir("./web/static")))

	log.Printf("Starting server on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
