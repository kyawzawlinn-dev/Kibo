package api

import (
	"net/http"
	"strings"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"

	"github.com/gorilla/mux"
)

func NewRouter(repo *bodyrecord.Repository, agent *chat2.ChatAgent, ui http.FileSystem) http.Handler {
	r := mux.NewRouter()
	handlers := NewHandlers(repo, agent)

	// --- CHAT ROUTES ---
	r.HandleFunc("/api/chat/{chat_id}/message", handlers.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chat/new", handlers.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chats", handlers.ListChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.GetChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.DeleteChat).Methods("DELETE", "OPTIONS")

	// --- HEALTH RECORD ROUTES ---
	r.HandleFunc("/api/records/body", handlers.HandleGetBodyRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/records/body", handlers.HandleCreateBodyRecord).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/records/diet", handlers.HandleGetDietRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/records/diet", handlers.HandleCreateDietRecord).Methods("POST", "OPTIONS")

	// --- EMBEDDED FRONTEND + SPA FALLBACK ---
	r.PathPrefix("/").Handler(spaHandler(ui)).Methods("GET")

	return r
}

// spaHandler serves the embedded frontend: real files as-is, and
// index.html for any other path so client-side routing works.
func spaHandler(ui http.FileSystem) http.Handler {
	fileServer := http.FileServer(ui)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		path := strings.TrimSuffix(r.URL.Path, "/")
		if path == "" {
			path = "/index.html"
		}
		if f, err := ui.Open(path); err != nil {
			// not a real file -> SPA fallback to index.html
			r.URL.Path = "/"
		} else {
			f.Close()
		}

		fileServer.ServeHTTP(w, r)
	})
}
