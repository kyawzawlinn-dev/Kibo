package api

import (
	"net/http"
	"path/filepath"
	"strings"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"

	"github.com/gorilla/mux"
)

func NewRouter(repo *bodyrecord.Repository, ragService *chat2.RAGService, agent *chat2.ChatAgent) http.Handler {
	r := mux.NewRouter()
	handlers := NewHandlers(repo, ragService, agent)

	// --- API ROUTES ---
	r.HandleFunc("/api/chat/{chat_id}/message", handlers.SendMessage).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chat/new", handlers.CreateChat).Methods("POST", "OPTIONS")
	r.HandleFunc("/api/chats", handlers.ListChats).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.GetChat).Methods("GET", "OPTIONS")
	r.HandleFunc("/api/chat/{chat_id}", handlers.DeleteChat).Methods("DELETE", "OPTIONS")

	// --- STATIC FILES ---
	staticDir := "../frontend/build"
	fs := http.FileServer(http.Dir(staticDir))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// --- SPA FALLBACK ---
	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			http.NotFound(w, r)
			return
		}

		indexPath := filepath.Join(staticDir, "index.html")
		http.ServeFile(w, r, indexPath)
	}).Methods("GET")

	return r
}

// package api

// import (
// 	"net/http"

// 	"Kibo/backend/bodyrecord"
// 	"Kibo/backend/chat2"
// )

// // NewRouter sets up all the application routes
// func NewRouter(repo *bodyrecord.Repository, ragService *chat2.RAGService, agent *chat2.ChatAgent) http.Handler {
// 	mux := http.NewServeMux()

// 	// Create handlers and inject dependencies
// 	handlers := NewHandlers(repo, ragService, agent)

// 	// --- API Routes ---
// 	// Chat endpoint (the main AI endpoint)
// 	mux.HandleFunc("POST /api/chat", handlers.HandleChat)

// 	// Health record endpoints
// 	//mux.HandleFunc("POST /api/body-records", handlers.HandleCreateBodyRecord)
// 	//mux.HandleFunc("GET /api/body-records", handlers.HandleGetBodyRecords)

// 	// Diet record endpoints (example)
// 	//mux.HandleFunc("POST /api/diet-records", handlers.HandleCreateDietRecord)
// 	//mux.HandleFunc("GET /api/diet-records", handlers.HandleGetDietRecords)

// 	// --- Static File Server for React Frontend ---
// 	// This serves your built React app from the '../frontend/build' directory
// 	// All requests that don't match an API route will be served a file
// 	// or the React app's index.html (which handles client-side routing)

// 	// 1. Create a file server
// 	staticDir := "../frontend/build"
// 	fs := http.FileServer(http.Dir(staticDir))

// 	// 2. Wrap it with our SPA handler
// 	mux.Handle("/", spaHandler(http.StripPrefix("/", fs), staticDir))

// 	return mux
// }

// // spaHandler serves the 'index.html' for any non-API, non-file request.
// // This allows React Router to handle routes like '/body-records'
// func spaHandler(h http.Handler, staticDir string) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// Check if the file exists
// 		// This path logic is simplified; in production, you'd use 'fs.Stat'
// 		// For this example, we'll just check for common static assets
// 		isStaticAsset := len(r.URL.Path) > 8 && r.URL.Path[:8] == "/static/"

// 		if r.URL.Path == "/" || r.URL.Path == "/index.html" || isStaticAsset {
// 			// Serve the file (e.g., /index.html, /static/js/bundle.js)
// 			h.ServeHTTP(w, r)
// 		} else {
// 			// Serve 'index.html' for any other path (e.g., /dashboard, /chat)
// 			http.ServeFile(w, r, staticDir+"/index.html")
// 		}
// 	}
// }
