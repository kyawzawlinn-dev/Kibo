package chat2

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Classification labels a user message in a single LLM call:
// Intent is what the message is about, Service is what kind of help
// the user wants. One call instead of two halves the classification
// latency on slow hardware.
type Classification struct {
	Intent  string
	Service string
}

// NeedsRAG reports whether the knowledge base is worth searching for
// this message. The knowledge base is health content, so retrieval
// only helps health-related intents; everything else goes straight to
// the LLM with conversation context.
func (c Classification) NeedsRAG() bool {
	return strings.HasPrefix(c.Intent, "HEALTH_") || c.Intent == "DEFINE_TERM"
}

var validIntents = map[string]bool{
	"GENERAL": true, "DEFINE_TERM": true, "LOG_RECORD": true,
	"HEALTH_SYMPTOM": true, "HEALTH_MEDICATION": true, "HEALTH_INFO": true,
}

var validServices = map[string]bool{
	"EXPLAIN": true, "INSTRUCT": true, "TROUBLESHOOT": true,
	"SUGGEST": true, "SUMMARIZE": true, "WARN": true,
}

const classifierPrompt = `Classify the user's message.

INTENT must be ONE of:
- GENERAL            (greetings, chit-chat, anything not health-related)
- LOG_RECORD         (stating a measurement to record: "I weighed 68kg today", "slept 6 hours", "drank 2L of water")
- DEFINE_TERM        (asking what a medical/health term means)
- HEALTH_SYMPTOM     (describing symptoms or asking about them)
- HEALTH_MEDICATION  (asking about medicines or dosages)
- HEALTH_INFO        (other health, diet, sleep, or fitness questions)

SERVICE must be ONE of: EXPLAIN, INSTRUCT, TROUBLESHOOT, SUGGEST, SUMMARIZE, WARN

Rule: if the user STATES measurements with numbers (weight, sleep hours,
water, exercise) without asking a question, it is LOG_RECORD — even if
they add commentary like "only 5 hours". Choose HEALTH_* only when they
ASK something.

Examples:
"yesterday I slept 5 hours and drank 2 liters of water" -> LOG_RECORD,SUMMARIZE
"why do I sleep so badly?" -> HEALTH_SYMPTOM,EXPLAIN

Reply with exactly two labels separated by a comma, nothing else.

User message: %s
Reply:`

// Classifier labels messages using the local LLM.
type Classifier struct {
	ollama *OllamaClient
}

func NewClassifier(ollama *OllamaClient) *Classifier {
	return &Classifier{ollama: ollama}
}

// Classify never fails the request: on any error or malformed reply it
// falls back to a safe default that routes through RAG.
func (c *Classifier) Classify(ctx context.Context, message string) Classification {
	fallback := Classification{Intent: "HEALTH_INFO", Service: "EXPLAIN"}

	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	reply, err := c.ollama.Generate(ctx, ChatModel, fmt.Sprintf(classifierPrompt, message))
	if err != nil {
		return fallback
	}

	parts := strings.SplitN(strings.ToUpper(strings.TrimSpace(reply)), ",", 2)
	if len(parts) != 2 {
		return fallback
	}

	result := Classification{
		Intent:  strings.TrimSpace(parts[0]),
		Service: strings.TrimSpace(parts[1]),
	}
	if !validIntents[result.Intent] {
		result.Intent = fallback.Intent
	}
	if !validServices[result.Service] {
		result.Service = fallback.Service
	}
	return result
}
