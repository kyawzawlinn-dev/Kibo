package chat2

import (
	"context"
	"fmt"
	"strings"

	"Kibo/backend/bodyrecord"
	"Kibo/backend/emergency"
	logger "Kibo/backend/kibo_utils"
)

// historyWindow is how many recent messages form the conversation
// context sent to the LLM.
const historyWindow = 20

// ChatAgent answers user messages: it classifies the message, loads the
// recent conversation from the database, and routes through RAG when
// the knowledge base can help.
//
// Conversation memory is the chat_history table itself — per chat by
// construction and it survives restarts. No in-RAM state to sync.
type ChatAgent struct {
	rag        *RAGService
	ollama     *OllamaClient
	classifier *Classifier
	logger     *RecordLogger
	repo       *bodyrecord.Repository
}

func NewChatAgent(rag *RAGService, ollama *OllamaClient, repo *bodyrecord.Repository) *ChatAgent {
	return &ChatAgent{
		rag:        rag,
		ollama:     ollama,
		classifier: NewClassifier(ollama),
		logger:     NewRecordLogger(ollama, repo),
		repo:       repo,
	}
}

// Answer generates the assistant reply for the latest user message in a
// chat. The caller has already saved that message to chat_history, so
// the loaded history includes it.
func (a *ChatAgent) Answer(ctx context.Context, userID, chatID int64, message string) (string, error) {
	// Red-flag messages get the first-aid card immediately — before
	// any LLM call. In an emergency nobody waits for token generation.
	if card := emergency.Match(message); card != nil {
		logger.Info("[agent.go/Answer]:\temergency card matched: %s", card.ID)
		return formatEmergencyReply(card), nil
	}

	cl := a.classifier.Classify(ctx, message)
	logger.Info("[agent.go/Answer]:\tintent=%s service=%s useRAG=%v", cl.Intent, cl.Service, cl.NeedsRAG())

	// "I weighed 68kg today" -> save a record and confirm. On
	// extraction failure, reply deterministically instead of falling
	// through to the LLM — a conversational reply here tends to
	// *pretend* the record was saved, which is worse than asking the
	// user to rephrase.
	if cl.Intent == "LOG_RECORD" {
		if reply, ok := a.logger.TryLog(ctx, userID, message); ok {
			return reply, nil
		}
		return `I couldn't read a measurement from that. Try something like: "weight 68.5 kg", "slept 7 hours", or "drank 2L of water yesterday".`, nil
	}

	history, err := a.repo.GetRecentChatHistory(ctx, chatID, historyWindow)
	if err != nil {
		logger.Warn("[agent.go/Answer]:\tloading chat history: %v", err)
		// degrade gracefully: answer without conversation context
		history = nil
	}

	prompt := buildPrompt(cl, history, message)

	resp, err := a.rag.Ask(ctx, prompt, userID, cl.NeedsRAG(), cl.Intent, cl.Service)
	if err != nil {
		return "", fmt.Errorf("agent failed: %w", err)
	}
	return resp, nil
}

// formatEmergencyReply renders a first-aid card as a chat message.
// Deterministic — the steps shown are exactly the curated card.
func formatEmergencyReply(card *emergency.Card) string {
	return fmt.Sprintf(
		"🚨 %s\n\n%s\n\n⚠️ This guidance does not replace professional care — get medical help as soon as possible. All first-aid cards are on the Emergency page, fully offline.",
		card.Title, card.Body,
	)
}

// buildPrompt merges the classification, conversation history, and the
// current question into the prompt the RAG service augments.
func buildPrompt(cl Classification, history []bodyrecord.ChatHistory, message string) string {
	var conv strings.Builder
	for _, m := range history {
		conv.WriteString(m.Role)
		conv.WriteString(": ")
		conv.WriteString(m.Message)
		conv.WriteString("\n")
	}
	if conv.Len() == 0 {
		// history unavailable — fall back to just the current message
		conv.WriteString("user: ")
		conv.WriteString(message)
		conv.WriteString("\n")
	}

	return fmt.Sprintf(`DETECTED_INTENT: %s
DETECTED_SERVICE: %s

CONVERSATION (oldest first; answer the last user message):
%s`,
		cl.Intent,
		cl.Service,
		conv.String(),
	)
}

// GenerateTitle produces a short chat title from the first message.
func (a *ChatAgent) GenerateTitle(ctx context.Context, userMsg string) (string, error) {
	prompt := fmt.Sprintf(
		"Generate a short title (max 4 words) summarizing this conversation topic:\n%q\nRespond with only the title, no quotes.",
		userMsg,
	)

	resp, err := a.ollama.Generate(ctx, ChatModel, prompt)
	if err != nil {
		logger.Error("[agent.go/GenerateTitle]:\terror: %v", err)
		return "", err
	}

	// Take only the first line and strip wrapping quotes the model
	// tends to add despite instructions
	title := strings.TrimSpace(strings.Split(resp, "\n")[0])
	title = strings.Trim(title, `"'`)
	logger.Info("[agent.go/GenerateTitle]:\tgenerated title: %s", title)
	return title, nil
}
