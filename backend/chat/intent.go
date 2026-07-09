package chat

import (
	"context"
	"fmt"
	"log"
	"strings"
)

// Intent is the high-level intent label returned by the LLM.
type Intent string

const (
	IntentSymptom    Intent = "symptom"
	IntentFirstAid   Intent = "first_aid"
	IntentMedication Intent = "medication"
	IntentEmergency  Intent = "emergency"
	IntentLifestyle  Intent = "lifestyle"
	IntentGeneral    Intent = "general"
	IntentUnknown    Intent = "unknown"
)

// IntentDetector uses Ollama to classify a user's message into a single Intent.
// It is LLM-driven, so you can update the prompt to refine behavior without code changes.
type IntentDetector struct {
	ollama *OllamaClient
	model  string
}

// NewIntentDetector(ollama) -> *IntentDetector
// Creates a new IntentDetector that will call Ollama using the provided OllamaClient.
func NewIntentDetector(ollama *OllamaClient) *IntentDetector {
	return &IntentDetector{
		ollama: ollama,
		model:  "llama3.2:latest",
	}
}

// Detect(ctx, message) -> (Intent, error)
// Calls Ollama to classify the message intent. Returns one of the Intent constants.
// COMMAND: Use this to get the user's intent. Example: intent, err := detector.Detect(ctx, "I have a headache")
func (d *IntentDetector) Detect(ctx context.Context, message string) (Intent, error) {

	log.Printf("[IntentDetector] Starting detection for message: %s", strings.ReplaceAll(message, "\n", " "))

	// Few-shot / instruction prompt — concise, robust.
	prompt := fmt.Sprintf(`Classify the user's intent with one of: symptom, first_aid, medication, emergency, lifestyle, general.
Return ONLY the single label on the first line, nothing else.

User message:
"""%s"""
`, message)

	out, err := d.ollama.Generate(ctx, d.model, prompt)
	if err != nil {
		return IntentUnknown, err
	}

	// Normalise & map
	l := strings.ToLower(strings.TrimSpace(out))
	switch {
	case strings.HasPrefix(l, "symptom"):
		return IntentSymptom, nil
	case strings.HasPrefix(l, "first_aid"):
		return IntentFirstAid, nil
	case strings.HasPrefix(l, "medication"):
		return IntentMedication, nil
	case strings.HasPrefix(l, "emergency"):
		return IntentEmergency, nil
	case strings.HasPrefix(l, "lifestyle"):
		return IntentLifestyle, nil
	case strings.HasPrefix(l, "general"):
		return IntentGeneral, nil
	default:
		// If the LLM returned a full sentence, attempt to find keywords
		if strings.Contains(l, "symptom") {
			return IntentSymptom, nil
		}
		if strings.Contains(l, "medication") {
			return IntentMedication, nil
		}
		return IntentUnknown, nil
	}
}
