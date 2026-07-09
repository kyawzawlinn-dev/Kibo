package chat2

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SeverityClassifier uses Ollama to estimate severity from natural text.
type ServiceClassifier struct {
	ollama *OllamaClient
	model  string
}

func NewServiceClassifier(ollama *OllamaClient) *ServiceClassifier {
	return &ServiceClassifier{ollama: ollama, model: "llama3.2:latest"}
}

func (s *ServiceClassifier) Classify(ctx context.Context, message string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()

	system := `
Classify the user's message into ONE service type:

- EXPLAIN
- INSTRUCT
- TROUBLESHOOT
- SUGGEST
- CALCULATE
- SUMMARIZE
- ANALYZE
- WARN

Return ONLY the label.
`

	prompt := fmt.Sprintf("%s\nMessage: %s\nService:", system, message)
	reply, err := s.ollama.Generate(ctx, s.model, prompt)
	if err != nil {
		return "EXPLAIN", err
	}

	out := strings.ToUpper(strings.TrimSpace(reply))
	valid := map[string]bool{
		"EXPLAIN": true, "INSTRUCT": true, "TROUBLESHOOT": true,
		"SUGGEST": true, "CALCULATE": true, "SUMMARIZE": true,
		"ANALYZE": true, "WARN": true,
	}

	if !valid[out] {
		return "EXPLAIN", nil
	}

	return out, nil
}
