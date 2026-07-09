package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"Kibo/backend/bodyrecord"

	"github.com/google/uuid" // Add this import for unique IDs
)

// RAGService orchestrates the Retrieve-Augment-Generate pipeline
type RAGService struct {
	repo        *bodyrecord.Repository
	ollama      *OllamaClient
	vectorStore ChromaVectorStore // <-- CHANGED: Use the interface
	embedModel  string
	chatModel   string
}

// NewRAGService creates a new RAG service
func NewRAGService(repo *bodyrecord.Repository, ollama *OllamaClient, vectorStore ChromaVectorStore) *RAGService {
	service := &RAGService{
		repo:        repo,
		ollama:      ollama,
		vectorStore: vectorStore, // <-- CHANGED: Accept via dependency injection
		embedModel:  "nomic-embed-text",
		chatModel:   "llama3.2:latest",
	}
	// Load mock documents into the store for demonstration
	//service.loadInitialReports()

	return service
}

// for testing purpose only
// loadInitialReports simulates loading complex, dated medical reports into the vector store.
func (s *RAGService) loadInitialReports() {
	ctx := context.Background()

	reports := map[string]string{
		"Cardiology Findings": `CARDIOLOGY REPORT (Date: 2024-01-15): The patient shows mild left ventricular hypertrophy (LVH). Ejection Fraction (EF) is normal at 62%. No significant valvular dysfunction noted. Recommendation: Monitor blood pressure.`,
		"Blood Work Summary":  `LAB WORK REPORT (Date: 2024-11-10): Routine blood panel shows slightly elevated LDL cholesterol (135 mg/dL). All other markers (glucose, kidney function) are within normal range.`,
	}

	log.Println("Simulating indexing of two historical medical reports...")

	for title, text := range reports {
		// Generate unique document ID
		docID := uuid.New().String()

		// Create metadata for this document
		metadata := map[string]string{
			"title":      title,
			"source":     "historical report",
			"created_at": time.Now().Format(time.RFC3339),
		}

		// Index with metadata
		err := s.vectorStore.Index(ctx, docID, text, metadata)
		if err != nil {
			log.Printf("⚠️ Failed to index report %s: %v", title, err)
			continue
		}

		log.Printf("✅ Successfully indexed report: %s (ID: %s)", title, docID)
	}
}

// Ask is the main entry point for the chat
func (s *RAGService) Ask(ctx context.Context, question string, userID int64) (string, error) {

	// --- 1. RETRIEVE (Personal Data) ---
	personalContext, err := s.retrievePersonalContext(ctx, userID)
	if err != nil {
		log.Printf("Warning: could not retrieve personal context: %v", err)
		personalContext = "User has no recent body records."
	}

	// --- 2. RETRIEVE (Knowledge Base & Historical Reports) ---
	knowledgeContext, err := s.retrieveKnowledgeContext(ctx, question)
	if err != nil {
		log.Printf("Warning: could not retrieve knowledge context: %v", err)
		knowledgeContext = "No general health information found."
	}

	// --- 3. AUGMENT ---
	prompt := s.buildPrompt(question, personalContext, knowledgeContext)
	log.Printf("[rag_service] Augmented Prompt:\n%s\n", prompt)

	// --- 4. GENERATE ---
	response, err := s.ollama.Generate(ctx, s.chatModel, prompt)
	if err != nil {
		return "", fmt.Errorf("Ollama generation failed: %w", err)
	}

	return response, nil

}

func (s *RAGService) retrievePersonalContext(ctx context.Context, userID int64) (string, error) {
	// ... (Existing implementation remains the same)
	bodyRecords, err := s.repo.GetBodyRecords(ctx, userID)
	if err != nil {
		return "", err
	}

	dietRecords, err := s.repo.GetDietRecords(ctx, userID)
	if err != nil {
		return "", err
	}

	if len(bodyRecords) == 0 && len(dietRecords) == 0 {
		return "User has no recent health records.", nil
	}

	var sb strings.Builder
	sb.WriteString("Here is the user's recent health data:\n")

	for _, r := range bodyRecords {
		sb.WriteString(fmt.Sprintf("- [Body] %s: %s = %.2f %s\n", r.Timestamp.Format("Jan 2"), r.RecordType, r.Value, r.Unit))
	}
	for _, r := range dietRecords {
		sb.WriteString(fmt.Sprintf("- [Diet] %s: %s (%d kcal)\n", r.Timestamp.Format("Jan 2"), r.FoodName, r.Calories))
	}
	return sb.String(), nil
}

func (s *RAGService) retrieveKnowledgeContext(ctx context.Context, question string) (string, error) {
	// 1. Create embedding for the user's question
	// embeddingResponse, err := s.ollama.Embed(ctx, s.embedModel, question)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to embed question: %w", err)
	// }

	// 2. Search the vector store for the top 3 related documents
	docs, err := s.vectorStore.Search(ctx, question, 3)
	if err != nil {
		return "", fmt.Errorf("vector store search failed: %w", err)
	}

	if len(docs) == 0 {
		return "No reports relevant to this question were found in the archive.", nil
	}

	// 3. Join the documents into a single string
	return strings.Join(docs, "\n---\n"), nil
}

// buildPrompt constructs the system prompt to guide the LLM's response.
func (s *RAGService) buildPrompt(question, personalContext, knowledgeContext string) string {
	// ... (This function remains the same as your provided code)
	systemInstruction := `You are a friendly, conversational, and highly helpful AI health assistant named Kibo.
Your primary goal is to answer the user's health questions accurately and safely.
Always maintain a human-like, conversational tone and DO NOT re-introduce yourself in every message.

You MUST use the provided context to form your answer, prioritizing the detailed information in the "GENERAL HEALTH KNOWLEDGE BASE" when available.
If personal context is unavailable, answer based on general health knowledge.
Pay close attention to dates within the CONTEXT FROM GENERAL HEALTH KNOWLEDGE BASE if the user asks for historical information (e.g., "my January report").

**SPECIFIC MEDICATION RULE (EMERGENCY/SYMPTOM-BASED):**
If the user describes common, non-severe symptoms (like headache, minor fever, body aches, mild allergies), you are authorized to mention common, widely accepted, over-the-counter (OTC) pain relievers or treatments (e.g., **Acetaminophen**, **Ibuprofen**, common antihistamines) only as GENERAL advice.
**IMMEDIATE DISCLAIMER MANDATORY:** Any mention of an OTC drug MUST be followed immediately by a strong caution to read the label, follow dosage instructions strictly, check for contraindications, and seek professional medical advice if symptoms persist or worsen. DO NOT recommend prescription drugs or specific brands.

Do not make up information. If the answer is not in the context, you must state that you cannot answer the question based on the provided information.`

	return fmt.Sprintf(`
%s
---
CONTEXT FROM USER'S PERSONAL HEALTH RECORDS:
%s
---
CONTEXT FROM GENERAL HEALTH KNOWLEDGE BASE:
%s
---

USER'S QUESTION:
%s

Your answer:
`, systemInstruction, personalContext, knowledgeContext, question)
}
