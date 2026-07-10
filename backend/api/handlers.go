package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	"Kibo/backend/emergency"
	logger "Kibo/backend/kibo_utils"
	"Kibo/backend/library"

	"github.com/gorilla/mux"
)

// demoUserID stands in for authentication until user accounts exist.
const demoUserID = int64(2)

// Handlers holds dependencies for HTTP handlers
type Handlers struct {
	repo    *bodyrecord.Repository
	agent   *chat2.ChatAgent
	library *library.Library
}

// NewHandlers creates a new Handlers struct
func NewHandlers(repo *bodyrecord.Repository, agent *chat2.ChatAgent, lib *library.Library) *Handlers {
	return &Handlers{
		repo:    repo,
		agent:   agent,
		library: lib,
	}
}

// --- Request/response shapes ---

type CreateChatResponse struct {
	ChatID int64  `json:"chat_id"`
	Title  string `json:"title"`
}

type GetChatResponse struct {
	ChatID   int64                    `json:"chat_id"`
	Title    string                   `json:"title"`
	Messages []bodyrecord.ChatHistory `json:"messages"`
}

type ChatResponse struct {
	Reply string `json:"reply"`
	Title string `json:"title,omitempty"`
}

// chatIDFromRequest parses the {chat_id} path variable.
func chatIDFromRequest(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["chat_id"], 10, 64)
}

// --- Chat handlers ---

// CreateChat creates a new chat row and returns its id.
func (h *Handlers) CreateChat(w http.ResponseWriter, r *http.Request) {
	chatID, err := h.repo.CreateChat(r.Context(), demoUserID)
	if err != nil {
		logger.Error("[handlers.go/CreateChat]:\t%v", err)
		http.Error(w, "Failed to create chat", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, CreateChatResponse{ChatID: chatID, Title: "New Chat"})
}

// GetChat returns chat metadata and messages.
func (h *Handlers) GetChat(w http.ResponseWriter, r *http.Request) {
	chatID, err := chatIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	title, err := h.repo.RetrieveChatTitle(chatID)
	if err != nil {
		logger.Error("[handlers.go/GetChat]:\tRetrieveChatTitle: %v", err)
		http.Error(w, "Chat not found", http.StatusNotFound)
		return
	}

	msgs, err := h.repo.GetChatHistory(r.Context(), chatID, 1000)
	if err != nil {
		logger.Error("[handlers.go/GetChat]:\tGetChatHistory: %v", err)
		http.Error(w, "Failed to load messages", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, GetChatResponse{ChatID: chatID, Title: title, Messages: msgs})
}

// ListChats returns all chats for the user.
func (h *Handlers) ListChats(w http.ResponseWriter, r *http.Request) {
	chats, err := h.repo.ListChatsByUser(r.Context(), demoUserID)
	if err != nil {
		logger.Error("[handlers.go/ListChats]:\t%v", err)
		http.Error(w, "Failed to list chats", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, chats)
}

// DeleteChat deletes a chat and all its messages.
func (h *Handlers) DeleteChat(w http.ResponseWriter, r *http.Request) {
	chatID, err := chatIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid chat id", http.StatusBadRequest)
		return
	}

	belongs, err := h.repo.ChatBelongsToUser(r.Context(), chatID, demoUserID)
	if err != nil {
		logger.Error("[handlers.go/DeleteChat]:\tChatBelongsToUser: %v", err)
		http.Error(w, "Failed to verify ownership", http.StatusInternalServerError)
		return
	}
	if !belongs {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	if err := h.repo.DeleteChat(r.Context(), chatID); err != nil {
		logger.Error("[handlers.go/DeleteChat]:\t%v", err)
		http.Error(w, "Failed to delete chat", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "chat_id": chatID})
}

// SendMessage saves the user's message, generates the AI reply, saves
// it, and returns it (plus a generated title on the first message).
func (h *Handlers) SendMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	chatID, err := chatIDFromRequest(r)
	if err != nil {
		http.Error(w, "Invalid chat ID", http.StatusBadRequest)
		return
	}

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

	ctx := r.Context()
	userID := demoUserID

	// Save the user message first — the agent reads conversation
	// context from chat_history, so this must land before Answer.
	if _, err := h.repo.AddChatHistory(ctx, chatID, &userID, "user", body.Message); err != nil {
		logger.Error("[handlers.go/SendMessage]:\tsaving user message: %v", err)
		http.Error(w, "Failed to save message", http.StatusInternalServerError)
		return
	}

	// Generate a title on the first message (best effort). For an
	// emergency, use the card title directly — an LLM call would delay
	// the instant first-aid reply by seconds.
	var title string
	if isFirst, _ := h.repo.IsFirstMessage(chatID); isFirst {
		if card := emergency.Match(body.Message); card != nil {
			title = card.Title
		} else if title, err = h.agent.GenerateTitle(ctx, body.Message); err != nil {
			title = ""
		}
		if title != "" {
			if err := h.repo.UpdateChatTitle(chatID, title); err != nil {
				logger.Warn("[handlers.go/SendMessage]:\tupdating title: %v", err)
			}
		}
	} else {
		title, _ = h.repo.RetrieveChatTitle(chatID)
	}

	reply, err := h.agent.Answer(ctx, userID, chatID, body.Message)
	if err != nil {
		logger.Error("[handlers.go/SendMessage]:\tagent: %v", err)
		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
		return
	}

	// Save the assistant reply (not fatal for the client if it fails)
	if _, err := h.repo.AddChatHistory(ctx, chatID, nil, "assistant", reply); err != nil {
		logger.Error("[handlers.go/SendMessage]:\tsaving assistant reply: %v", err)
	}

	writeJSON(w, http.StatusOK, ChatResponse{Reply: reply, Title: title})
}

// --- Share handlers ---

// serverPort must match the address the server listens on (main.go).
const serverPort = 8080

// HandleGetShareInfo returns the LAN URLs other devices on the same
// Wi-Fi can use to reach this Kibo instance.
func (h *Handlers) HandleGetShareInfo(w http.ResponseWriter, r *http.Request) {
	var urls []string

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		logger.Error("[handlers.go/HandleGetShareInfo]:\t%v", err)
		http.Error(w, "Failed to read network interfaces", http.StatusInternalServerError)
		return
	}

	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || !ip.IsPrivate() {
			continue
		}
		urls = append(urls, fmt.Sprintf("http://%s:%d", ip, serverPort))
	}

	writeJSON(w, http.StatusOK, map[string]any{"urls": urls})
}

// --- Library handlers ---

// HandleGetLibrary returns all health library articles.
func (h *Handlers) HandleGetLibrary(w http.ResponseWriter, r *http.Request) {
	articles, err := h.library.List()
	if err != nil {
		logger.Error("[handlers.go/HandleGetLibrary]:\t%v", err)
		http.Error(w, "Failed to load library", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, articles)
}

// HandleAddLibraryArticle saves a new article and indexes it live.
func (h *Handlers) HandleAddLibraryArticle(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Title   string `json:"title"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	article, err := h.library.Add(r.Context(), body.Title, body.Content)
	switch {
	case errors.Is(err, library.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, library.ErrExists):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case err != nil:
		logger.Error("[handlers.go/HandleAddLibraryArticle]:\t%v", err)
		http.Error(w, "Failed to save article", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, article)
}

// HandleUpdateLibraryArticle replaces an article's content and reindexes it.
func (h *Handlers) HandleUpdateLibraryArticle(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	var body struct {
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	article, err := h.library.Update(r.Context(), id, body.Content)
	switch {
	case errors.Is(err, library.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, library.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		logger.Error("[handlers.go/HandleUpdateLibraryArticle]:\t%v", err)
		http.Error(w, "Failed to update article", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, article)
}

// HandleDeleteLibraryArticle removes an article and its indexed chunks.
func (h *Handlers) HandleDeleteLibraryArticle(w http.ResponseWriter, r *http.Request) {
	id := mux.Vars(r)["id"]

	err := h.library.Delete(r.Context(), id)
	switch {
	case errors.Is(err, library.ErrInvalid):
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	case errors.Is(err, library.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		logger.Error("[handlers.go/HandleDeleteLibraryArticle]:\t%v", err)
		http.Error(w, "Failed to delete article", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "id": id})
}

// --- Emergency handlers ---

// HandleGetEmergencyCards returns the embedded first-aid cards.
func (h *Handlers) HandleGetEmergencyCards(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, emergency.All())
}

// --- Health record handlers ---

func (h *Handlers) HandleCreateBodyRecord(w http.ResponseWriter, r *http.Request) {
	var record bodyrecord.BodyRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Keep a client-supplied timestamp so records can be logged for
	// past dates; stamp "now" only when none was given.
	record.UserID = demoUserID
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	id, err := h.repo.AddBodyRecord(r.Context(), record)
	if err != nil {
		logger.Error("[handlers.go/HandleCreateBodyRecord]:\t%v", err)
		http.Error(w, "Failed to save record", http.StatusInternalServerError)
		return
	}
	record.ID = id

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handlers) HandleGetBodyRecords(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetBodyRecords(r.Context(), demoUserID)
	if err != nil {
		logger.Error("[handlers.go/HandleGetBodyRecords]:\t%v", err)
		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

func (h *Handlers) HandleCreateDietRecord(w http.ResponseWriter, r *http.Request) {
	var record bodyrecord.DietRecord
	if err := json.NewDecoder(r.Body).Decode(&record); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	record.UserID = demoUserID
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}

	id, err := h.repo.AddDietRecord(r.Context(), record)
	if err != nil {
		logger.Error("[handlers.go/HandleCreateDietRecord]:\t%v", err)
		http.Error(w, "Failed to save record", http.StatusInternalServerError)
		return
	}
	record.ID = id

	writeJSON(w, http.StatusCreated, record)
}

func (h *Handlers) HandleGetDietRecords(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetDietRecords(r.Context(), demoUserID)
	if err != nil {
		logger.Error("[handlers.go/HandleGetDietRecords]:\t%v", err)
		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// --- Helpers ---

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		logger.Error("[handlers.go/writeJSON]:\t%v", err)
	}
}
