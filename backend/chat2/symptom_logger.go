package chat2

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	logger "Kibo/backend/kibo_utils"
)

// ownershipCue matches phrases that claim someone actually has a
// symptom ("I have", "my head hurts", "my son is coughing"). Requiring
// one before suggesting a log entry keeps pure questions ("is a fever
// dangerous?") from producing false suggestions — biasing toward
// silence, as an over-eager prompt is worse than a missed offer.
var ownershipCue = regexp.MustCompile(`(?i)\b(i have|i've|i am|i'm|i feel|i felt|i got|i had|i keep|my|he has|she has|they have|we have|having a|got a)\b`)

// LogSuggestion is an offer to add a symptom to the health log,
// extracted from a chat message. It is never saved automatically — the
// user confirms it in the UI.
type LogSuggestion struct {
	Date     string `json:"date"` // YYYY-MM-DD
	Title    string `json:"title"`
	Severity string `json:"severity"` // "", mild, moderate, severe
}

// SymptomLogger detects when a chat message states a real symptom
// worth logging, versus merely asking about one.
type SymptomLogger struct {
	ollama *OllamaClient
}

func NewSymptomLogger(ollama *OllamaClient) *SymptomLogger {
	return &SymptomLogger{ollama: ollama}
}

const symptomPrompt = `A user wrote a message in a health chat. Decide whether they are STATING a symptom or illness they (or a family member) actually have now or had recently — as opposed to only ASKING about one, speaking hypothetically, or discussing it in general.

Reply with ONLY JSON:
- To log:  {"symptom":"Headache","severity":"mild","days_ago":0}
- Nothing: {}

severity is one of "", "mild", "moderate", "severe". days_ago is 0 for today/this morning, 1 for yesterday, etc.

Examples:
"I have a bad headache" -> {"symptom":"Headache","severity":"severe","days_ago":0}
"what should I take for a headache?" -> {}
"I've had a fever since yesterday" -> {"symptom":"Fever","severity":"","days_ago":1}
"is a fever of 38 dangerous?" -> {}
"my son is coughing a lot today" -> {"symptom":"Cough","severity":"","days_ago":0}

Message: %s
JSON:`

type extractedSymptom struct {
	Symptom  string `json:"symptom"`
	Severity string `json:"severity"`
	DaysAgo  int    `json:"days_ago"`
}

var validSeverity = map[string]bool{"": true, "mild": true, "moderate": true, "severe": true}

// Suggest returns a log suggestion if the message states a loggable
// symptom, or nil. It never blocks the user's answer — the caller runs
// it after replying and treats any failure as "no suggestion".
func (s *SymptomLogger) Suggest(ctx context.Context, message string) *LogSuggestion {
	// Gate on personal ownership first — cheap, and it filters out
	// hypotheticals and general questions before any LLM call.
	if !ownershipCue.MatchString(message) {
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()

	reply, err := s.ollama.Generate(ctx, ChatModel, fmt.Sprintf(symptomPrompt, message))
	if err != nil {
		logger.Warn("[symptom_logger.go/Suggest]:\textraction failed: %v", err)
		return nil
	}

	start := strings.Index(reply, "{")
	end := strings.LastIndex(reply, "}")
	if start == -1 || end <= start {
		return nil
	}

	var ex extractedSymptom
	if err := json.Unmarshal([]byte(reply[start:end+1]), &ex); err != nil {
		return nil
	}

	title := strings.TrimSpace(ex.Symptom)
	if title == "" {
		return nil
	}
	if !validSeverity[ex.Severity] {
		ex.Severity = ""
	}
	if ex.DaysAgo < 0 || ex.DaysAgo > 365 {
		ex.DaysAgo = 0
	}

	day := time.Now().AddDate(0, 0, -ex.DaysAgo).Format("2006-01-02")
	return &LogSuggestion{Date: day, Title: title, Severity: ex.Severity}
}
