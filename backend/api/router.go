package api

import (
	"net/http"
	"strings"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	"Kibo/backend/library"

	"github.com/gorilla/mux"
)

func NewRouter(repo *bodyrecord.Repository, agent *chat2.ChatAgent, lib *library.Library, ui http.FileSystem) http.Handler {
	r := mux.NewRouter()
	handlers := NewHandlers(repo, agent, lib)

	// --- CHAT ROUTES ---
	r.HandleFunc("/api/chat/{chat_id}/message", handlers.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chat/new", handlers.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chats", handlers.ListChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.GetChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.DeleteChat).Methods("DELETE", "OPTIONS")

	// --- PROFILE ROUTES ---
	r.HandleFunc("/api/profiles", handlers.HandleGetProfiles).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/profiles", handlers.HandleCreateProfile).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/profiles/{id}", handlers.HandleDeleteProfile).Methods("DELETE", "OPTIONS")

	// --- SHARE ROUTES ---
	r.HandleFunc("/api/share", handlers.HandleGetShareInfo).Methods("GET", "OPTIONS")

	// --- LIBRARY ROUTES ---
	r.HandleFunc("/api/library", handlers.HandleGetLibrary).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/library", handlers.HandleAddLibraryArticle).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/library/{id}", handlers.HandleUpdateLibraryArticle).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/library/{id}", handlers.HandleDeleteLibraryArticle).Methods("DELETE", "OPTIONS")

	// --- HEALTH LOG ROUTES ---
	r.HandleFunc("/api/health-log", handlers.HandleGetHealthLog).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/health-log", handlers.HandleAddHealthLogEntry).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/health-log/{id}", handlers.HandleUpdateHealthLogEntry).Methods("PUT", "OPTIONS")
	r.HandleFunc("/api/health-log/{id}", handlers.HandleDeleteHealthLogEntry).Methods("DELETE", "OPTIONS")

	// --- EMERGENCY ROUTES ---
	r.HandleFunc("/api/emergency", handlers.HandleGetEmergencyCards).Methods("GET", "OPTIONS")

	// --- HEALTH RECORD ROUTES ---
	r.HandleFunc("/api/records/export.csv", handlers.HandleExportRecordsCSV).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/records/import", handlers.HandleImportRecordsCSV).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/records/body", handlers.HandleGetBodyRecords).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/records/body", handlers.HandleCreateBodyRecord).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/records/day", handlers.HandleSaveDayRecords).Methods("POST", "OPTIONS")
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
