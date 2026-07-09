package chat

import (
	"context"
	"strings"
	"sync"
	"time"
)

// MemoryMessage is a single chat turn stored in memory
type MemoryMessage struct {
	Role      string    // "user" or "assistant"
	Content   string    // message text
	Timestamp time.Time // when it was stored
}

// ChatMemory is a mutex-protected in-memory buffer of recent chat messages.
// Optionally it can call Ollama to summarize older messages (to keep memory bounded).
type ChatMemory struct {
	mu         sync.Mutex
	messages   []MemoryMessage
	limit      int           // keep at most limit messages (before summarization/trim)
	ollama     *OllamaClient // optional; if set, memory can summarize itself
	summarized bool          // whether summarization has occurred
}

// NewChatMemory(limit) -> *ChatMemory
// Create a new ChatMemory that retains `limit` messages before trimming/summarizing.
func NewChatMemory(limit int) *ChatMemory {
	return &ChatMemory{
		messages: make([]MemoryMessage, 0, limit),
		limit:    limit,
	}
}

// SetSummarizer(ollama) -> void
// Attach an OllamaClient so memory can call the LLM to summarize old messages when needed.
func (m *ChatMemory) SetSummarizer(ollama *OllamaClient) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ollama = ollama
}

// Add(role, content) -> void
// Thread-safe append of a new message. If the buffer grows beyond 'limit',
// either drop oldest messages or (if summarizer attached) compress them.
func (m *ChatMemory) Add(role, content string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.messages = append(m.messages, MemoryMessage{
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Trim or summarize if necessary
	if len(m.messages) > m.limit {
		// If LLM summarization is available, compress older entries
		if m.ollama != nil && !m.summarized {
			// spawn a goroutine to avoid blocking the request path heavily
			go m.summarizeOlder()
		} else {
			// no summarizer attached -> drop oldest
			excess := len(m.messages) - m.limit
			m.messages = m.messages[excess:]
		}
	}
}

// GetAll() -> []MemoryMessage
// Thread-safe snapshot copy of the messages.
func (m *ChatMemory) GetAll() []MemoryMessage {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]MemoryMessage, len(m.messages))
	copy(out, m.messages)
	return out
}

// Format() -> string
// Returns a formatted textual representation to inject into prompts.
func (m *ChatMemory) Format() string {
	msgs := m.GetAll()
	var sb strings.Builder
	for _, mm := range msgs {
		sb.WriteString(mm.Role)
		sb.WriteString(": ")
		sb.WriteString(mm.Content)
		sb.WriteString("\n")
	}
	return sb.String()
}

// summarizeOlder() -> (internal)
// When memory is full and an OllamaClient is present we request a short summary
// of older messages and replace them with a single summarized entry.
// This keeps memory small while preserving context.
func (m *ChatMemory) summarizeOlder() {
	m.mu.Lock()
	// quickly snapshot the items to summarize, keep the newest half
	n := len(m.messages)
	if n <= m.limit {
		m.mu.Unlock()
		return
	}
	// We'll summarize the first N/2 messages, keep the newest half
	toSummarize := make([]MemoryMessage, n/2)
	copy(toSummarize, m.messages[:n/2])
	rest := make([]MemoryMessage, n-n/2)
	copy(rest, m.messages[n/2:])
	m.mu.Unlock()

	// Build summarization prompt
	var sb strings.Builder
	sb.WriteString("Summarize the following conversation into a short context paragraph (one or two sentences). Preserve facts like allergies, important diseases, medications, and preferences.\n\n")
	for _, mm := range toSummarize {
		sb.WriteString(mm.Role + ": " + mm.Content + "\n")
	}
	prompt := sb.String()

	// Call Ollama (no context passed here)
	ctx := context.Background()
	if m.ollama == nil {
		// fallback: just drop oldest
		m.mu.Lock()
		m.messages = rest
		m.mu.Unlock()
		return
	}
	summary, err := m.ollama.Generate(ctx, "llama3.2:latest", prompt)
	if err != nil || len(summary) == 0 {
		// fallback: drop oldest
		m.mu.Lock()
		m.messages = rest
		m.mu.Unlock()
		return
	}

	// Replace older messages with a single "system" message containing the summary
	m.mu.Lock()
	defer m.mu.Unlock()
	summaryMsg := MemoryMessage{
		Role:      "system_summary",
		Content:   summary,
		Timestamp: time.Now(),
	}
	newMessages := make([]MemoryMessage, 0, len(rest)+1)
	newMessages = append(newMessages, summaryMsg)
	newMessages = append(newMessages, rest...)
	m.messages = newMessages
	m.summarized = true
}
