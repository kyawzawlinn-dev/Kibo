package chat

import (
	"context"
	"fmt"
	"log"
)

// ChatAgent is an orchestrator that:
// 1) detects intent (LLM)
// 2) detects severity (LLM)
// 3) maintains in-memory conversation memory (mutex-safe)
// 4) builds a master prompt and calls existing RAGService.Ask()
// It does NOT modify RAGService internals — it only calls Ask().
type ChatAgent struct {
	intent    *IntentDetector
	severity  *SeverityClassifier
	memoryMap map[int64]*ChatMemory
	rag       *RAGService
	ollama    *OllamaClient
}

// NewChatAgent(rag, ollama) -> *ChatAgent
// Create a new agent. Do NOT change RAGService internals.
func NewChatAgent(rag *RAGService, ollama *OllamaClient) *ChatAgent {
	return &ChatAgent{
		intent:    NewIntentDetector(ollama),
		severity:  NewSeverityClassifier(ollama),
		memoryMap: make(map[int64]*ChatMemory),
		rag:       rag,
		ollama:    ollama,
	}
}

// Get or create memory for this user
func (a *ChatAgent) getMemory(userID int64) *ChatMemory {
	mem, ok := a.memoryMap[userID]
	if !ok {
		mem = NewChatMemory(40)
		mem.SetSummarizer(a.ollama)
		a.memoryMap[userID] = mem
	}
	return mem
}

// Handle(ctx, userMsg, userID) -> (string, error)
// High-level function to process a user message.
// Steps:
//  1. Add user message to memory
//  2. Detect intent via LLM
//  3. Detect severity via LLM
//  4. Build finalPrompt (memory + intent + severity + userMsg + system instructions)
//  5. Call rag.Ask(ctx, finalPrompt, userID)
//  6. Store assistant reply to memory and return it
//
// COMMAND: Use this from your API handler instead of calling rag.Ask directly.
// Example: reply, err := agent.Handle(r.Context(), req.Message, currentUserID)
func (a *ChatAgent) Handle(ctx context.Context, userMsg string, userID int64) (string, error) {
	mem := a.getMemory(userID)

	mem.Add("user", userMsg)

	intent, _ := a.intent.Detect(ctx, userMsg)
	severity, _ := a.severity.Classify(ctx, userMsg)

	finalPrompt := a.buildMasterPrompt(mem, userMsg, intent, severity)

	answer, err := a.rag.Ask(ctx, finalPrompt, userID)
	if err != nil {
		return "", err
	}

	mem.Add("assistant", answer)
	return answer, nil
}

// buildMasterPrompt merges memory, intent, severity, and user question and returns a single large prompt
// This prompt will be passed directly into RAGService.Ask() which will then retrieve context and call Ollama.
func (a *ChatAgent) buildMasterPrompt(mem *ChatMemory, userMsg string, intent Intent, severity Severity) string {
	systemInstruction := `You are Kibo, an offline health assistant... (same as before)`

	log.Printf("[Chat_Agent] Building master prompt with INTENT=%s and SEVERITY=%s", intent, severity)
	log.Printf("[Chat_Agent] Conversation memory:\n%s and user message: %s", mem.Format(), userMsg)
	return fmt.Sprintf(`%s

CONVERSATION MEMORY:
%s

DETECTED_INTENT: %s
DETECTED_SEVERITY: %s

USER MESSAGE:
%s`, systemInstruction, mem.Format(), intent, severity, userMsg)
}
