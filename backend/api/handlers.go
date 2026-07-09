package api

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	logger "Kibo/backend/kibo_utils"

	"github.com/gorilla/mux"
)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	repo       *bodyrecord.Repository
	ragService *chat2.RAGService
	agent      *chat2.ChatAgent
}

// NewHandlers creates a new Handlers struct
func NewHandlers(repo *bodyrecord.Repository, ragService *chat2.RAGService, agent *chat2.ChatAgent) *Handlers {
	return &Handlers{
		repo:       repo,
		ragService: ragService,
		agent:      agent,
	}
}

// --- Chat Handler ---
// ---- response for user creating a new chat session ----
type CreateChatResponse struct {
	ChatID int64  `json:"chat_id"`
	Title  string `json:"title"`
}

// ---- response for user getting chat history ----
type GetChatResponse struct {
	ChatID   int64                    `json:"chat_id"`
	Title    string                   `json:"title"`
	Messages []bodyrecord.ChatHistory `json:"message"`
}

// ---- request for user sending a message ----
type ChatRequest struct {
	ChatID  int64  `json:"chat_id"`
	Message string `json:"message"`
}

// ---- response for user sending a message ----
type ChatResponse struct {
	Reply string `json:"reply"`
	Title string `json:"title,omitempty"` // New title if generated
}

// CreateChat creates a new chat row and returns chat_id
func (h *Handlers) CreateChat(w http.ResponseWriter, r *http.Request) {
	// For demo we use fixed user id. Replace with auth in production.
	userID := int64(2)

	chatID, err := h.repo.CreateChat(r.Context(), userID)
	if err != nil {
		logger.Error("[Handlers.go/CreateChat]:\tCreateChat: %v", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, CreateChatResponse{
		ChatID: chatID,
		Title:  "New Chat",
	})
}

// HandleChat handles sending a message to a chat (save user msg, call AI, save reply)
func (h *Handlers) HandleChat(w http.ResponseWriter, r *http.Request) {

	var req ChatRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	userID := int64(2)
	chatID := req.ChatID

	// 1. Save user's message
	if _, err := h.repo.AddChatHistory(ctx, chatID, &userID, "user", req.Message); err != nil {
		logger.Error("[Handlers.go/HandleChat]:\tsaving user message: %v", err)
		http.Error(w, "Failed to save user message", http.StatusInternalServerError)
		return
	}

	var title string

	// 2. Title logic: if first message -> generate title and update chat
	isFirst, err := h.repo.IsFirstMessage(chatID)
	if err != nil {
		logger.Error("[Handlers.go/HandleChat]:\tchecking first message: %v", err)
	}

	if isFirst {
		title, err = h.agent.GenerateTitle(ctx, req.Message)
		if err != nil {
			logger.Warn("[Handlers.go/HandleChat]:\tfailed to generate title: %v", err)
			title = ""
		} else {
			if err := h.repo.UpdateChatTitle(chatID, title); err != nil {
				logger.Warn("[Handlers.go/HandleChat]:\tfailed to update chat title: %v", err)
			}
		}
	} else {
		// retrieve existing title (best-effort)
		title, _ = h.repo.RetrieveChatTitle(chatID)
	}

	// 3. Generate AI reply
	reply, err := h.agent.Refine_prompt(ctx, userID, req.Message)
	if err != nil {

		logger.Error("[Handlers.go/HandleChat]:\tChatAgent.Refine_prompt: %v", err)

		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
		return
	}

	// 4. Save assistant reply (userID = nil)
	if _, err := h.repo.AddChatHistory(ctx, chatID, nil, "assistant", reply); err != nil {
		logger.Error("[Handlers.go/HandleChat]:\tsaving assistant reply: %v", err)
		// not fatal for the client; we'll still return the reply
	}

	// 5. Return reply + optional title
	writeJSON(w, http.StatusOK, ChatResponse{
		Reply: reply,
		Title: title,
	})
}

// GetChat returns chat metadata and messages
func (h *Handlers) GetChat(w http.ResponseWriter, r *http.Request) {

	// vars := map[string]string{}

	// // try to get chat_id from URL path
	// if rc := r.Context().Value("vars"); rc != nil {
	// 	if m, ok := rc.(map[string]string); ok {
	// 		vars = m
	// 	}
	// }

	// if using gorilla mux, use vars from request
	// but to keep portability we attempt both:
	chatIDStr := ""

	if chi := r.URL.Query().Get("chat_id"); chi != "" {
		chatIDStr = chi
	} else {
		// extract from path using gorilla mux style
		// gorilla/mux sets the vars; use mux.Vars(r) if available
		// if not available, extract last segment
		// simpler: try mux.Vars via type assertion
		if muxVars := muxVarsFromContext(r); muxVars != nil {
			chatIDStr = muxVars["chat_id"]
		}
	}

	if chatIDStr == "" {
		// fallback: try parsing path suffix
		// /api/chat/{chat_id}
		// last segment:
		segments := splitPath(r.URL.Path)
		if len(segments) > 0 {
			chatIDStr = segments[len(segments)-1]
		}
	}

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	title, err := h.repo.RetrieveChatTitle(chatID)

	if err != nil {
		logger.Error("[Handlers.go/GetChat]:\tRetrieveChatTitle: %v", err)
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	msgs, err := h.repo.GetChatHistory(r.Context(), chatID, 1000) // return up to 1000 messages (or choose a limit)

	if err != nil {
		logger.Error("[Handlers.go/GetChat]:\tGetChatHistory: %v", err)
		http.Error(w, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, GetChatResponse{
		ChatID:   chatID,
		Title:    title,
		Messages: msgs,
	})
}

// ListChats returns all chats for the user
func (h *Handlers) ListChats(w http.ResponseWriter, r *http.Request) {

	userID := int64(2) // demo user

	chats, err := h.repo.ListChatsByUser(r.Context(), userID)

	if err != nil {
		logger.Error("[Handlers.go/ListChats]:\tListChatsByUser: %v", err)
		http.Error(w, "Failed to list chats", http.StatusInternalServerError)
		return
	}

	// jsonData, _ := json.MarshalIndent(chats, "", "  ")
	// logger.Debug("[Handlers.go/ListChats]:\t/api/chats RESPONSE:", string(jsonData))

	writeJSON(w, http.StatusOK, chats)
}

// DeleteChat deletes a chat and all its messages
func (h *Handlers) DeleteChat(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	chatIDStr := vars["chat_id"]

	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)

	if err != nil {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	// check ownership
	userID := int64(2)

	belongs, err := h.repo.ChatBelongsToUser(r.Context(), chatID, userID)

	if err != nil {
		logger.Error("[Handlers.go/DeleteChat]:\tChatBelongsToUser: %v", err)
		http.Error(w, "Failed to verify ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.repo.DeleteChat(r.Context(), chatID); err != nil {
		logger.Error("[Handlers.go/DeleteChat]:\tDeleteChat: %v", err)
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	logger.Info("[Handlers.go/DeleteChat]:\tChat deleted successfully: %d", chatID)

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "chat_id": chatID})
}

// NEW: Send message to specific chat
func (h *Handlers) SendMessage(w http.ResponseWriter, r *http.Request) {

	// --------------------------------------------
	// 🔥 FIX 1: Immediately handle OPTIONS (CORS)
	// Without this, OPTIONS enters JSON decoder → 400 Invalid JSON
	// --------------------------------------------
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	vars := mux.Vars(r)
	chatIDStr := vars["chat_id"]
	chatID, err := strconv.ParseInt(chatIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

	// --------------------------------------------
	// 🔥 FIX 2: Decode JSON ONLY for actual POST
	// --------------------------------------------
	var body struct {
		Message string `json:"message"`
	}

	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if body.Message == "" {
		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
		return
	}

	userID := int64(2)

	// Save user message
	_, err = h.repo.AddChatHistory(r.Context(), chatID, &userID, "user", body.Message)
	if err != nil {
		http.Error(w, "Failed to save message", 500)
		return
	}

	// Generate title if first message
	isFirst, _ := h.repo.IsFirstMessage(chatID)
	var title string

	if isFirst {
		title, _ = h.agent.GenerateTitle(r.Context(), body.Message)
		h.repo.UpdateChatTitle(chatID, title)
	} else {
		title, _ = h.repo.RetrieveChatTitle(chatID)
	}

	// Generate AI reply
	reply, err := h.agent.Refine_prompt(r.Context(), userID, body.Message)
	if err != nil {
		http.Error(w, "AI Error", 500)
		return
	}

	// Save reply
	h.repo.AddChatHistory(r.Context(), chatID, nil, "assistant", reply)

	writeJSON(w, http.StatusOK, ChatResponse{
		Reply: reply,
		Title: title,
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {

	w.Header().Set("Content-Type", "application/json")

	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("❗️ERROR in [Handlers.go]:  Failed to write JSON response: %v", err)
	}
}

// small helper to split path segments
func splitPath(path string) []string {

	// naive split ignoring empty segments
	var out []string

	start := 0

	for i := 0; i < len(path); i++ {
		if path[i] == '/' {
			if i > start {
				out = append(out, path[start:i])
			}
			start = i + 1
		}
	}
	if start < len(path) {
		out = append(out, path[start:])
	}
	return out
}

// muxVarsFromContext returns vars if gorilla/mux has stored them in the request context
// this avoids importing mux package directly in this file (keeps the handler generic)
func muxVarsFromContext(r *http.Request) map[string]string {

	if r == nil {
		return nil
	}

	// gorilla/mux stores vars in request context under key "vars"
	// the exact type assertion below will work when mux is used
	if v := r.Context().Value("vars"); v != nil {
		if m, ok := v.(map[string]string); ok {
			return m
		}
	}

	// fallback: try mux.Vars if available (importing mux would be another option)
	return nil
}

// --- Health Data Handlers (unchanged, using repo pointer) ---

// --- Health Data Handlers ---

func (h *Handlers) HandleCreateBodyRecord(w http.ResponseWriter, r *http.Request) {

	var record bodyrecord.BodyRecord

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set server-side fields
	record.UserID = 2 // demo user
	record.Timestamp = time.Now()

	id, err := h.repo.AddBodyRecord(r.Context(), record)
	if err != nil {
		log.Printf("ERROR: Failed to create body record: %v", err)
		http.Error(w, "Failed to save record", http.StatusInternalServerError)
		return
	}
	record.ID = id

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handlers) HandleGetBodyRecords(w http.ResponseWriter, r *http.Request) {

	userID := int64(2) // demo user

	records, err := h.repo.GetBodyRecords(r.Context(), userID)

	if err != nil {
		log.Printf("ERROR: Failed to get body records: %v", err)
		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// --- UPDATED DIET HANDLERS ---

func (h *Handlers) HandleCreateDietRecord(w http.ResponseWriter, r *http.Request) {

	var record bodyrecord.DietRecord

	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Set server-side fields
	record.UserID = 2 // demo user
	record.Timestamp = time.Now()

	id, err := h.repo.AddDietRecord(r.Context(), record)
	if err != nil {
		log.Printf("ERROR: Failed to create diet record: %v", err)
		http.Error(w, "Failed to save record", http.StatusInternalServerError)
		return
	}
	record.ID = id

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handlers) HandleGetDietRecords(w http.ResponseWriter, r *http.Request) {

	userID := int64(2) // demo user

	records, err := h.repo.GetDietRecords(r.Context(), userID)

	if err != nil {
		logger.Error("[Handlers.go/HandleGetDietRecords]:\tFailed to get diet records: %v", err)
		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// package api

// import (
// 	"encoding/json"
// 	"log"
// 	"net/http"
// 	"time"

// 	"Kibo/backend/bodyrecord"
// 	"Kibo/backend/chat2"
// )

// // Handlers holds dependencies for HTTP handlers
// type Handlers struct {
// 	repo       *bodyrecord.Repository
// 	ragService *chat2.RAGService
// 	agent      *chat2.ChatAgent
// }

// // NewHandlers creates a new Handlers struct
// func NewHandlers(repo *bodyrecord.Repository, ragService *chat2.RAGService, agent *chat2.ChatAgent) *Handlers {
// 	return &Handlers{
// 		repo:       repo,
// 		ragService: ragService,
// 		agent:      agent,
// 	}
// }

// // --- Chat Handler ---

// type ChatRequest struct {
// 	ChatID  int64  `json:"chat_id"`
// 	Message string `json:"message"`
// }

// type ChatResponse struct {
// 	Reply string `json:"reply"`
// 	Title string `json:"title,omitempty"` // New title if generated
// }

// func (h *Handlers) HandleChat(w http.ResponseWriter, r *http.Request) {
// 	var req ChatRequest
// 	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	if req.Message == "" {
// 		http.Error(w, "Message cannot be empty", http.StatusBadRequest)
// 		return
// 	}

// 	userID := int64(2)   // tmp fixed user ID for demo
// 	chatID := req.ChatID // In real app, validate chat ownership

// 	// Save user's message
// 	if _, err := h.repo.AddChatMessage(r.Context(), chatID, &userID, "user", req.Message); err != nil {
// 		http.Error(w, "Failed to save user message", http.StatusInternalServerError)
// 		return
// 	}

// 	// Check if first message in this chat → auto-generate title
// 	isFirst, err := h.repo.IsFirstMessage(chatID)
// 	if err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]: checking first message: %v", err)
// 	}
// 	log.Printf("🛑Debug in [Handlers.go]: isFirstMessage: <%v>", isFirst)
// 	var newTitle string
// 	if isFirst {
// 		newTitle, err = h.agent.GenerateTitle(r.Context(), req.Message)
// 		if err := h.repo.UpdateChatTitle(chatID, newTitle); err != nil {
// 			log.Printf("❗️ERROR in [Handlers.go]: updating chat title: %v", err)
// 		}
// 	}

// 	if newTitle == "" {
// 		newTitle, err = h.repo.RetrieveChatTitle(chatID)
// 		if err != nil {
// 			log.Printf("❗️ERROR in [Handlers.go]: retrieving chat title: %v", err)
// 		}
// 	}

// 	log.Printf("🛑Debug in [Handlers.go]: title: %s", newTitle)

// 	reply, err := h.agent.Refine_prompt(r.Context(), userID, req.Message)
// 	if err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]:  ChatAgent failed: %v", err)
// 		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
// 		return
// 	}

// 	// Save assistant reply
// 	if _, err := h.repo.AddChatMessage(r.Context(), chatID, nil, "assistant", reply); err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]: saving assistant reply: %v", err)
// 	}

// 	writeJSON(w, http.StatusOK, ChatResponse{
// 		Reply: reply,
// 		Title: newTitle})
// }

// func writeJSON(w http.ResponseWriter, status int, v any) {
// 	w.Header().Set("Content-Type", "application/json")
// 	w.WriteHeader(status)
// 	if err := json.NewEncoder(w).Encode(v); err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]:  Failed to write JSON response: %v", err)
// 	}
// }

// // --- Health Data Handlers (unchanged, using repo pointer) ---

// func (h *Handlers) HandleCreateBodyRecord(w http.ResponseWriter, r *http.Request) {
// 	var record bodyrecord.BodyRecord
// 	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	// Set server-side timestamp
// 	record.Timestamp = time.Now()

// 	if _, err := h.repo.AddBodyRecord(r.Context(), record); err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]: Failed to create body record: %v", err)
// 		http.Error(w, "Failed to save record", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusCreated, record)
// }

// func (h *Handlers) HandleGetBodyRecords(w http.ResponseWriter, r *http.Request) {
// 	recs, err := h.repo.GetBodyRecords(r.Context(), 1)
// 	if err != nil {
// 		log.Printf("❗️ERROR in [Handlers.go]: Failed to get body records: %v", err)
// 		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusOK, recs)
// }

// // --- Health Data Handlers ---

// func (h *Handlers) HandleCreateBodyRecord(w http.ResponseWriter, r *http.Request) {
// 	var record bodyrecord.BodyRecord
// 	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	// Set server-side timestamp
// 	record.Timestamp = time.Now()

// 	if err := h.repo.CreateBodyRecord(r.Context(), record); err != nil {
// 		log.Printf("ERROR: Failed to create body record: %v", err)
// 		http.Error(w, "Failed to save record", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusCreated, record)
// }

// func (h *Handlers) HandleGetBodyRecords(w http.ResponseWriter, r *http.Request) {
// 	records, err := h.repo.GetBodyRecords(r.Context())
// 	if err != nil {
// 		log.Printf("ERROR: Failed to get body records: %v", err)
// 		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusOK, records)
// }

// // --- UPDATED DIET HANDLERS ---

// func (h *Handlers) HandleCreateDietRecord(w http.ResponseWriter, r *http.Request) {
// 	var record bodyrecord.DietRecord
// 	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
// 		http.Error(w, "Invalid request body", http.StatusBadRequest)
// 		return
// 	}

// 	// Set server-side timestamp
// 	record.Timestamp = time.Now()

// 	if err := h.repo.CreateDietRecord(r.Context(), record); err != nil {
// 		log.Printf("ERROR: Failed to create diet record: %v", err)
// 		http.Error(w, "Failed to save record", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusCreated, record)
// }

// func (h *Handlers) HandleGetDietRecords(w http.ResponseWriter, r *http.Request) {
// 	records, err := h.repo.GetDietRecords(r.Context())
// 	if err != nil {
// 		log.Printf("ERROR: Failed to get diet records: %v", err)
// 		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
// 		return
// 	}

// 	writeJSON(w, http.StatusOK, records)
// }

// --- JSON Helper ---
