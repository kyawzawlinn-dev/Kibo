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
- Take any symptom the user reports at face value. Never dispute or
  explain away a symptom the user tells you they have.
- The personal health records (weight, sleep, activity, water) are
  background only. Use them only when directly relevant to the
  question, and never to reason about an unrelated symptom.
- NEVER refuse to help or say "I can't provide medical advice." You are
  an offline companion for people who may not be able to reach a clinic.
  If professional care is not immediately reachable, still give safe,
  practical interim steps — rest, safe positioning, hydration, what to
  avoid, and what warning signs to watch for — while strongly urging
  them to reach medical help as soon as they can.`

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
// passages retrieved for `query` (the raw user message — searching on
// the full prompt with history dilutes retrieval badly).
func (s *RAGService) Ask(ctx context.Context, agentPrompt, query string, userID int64, useRAG bool) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, generateTimeout)
	defer cancel()

	if !useRAG {
		full := fmt.Sprintf("%s\n---\n%s", systemInstruction, agentPrompt)
		return s.ollama.Generate(ctx, ChatModel, full)
	}

	personalContext := s.retrievePersonalContext(ctx, userID)
	knowledgeContext, sources := s.retrieveKnowledgeContext(ctx, query)

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

	reply, err := s.ollama.Generate(ctx, ChatModel, full)
	if err != nil {
		return "", err
	}

	// Cite deterministically: the sources listed are exactly the
	// passages that were retrieved — never up to the LLM.
	if len(sources) > 0 {
		reply += "\n\n📚 Sources: " + strings.Join(sources, ", ")
	}
	return reply, nil
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

// retrieveKnowledgeContext fetches relevant knowledge base passages
// and the deduplicated source names they came from.
func (s *RAGService) retrieveKnowledgeContext(ctx context.Context, query string) (string, []string) {
	if s.vectorStore == nil {
		return "No relevant passages found.", nil
	}

	docs, err := s.vectorStore.Search(ctx, query, 4)
	if err != nil || len(docs) == 0 {
		return "No relevant passages found.", nil
	}

	var passages []string
	var sources []string
	seen := map[string]bool{}
	for _, d := range docs {
		name := sourceName(d.Source)
		passages = append(passages, fmt.Sprintf("[from: %s]\n%s", name, d.Content))
		if name != "" && !seen[name] {
			seen[name] = true
			sources = append(sources, name)
		}
	}
	return strings.Join(passages, "\n---\n"), sources
}

// sourceName turns "symptoms_fever.md" into "symptoms_fever".
func sourceName(file string) string {
	return strings.TrimSuffix(strings.TrimSpace(file), ".md")
}
