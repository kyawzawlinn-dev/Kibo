package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Severity labels whether the case is mild/moderate/severe
type Severity string

const (
	SeverityMild     Severity = "mild"
	SeverityModerate Severity = "moderate"
	SeveritySevere   Severity = "severe"
)

// SeverityClassifier uses Ollama to estimate severity from natural text.
type SeverityClassifier struct {
	ollama *OllamaClient
	model  string
}

// NewSeverityClassifier(ollama) -> *SeverityClassifier
// Creates a severity classifier that uses the provided OllamaClient.
func NewSeverityClassifier(ollama *OllamaClient) *SeverityClassifier {
	return &SeverityClassifier{
		ollama: ollama,
		model:  "llama3.2:latest",
	}
}

// Classify(ctx, message) -> (Severity, error)
// Ask the LLM to return mild/moderate/severe only. Use this before deciding whether to add emergency escalation.
// COMMAND: Use to decide how risky the user's message is. Example: sev, err := sc.Classify(ctx, "I have chest pain and can't breathe")
func (s *SeverityClassifier) Classify(ctx context.Context, message string) (Severity, error) {

	log.Printf("[Classifier] Starting classification for message: %s", strings.ReplaceAll(message, "\n", " "))

	prompt := fmt.Sprintf(`
Assess severity and return ONE word: mild, moderate, or severe.
Consider seriousness of symptoms, urgency, and risk of harm.

User message:
"""%s"""
`, message)

	out, err := s.ollama.Generate(ctx, s.model, prompt)
	if err != nil {
		return SeverityModerate, err
	}

	l := strings.ToLower(strings.TrimSpace(out))
	switch {
	case strings.HasPrefix(l, "mild"):
		return SeverityMild, nil
	case strings.HasPrefix(l, "moderate"):
		return SeverityModerate, nil
	case strings.HasPrefix(l, "severe"):
		return SeveritySevere, nil
	default:
		// fallback heuristics
		if strings.Contains(l, "severe") || strings.Contains(l, "emergency") {
			return SeveritySevere, nil
		}
		return SeverityModerate, nil
	}
}
