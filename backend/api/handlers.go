package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/chat2"
	"Kibo/backend/emergency"
	logger "Kibo/backend/kibo_utils"
	"Kibo/backend/library"

	"github.com/gorilla/mux"
)

// defaultProfileID is the seeded "Family" profile — used when a
// request carries no profile header (first run, old clients).
const defaultProfileID = int64(2)

// profileID resolves the active profile for a request. Profiles are a
// device-trust model (a shared family laptop), not authentication: the
// frontend sends the selected profile in a header, or as a query
// parameter for plain-link downloads like the CSV export.
func profileID(r *http.Request) int64 {
	v := r.Header.Get("X-Kibo-Profile")
	if v == "" {
		v = r.URL.Query().Get("profile")
	}
	if id, err := strconv.ParseInt(v, 10, 64); err == nil && id > 0 {
		return id
	}
	return defaultProfileID
}

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
	Reply         string               `json:"reply"`
	Title         string               `json:"title,omitempty"`
	LogSuggestion *chat2.LogSuggestion `json:"log_suggestion,omitempty"`
}

// chatIDFromRequest parses the {chat_id} path variable.
func chatIDFromRequest(r *http.Request) (int64, error) {
	return strconv.ParseInt(mux.Vars(r)["chat_id"], 10, 64)
}

// --- Chat handlers ---

// CreateChat creates a new chat row and returns its id.
func (h *Handlers) CreateChat(w http.ResponseWriter, r *http.Request) {
	chatID, err := h.repo.CreateChat(r.Context(), profileID(r))
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
	chats, err := h.repo.ListChatsByUser(r.Context(), profileID(r))
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

	belongs, err := h.repo.ChatBelongsToUser(r.Context(), chatID, profileID(r))
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
	userID := profileID(r)

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

	reply, suggestion, err := h.agent.Answer(ctx, userID, chatID, body.Message)
	if err != nil {
		logger.Error("[handlers.go/SendMessage]:\tagent: %v", err)
		http.Error(w, "Failed to get AI response", http.StatusInternalServerError)
		return
	}

	// Save the assistant reply (not fatal for the client if it fails)
	if _, err := h.repo.AddChatHistory(ctx, chatID, nil, "assistant", reply); err != nil {
		logger.Error("[handlers.go/SendMessage]:\tsaving assistant reply: %v", err)
	}

	writeJSON(w, http.StatusOK, ChatResponse{Reply: reply, Title: title, LogSuggestion: suggestion})
}

// --- Profile handlers ---

type ProfileResponse struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// HandleGetProfiles lists all profiles on this device.
func (h *Handlers) HandleGetProfiles(w http.ResponseWriter, r *http.Request) {
	users, err := h.repo.ListUsers(r.Context())
	if err != nil {
		logger.Error("[handlers.go/HandleGetProfiles]:\t%v", err)
		http.Error(w, "Failed to list profiles", http.StatusInternalServerError)
		return
	}

	out := make([]ProfileResponse, 0, len(users))
	for _, u := range users {
		out = append(out, ProfileResponse{ID: u.ID, Name: u.Username})
	}
	writeJSON(w, http.StatusOK, out)
}

// HandleCreateProfile adds a new profile.
func (h *Handlers) HandleCreateProfile(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	name := strings.TrimSpace(body.Name)
	if name == "" || len(name) > 50 {
		http.Error(w, "Profile needs a name (max 50 characters)", http.StatusBadRequest)
		return
	}

	id, err := h.repo.CreateUser(r.Context(), name, "")
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE") {
			http.Error(w, "A profile with this name already exists", http.StatusConflict)
			return
		}
		logger.Error("[handlers.go/HandleCreateProfile]:\t%v", err)
		http.Error(w, "Failed to create profile", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusCreated, ProfileResponse{ID: id, Name: name})
}

// HandleDeleteProfile removes a profile and all its data. The last
// remaining profile cannot be deleted.
func (h *Handlers) HandleDeleteProfile(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid profile id", http.StatusBadRequest)
		return
	}

	count, err := h.repo.CountUsers(r.Context())
	if err != nil {
		logger.Error("[handlers.go/HandleDeleteProfile]:\t%v", err)
		http.Error(w, "Failed to check profiles", http.StatusInternalServerError)
		return
	}
	if count <= 1 {
		http.Error(w, "Cannot delete the last profile", http.StatusBadRequest)
		return
	}

	if err := h.repo.DeleteUser(r.Context(), id); err != nil {
		logger.Error("[handlers.go/HandleDeleteProfile]:\t%v", err)
		http.Error(w, "Failed to delete profile", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{"success": true, "id": id})
}

// --- CSV export/import ---

// HandleExportRecordsCSV streams all body records as a CSV download —
// the backup, migration, and spreadsheet format.
func (h *Handlers) HandleExportRecordsCSV(w http.ResponseWriter, r *http.Request) {
	records, err := h.repo.GetBodyRecords(r.Context(), profileID(r))
	if err != nil {
		logger.Error("[handlers.go/HandleExportRecordsCSV]:\t%v", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`attachment; filename="kibo-records-%s.csv"`, time.Now().Format("2006-01-02")))

	cw := csv.NewWriter(w)
	cw.Write([]string{"date", "type", "value", "unit"})
	// chronological order (the query returns newest first)
	for i := len(records) - 1; i >= 0; i-- {
		rec := records[i]
		cw.Write([]string{
			rec.Timestamp.Format("2006-01-02"),
			rec.RecordType,
			strconv.FormatFloat(rec.Value, 'f', -1, 64),
			rec.Unit,
		})
	}
	cw.Flush()
}

// importResult reports what happened to each row of an imported CSV.
type importResult struct {
	Imported          int `json:"imported"`
	SkippedDuplicates int `json:"skipped_duplicates"`
	SkippedInvalid    int `json:"skipped_invalid"`
}

// HandleImportRecordsCSV imports records from a CSV body with columns
// date,type,value,unit (any order; unit optional). Duplicate rows —
// same type, day, and value as an existing record — are skipped, so
// re-importing a backup is always safe.
func (h *Handlers) HandleImportRecordsCSV(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	reader := csv.NewReader(http.MaxBytesReader(w, r.Body, 5<<20))
	reader.FieldsPerRecord = -1

	header, err := reader.Read()
	if err != nil {
		http.Error(w, "Empty or unreadable CSV", http.StatusBadRequest)
		return
	}
	col := map[string]int{}
	for i, name := range header {
		col[strings.ToLower(strings.TrimSpace(name))] = i
	}
	if _, ok := col["record_type"]; ok {
		col["type"] = col["record_type"]
	}
	for _, required := range []string{"date", "type", "value"} {
		if _, ok := col[required]; !ok {
			http.Error(w, "CSV needs date, type, and value columns", http.StatusBadRequest)
			return
		}
	}

	// existing records index for deduplication
	existing, err := h.repo.GetBodyRecords(ctx, profileID(r))
	if err != nil {
		logger.Error("[handlers.go/HandleImportRecordsCSV]:\t%v", err)
		http.Error(w, "Failed to load records", http.StatusInternalServerError)
		return
	}
	seen := map[string]bool{}
	recordKey := func(recType string, t time.Time, value float64) string {
		return fmt.Sprintf("%s|%s|%.4f", recType, t.Format("2006-01-02"), value)
	}
	for _, rec := range existing {
		seen[recordKey(rec.RecordType, rec.Timestamp, rec.Value)] = true
	}

	var result importResult
	field := func(row []string, name string) string {
		if i, ok := col[name]; ok && i < len(row) {
			return strings.TrimSpace(row[i])
		}
		return ""
	}

	for {
		row, err := reader.Read()
		if err != nil {
			break
		}

		recType := normalizeRecordType(field(row, "type"))
		defUnit, known := bodyrecord.DefaultUnits[recType]
		value, verr := strconv.ParseFloat(field(row, "value"), 64)
		when, derr := parseImportDate(field(row, "date"))
		if !known || verr != nil || derr != nil || value <= 0 || value > 100000 {
			result.SkippedInvalid++
			continue
		}

		if seen[recordKey(recType, when, value)] {
			result.SkippedDuplicates++
			continue
		}

		unit := field(row, "unit")
		if unit == "" {
			unit = defUnit
		}

		if _, err := h.repo.AddBodyRecord(ctx, bodyrecord.BodyRecord{
			UserID:     profileID(r),
			RecordType: recType,
			Value:      value,
			Unit:       unit,
			Timestamp:  when,
		}); err != nil {
			logger.Error("[handlers.go/HandleImportRecordsCSV]:\tsaving row: %v", err)
			result.SkippedInvalid++
			continue
		}
		seen[recordKey(recType, when, value)] = true
		result.Imported++
	}

	writeJSON(w, http.StatusOK, result)
}

// normalizeRecordType maps "weight"/"WEIGHT" to "Weight".
func normalizeRecordType(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// parseImportDate accepts plain dates and RFC3339 timestamps; plain
// dates land at noon local so the calendar day is timezone-stable.
func parseImportDate(s string) (time.Time, error) {
	if t, err := time.ParseInLocation("2006-01-02", s, time.Local); err == nil {
		return t.Add(12 * time.Hour), nil
	}
	return time.Parse(time.RFC3339, s)
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

// --- Health log handlers ---

func (h *Handlers) HandleGetHealthLog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.repo.ListHealthLog(r.Context(), profileID(r))
	if err != nil {
		logger.Error("[handlers.go/HandleGetHealthLog]:\t%v", err)
		http.Error(w, "Failed to load health log", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

// decodeHealthLogEntry reads and validates an episode from the body.
func decodeHealthLogEntry(r *http.Request) (bodyrecord.HealthLogEntry, error) {
	var e bodyrecord.HealthLogEntry
	if err := json.NewDecoder(r.Body).Decode(&e); err != nil {
		return e, err
	}
	e.UserID = profileID(r)
	e.Title = strings.TrimSpace(e.Title)
	e.Date = strings.TrimSpace(e.Date)
	if e.Title == "" || e.Date == "" {
		return e, errors.New("episode needs a date and a description")
	}
	if _, err := time.Parse("2006-01-02", e.Date); err != nil {
		return e, errors.New("invalid date")
	}
	return e, nil
}

func (h *Handlers) HandleAddHealthLogEntry(w http.ResponseWriter, r *http.Request) {
	entry, err := decodeHealthLogEntry(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	id, err := h.repo.AddHealthLogEntry(r.Context(), entry)
	if err != nil {
		logger.Error("[handlers.go/HandleAddHealthLogEntry]:\t%v", err)
		http.Error(w, "Failed to save episode", http.StatusInternalServerError)
		return
	}
	entry.ID = id
	writeJSON(w, http.StatusCreated, entry)
}

func (h *Handlers) HandleUpdateHealthLogEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	entry, err := decodeHealthLogEntry(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	entry.ID = id

	if err := h.repo.UpdateHealthLogEntry(r.Context(), entry); err != nil {
		logger.Error("[handlers.go/HandleUpdateHealthLogEntry]:\t%v", err)
		http.Error(w, "Failed to update episode", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, entry)
}

func (h *Handlers) HandleDeleteHealthLogEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid id", http.StatusBadRequest)
		return
	}
	if err := h.repo.DeleteHealthLogEntry(r.Context(), profileID(r), id); err != nil {
		logger.Error("[handlers.go/HandleDeleteHealthLogEntry]:\t%v", err)
		http.Error(w, "Failed to delete episode", http.StatusInternalServerError)
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
	record.UserID = profileID(r)
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
	records, err := h.repo.GetBodyRecords(r.Context(), profileID(r))
	if err != nil {
		logger.Error("[handlers.go/HandleGetBodyRecords]:\t%v", err)
		http.Error(w, "Failed to retrieve records", http.StatusInternalServerError)
		return
	}

	writeJSON(w, http.StatusOK, records)
}

// HandleSaveDayRecords saves a whole day's record sheet: each metric
// with a value is upserted (one row per metric per day), and each
// metric explicitly cleared (value null) is deleted. Only known metric
// types are accepted; the date cannot be in the future.
func (h *Handlers) HandleSaveDayRecords(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Date    string              `json:"date"` // YYYY-MM-DD
		Metrics map[string]*float64 `json:"metrics"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	day, err := time.ParseInLocation("2006-01-02", body.Date, time.Local)
	if err != nil {
		http.Error(w, "Invalid date", http.StatusBadRequest)
		return
	}
	if day.After(time.Now()) {
		http.Error(w, "Cannot record a future date", http.StatusBadRequest)
		return
	}
	// noon local keeps the record on the chosen calendar day regardless
	// of timezone
	timestamp := day.Add(12 * time.Hour)

	userID := profileID(r)
	for recType, value := range body.Metrics {
		unit, known := bodyrecord.DefaultUnits[recType]
		if !known {
			continue
		}

		if value == nil {
			// cleared field → remove any existing entry for the day
			if err := h.repo.DeleteBodyRecordForDay(r.Context(), userID, recType, body.Date); err != nil {
				logger.Error("[handlers.go/HandleSaveDayRecords]:\tdelete %s: %v", recType, err)
				http.Error(w, "Failed to save records", http.StatusInternalServerError)
				return
			}
			continue
		}
		if *value <= 0 || *value > 100000 {
			continue // ignore nonsensical values
		}

		if err := h.repo.UpsertBodyRecordForDay(r.Context(), bodyrecord.BodyRecord{
			UserID:     userID,
			RecordType: recType,
			Value:      *value,
			Unit:       unit,
			Timestamp:  timestamp,
		}); err != nil {
			logger.Error("[handlers.go/HandleSaveDayRecords]:\tupsert %s: %v", recType, err)
			http.Error(w, "Failed to save records", http.StatusInternalServerError)
			return
		}
	}

	records, err := h.repo.GetBodyRecords(r.Context(), userID)
	if err != nil {
		logger.Error("[handlers.go/HandleSaveDayRecords]:\t%v", err)
		http.Error(w, "Failed to reload records", http.StatusInternalServerError)
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

	record.UserID = profileID(r)
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
	records, err := h.repo.GetDietRecords(r.Context(), profileID(r))
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
