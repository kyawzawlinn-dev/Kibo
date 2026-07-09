package chat2

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// IntentDetector uses the LLM to classify high-level intent
type IntentDetector struct {
	ollama *OllamaClient
	model  string
}

// NewIntentDetector(ollama) -> *IntentDetector
func NewIntentDetector(ollama *OllamaClient) *IntentDetector {
	return &IntentDetector{
		ollama: ollama,
		model:  "llama3.2:latest",
	}
}

func (d *IntentDetector) Detect(ctx context.Context, message string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	system := `
Classify the user's message into ONE intent from this list:

- DEFINE_TERM
- HEALTH_SYMPTOM
- HEALTH_MEDICATION
- RECIPE_COOKING
- CAR_REPAIR
- DEVICE_FIX
- DOCUMENT_ANALYSIS
- CODE_DEBUG
- GENERAL_ADVICE
- UNKNOWN

Rules:
- Return ONLY the label.
- Do NOT add explanation.
`

	prompt := fmt.Sprintf("%s\nUser: %s\nLabel:", system, message)
	reply, err := d.ollama.Generate(ctx, d.model, prompt)
	if err != nil {
		return "UNKNOWN", err
	}

	out := strings.ToUpper(strings.TrimSpace(reply))
	valid := map[string]bool{
		"DEFINE_TERM": true, "HEALTH_SYMPTOM": true, "HEALTH_MEDICATION": true,
		"RECIPE_COOKING": true, "CAR_REPAIR": true, "DEVICE_FIX": true,
		"DOCUMENT_ANALYSIS": true, "CODE_DEBUG": true, "GENERAL_ADVICE": true,
		"UNKNOWN": true,
	}

	if !valid[out] {
		return "UNKNOWN", nil
	}

	return out, nil
}
