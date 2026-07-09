package chat2

import (
	logger "Kibo/backend/kibo_utils"
	"context"
	"fmt"
	"strings"
)

// ChatAgent coordinates intent, severity, memory and RAG
type ChatAgent struct {
	rag      *RAGService
	ollama   *OllamaClient
	intent   *IntentDetector
	service  *ServiceClassifier
	memory   *MemoryManager
	memLimit int
}

// NewChatAgent(rag, ollama) -> *ChatAgent
func NewChatAgent(rag *RAGService, ollama *OllamaClient) *ChatAgent {
	return &ChatAgent{
		rag:      rag,
		ollama:   ollama,
		intent:   NewIntentDetector(ollama),
		service:  NewServiceClassifier(ollama),
		memory:   globalMemory,
		memLimit: 40,
	}
}

// Refine_prompt returns answer for user; orchestrates flow
// 1. detect intent and severity
// 2. get memory and add user message
// 3. build refined prompt
// 4. decide whether to use RAG
// 5. call RAGService
// 6. save assistant reply to memory
func (a *ChatAgent) Refine_prompt(ctx context.Context, userID int64, message string) (string, error) {
	// 1. detect intent and severity
	intentLabel, _ := a.intent.Detect(ctx, message)
	serviceLabel, _ := a.service.Classify(ctx, message)

	logger.Info(fmt.Sprintf("[Chat_Agent/Refine_prompt]: Building master prompt with INTENT=%s and SERVICE=%s", intentLabel, serviceLabel))

	// 2. memory
	cm := a.memory.getOrCreate(userID)
	cm.Add("user", message)
	memSnapshot := cm.Snapshot()

	logger.Info(fmt.Sprintf("[Chat_Agent/Refine_prompt]: Conversation memory:\n%s\n and user message: %s", formatMemory(memSnapshot), message))

	// 3. build agent prompt
	agentPrompt := a.buildRefinedPrompt(intentLabel, serviceLabel, memSnapshot, message)

	// 4. decide whether to use RAG
	useRAG := true
	if intentLabel == "GENERAL" {
		useRAG = false
	}

	// 5. call RAGService
	resp, err := a.rag.Ask(ctx, agentPrompt, userID, useRAG, intentLabel, serviceLabel)
	if err != nil {
		return "", fmt.Errorf("agent failed: %w", err)
	}

	// 6. save assistant reply to memory
	cm.Add("assistant", resp)

	return resp, nil
}

// buildRefinedPrompt merges memory, intent, severity, and user question and returns a single large prompt
func (a *ChatAgent) buildRefinedPrompt(intent, severity string, memory []string, message string) string {
	memText := "No previous messages."
	if len(memory) > 0 {
		memText = "- " + formatMemory(memory)
	}

	sys := `You are Kibo, an offline health assistant. Be concise, helpful, and prioritize user safety.`

	return fmt.Sprintf(`%s

DETECTED_INTENT: %s
DETECTED_SERVICE: %s

CONVERSATION_MEMORY:
%s

USER_MESSAGE:
%s

Please answer clearly.`,
		sys,
		intent,
		severity,
		memText,
		message,
	)
}

func (a *ChatAgent) GenerateTitle(ctx context.Context, userMsg string) (string, error) {
	logger.Info(fmt.Sprintf("[agent.go/GenerateTitle]:\tGenerating title for message: %s", userMsg))
	prompt := fmt.Sprintf(`
    Generate a short title (max 4 words) summarizing this conversation topic:
    "%s"
    Respond with only the title.
    `, userMsg)

	resp, err := a.ollama.Generate(ctx, "llama3.2:latest", prompt)
	if err != nil {
		logger.Error(fmt.Sprintf("[agent.go/GenerateTitle]:\terror: %v", err))
		return "", err
	}

	logger.Info(fmt.Sprintf("[agent.go/GenerateTitle]:\tgenerated title: %s", resp))
	// Take only the first line in case the model returns multiple lines
	title := strings.Split(resp, "\n")[0]
	title = strings.TrimSpace(title)
	return title, nil
}

// formatMemory converts memory slice to a single string
func formatMemory(mem []string) string {
	if len(mem) == 0 {
		return ""
	}
	out := ""
	for _, m := range mem {
		out += m + "\n"
	}
	return out
}
