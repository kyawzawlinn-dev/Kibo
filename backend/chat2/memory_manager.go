package chat2

import (
	"context"
	"log"
	"sync"
)

// ConversationMemory stores last N messages per user with mutex
type ConversationMemory struct {
	mu     sync.Mutex
	msgs   []string
	limit  int
	ollama *OllamaClient // optional summarizer
}

// MemoryManager keeps per-user memories in RAM
type MemoryManager struct {
	mu    sync.Mutex
	store map[int64]*ConversationMemory
}

// globalMemory is the singleton MemoryManager
var globalMemory = &MemoryManager{store: make(map[int64]*ConversationMemory)}

// NewConversationMemory creates a new ConversationMemory with a limit and optional OllamaClient
func NewConversationMemory(limit int, ollama *OllamaClient) *ConversationMemory {
	return &ConversationMemory{msgs: make([]string, 0, limit), limit: limit, ollama: ollama}
}

func (m *MemoryManager) getOrCreate(userID int64) *ConversationMemory {
	m.mu.Lock()
	defer m.mu.Unlock()
	if cm, ok := m.store[userID]; ok {
		return cm
	}
	cm := NewConversationMemory(40, nil)
	m.store[userID] = cm
	return cm
}

func (cm *ConversationMemory) Add(role, text string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	entry := role + ": " + text
	cm.msgs = append(cm.msgs, entry)
	if len(cm.msgs) > cm.limit {
		cm.msgs = cm.msgs[len(cm.msgs)-cm.limit:]
	}
}

func (cm *ConversationMemory) Snapshot() []string {
	cm.mu.Lock()
	defer cm.mu.Unlock()
	out := make([]string, len(cm.msgs))
	copy(out, cm.msgs)
	return out
}

// Summarize calls Ollama to summarize the conversation messages
// When memory is too long, this will be used to compress older messages
func (cm *ConversationMemory) Summarize(ctx context.Context) string {
	// Optional: call ollama to summarize if configured
	if cm.ollama == nil {
		return ""
	}
	// naive summarization prompt
	prompt := "Summarize these conversation snippets into 1-2 lines:\n" + "\n" + "\n"
	cm.mu.Lock()
	batch := cm.msgs
	cm.mu.Unlock()
	for _, m := range batch {
		prompt += "- " + m + "\n"
	}

	res, err := cm.ollama.Generate(ctx, "llama3.2:latest", prompt)
	if err != nil {
		log.Println("summarize failed:", err)
		return ""
	}
	return res
}
