package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"

	"github.com/gorilla/mux"
)

func NewRouter(repo *bodyrecord.Repository, agent *chat2.ChatAgent) http.Handler {
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

	// --- STATIC FILES + SPA FALLBACK (built frontend) ---
	staticDir := "../frontend/build"
	fs := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(staticDir, "index.html"))
	}).Methods("GET")

	return r
}
