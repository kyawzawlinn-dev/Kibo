package chat2

import (
	"context"
	"fmt"
	"strings"
	"time"

	"Kibo/backend/bodyrecord"
	logger "Kibo/backend/kibo_utils"
)

// generateTimeout bounds a single LLM reply; local models on weak
// hardware can be slow, so this is generous.
const generateTimeout = 45 * time.Second

// maxPersonalRecords caps how many of the user's health records go
// into the prompt (they arrive newest first).
const maxPersonalRecords = 10

const systemInstruction = `You are Kibo — a friendly, offline personal health companion.

RULES:
- Be concise, helpful, and clear.
- Never introduce yourself again after the first message.
- Reason step-by-step internally, but output only the final answer.
- Ground health facts in the provided context. If the context does not
  cover the question, say you don't have enough information from the
  knowledge base — do NOT invent medical facts, dosages, or diagnoses.
- If medication is discussed, add a short safety disclaimer.
- If symptoms sound dangerous, recommend seeking professional care.
- Use the user's personal health records when they are relevant.`

// RAGService orchestrates retrieval + augmentation + generation.
type RAGService struct {
	repo        *bodyrecord.Repository
	ollama      *OllamaClient
	vectorStore *VectorStore
}

func NewRAGService(repo *bodyrecord.Repository, ollama *OllamaClient, vectorStore *VectorStore) *RAGService {
	return &RAGService{
		repo:        repo,
		ollama:      ollama,
		vectorStore: vectorStore,
	}
}

// Ask generates a reply for the agent prompt. When useRAG is set it
// augments the prompt with the user's records and knowledge base
// passages; otherwise it calls the LLM directly.
func (s *RAGService) Ask(ctx context.Context, agentPrompt string, userID int64, useRAG bool, intent, service string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()

	if !useRAG {
		full := fmt.Sprintf("%s\n---\n%s", systemInstruction, agentPrompt)
		return s.ollama.Generate(ctx, ChatModel, full)
	}

	personalContext := s.retrievePersonalContext(ctx, userID)
	knowledgeContext := s.retrieveKnowledgeContext(ctx, intent+" "+service+" "+agentPrompt)

	full := fmt.Sprintf(`%s
---
USER'S PERSONAL HEALTH RECORDS:
%s
---
HEALTH KNOWLEDGE BASE PASSAGES:
%s
---
%s`,
		systemInstruction, personalContext, knowledgeContext, agentPrompt)

	logger.Debug("[rag_service.go/Ask]:\taugmented prompt:\n%s", full)

	return s.ollama.Generate(ctx, ChatModel, full)
}

// retrievePersonalContext summarizes the user's recent health records.
func (s *RAGService) retrievePersonalContext(ctx context.Context, userID int64) string {
	if s.repo == nil {
		return "No recent health records."
	}

	records, err := s.repo.GetBodyRecords(ctx, userID)
	if err != nil || len(records) == 0 {
		return "No recent health records."
	}
	if len(records) > maxPersonalRecords {
		records = records[:maxPersonalRecords] // newest first from the query
	}

	var sb strings.Builder
	for _, r := range records {
		fmt.Fprintf(&sb, "- %s: %s = %.2f %s\n",
			r.Timestamp.Format("Jan 2"), r.RecordType, r.Value, r.Unit)
	}
	return sb.String()
}

// retrieveKnowledgeContext fetches relevant knowledge base passages.
func (s *RAGService) retrieveKnowledgeContext(ctx context.Context, query string) string {
	if s.vectorStore == nil {
		return "No relevant passages found."
	}

	docs, err := s.vectorStore.Search(ctx, query, 4)
	if err != nil || len(docs) == 0 {
		return "No relevant passages found."
	}
	return strings.Join(docs, "\n---\n")
}
