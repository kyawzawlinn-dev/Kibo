package chat2

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"Kibo/backend/bodyrecord"
)

// RAGService orchestrates retrieval + augmentation + generation
type RAGService struct {
	repo        *bodyrecord.Repository
	ollama      *OllamaClient
	vectorStore *ChromaVectorStore
	embedModel  string
	chatModel   string
}

func NewRAGService(repo *bodyrecord.Repository, ollama *OllamaClient, vectorStore *ChromaVectorStore) *RAGService {
	return &RAGService{
		repo:        repo,
		ollama:      ollama,
		vectorStore: vectorStore,
		embedModel:  "nomic-embed-text",
		chatModel:   "llama3.2:latest",
	}
}

// Ask performs retrieval (if enabled) and generation. If useRAG is false, it will call LLM directly with agentPrompt.
func (s *RAGService) Ask(ctx context.Context, agentPrompt string, userID int64, useRAG bool, intent, severity string) (string, error) {

	// If not using RAG, directly ask Ollama with the agent prompt
	if !useRAG {
		// Directly ask Ollama with the agent prompt
		return s.ollama.Generate(ctx, s.chatModel, agentPrompt)
	}

	// Retrieve context
	personalContext := s.retrievePersonalContext(ctx, userID)
	knowledgeContext := s.retrieveKnowledgeContext(ctx, intent+" "+severity+" "+agentPrompt)

	// --- Build full system prompt ---
	systemInstruction := `
You are Kibo — a friendly, offline, multi-domain assistant.

GENERAL BEHAVIOR RULES:
- Be concise, helpful, and clear.
- Never introduce yourself again after the first message.
- Always reason step-by-step internally, but output only the final answer.
- If information is not found in context sources, say “I don’t have enough information from your knowledge base.”
- Do NOT hallucinate technical or medical facts that are not in the context.

DOMAIN HANDLING:
- HEALTH QUESTIONS:
    * Provide general advice only.
    * If medication is mentioned → give OTC guidance + mandatory safety disclaimer.
    * If symptoms are dangerous → politely recommend seeking help.
- COOKING QUESTIONS:
    * Provide steps, ingredients, variations, cooking times.
- CAR / DEVICE REPAIR:
    * Give troubleshooting steps, safety instructions, tools needed.
- DOCUMENT / TEXT QUESTIONS:
    * Explain definitions, summarize, rewrite, analyze.
- CODE QUESTIONS:
    * Provide fixes, explanations, debugging steps.

CONTEXT SOURCES:
1. PERSONAL CONTEXT (user records)
2. KNOWLEDGE BASE CONTEXT (RAG)
3. AGENT INSTRUCTIONS (intent engine output)


Merge the information in the above order when answering.
`
	full := fmt.Sprintf("%s\n---\nCONTEXT FROM USER'S PERSONAL HEALTH RECORDS:\n%s\n---\nCONTEXT FROM GENERAL HEALTH KNOWLEDGE BASE:\n%s\n---\nAGENT PROMPT:\n%s\n", systemInstruction, personalContext, knowledgeContext, agentPrompt)

	log.Printf("[rag_service] Augmented Prompt:\n\n%s\n", full)

	// call ollama
	ctx2, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	return s.ollama.Generate(ctx2, s.chatModel, full)
}

// retrievePersonalContext fetches user's personal health data.
func (s *RAGService) retrievePersonalContext(ctx context.Context, userID int64) string {
	if s.repo == nil {
		return "User has no recent health records."
	}

	brs, err := s.repo.GetBodyRecords(ctx, userID)
	if err != nil || len(brs) == 0 {
		return "User has no recent health records."
	}

	var sb strings.Builder
	sb.WriteString("Here is the user's recent health data:\n")
	for _, r := range brs {
		sb.WriteString(fmt.Sprintf(
			"- [Body] %s: %s = %.2f %s\n",
			r.Timestamp.Format("Jan 2"),
			r.RecordType,
			r.Value,
			r.Unit,
		))
	}
	return sb.String()
}

// retrieveKnowledgeContext fetches relevant documents from vector store.
func (s *RAGService) retrieveKnowledgeContext(ctx context.Context, query string) string {
	if s.vectorStore == nil {
		return "No general health information found."
	}

	docs, err := s.vectorStore.Search(ctx, query, 4)
	if err != nil || len(docs) == 0 {
		return "No general health information found."
	}

	return strings.Join(docs, "\n---\n")
}
