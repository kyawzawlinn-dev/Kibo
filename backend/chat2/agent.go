package chat2

import (
	"context"
	"fmt"
	"strings"

	"Kibo/backend/bodyrecord"
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
	repo       *bodyrecord.Repository
}

func NewChatAgent(rag *RAGService, ollama *OllamaClient, repo *bodyrecord.Repository) *ChatAgent {
	return &ChatAgent{
		rag:        rag,
		ollama:     ollama,
		classifier: NewClassifier(ollama),
		repo:       repo,
	}
}

// Answer generates the assistant reply for the latest user message in a
// chat. The caller has already saved that message to chat_history, so
// the loaded history includes it.
func (a *ChatAgent) Answer(ctx context.Context, userID, chatID int64, message string) (string, error) {
	cl := a.classifier.Classify(ctx, message)
	logger.Info("[agent.go/Answer]:\tintent=%s service=%s useRAG=%v", cl.Intent, cl.Service, cl.NeedsRAG())

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
