package chat2

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"Kibo/backend/bodyrecord"
	logger "Kibo/backend/kibo_utils"
)

// RecordLogger turns messages like "I weighed 68kg today" into saved
// body records. Extraction uses one LLM call; the confirmation reply
// is built deterministically, so logging feels instant and never
// hallucinates what was saved.
type RecordLogger struct {
	ollama *OllamaClient
	repo   *bodyrecord.Repository
}

func NewRecordLogger(ollama *OllamaClient, repo *bodyrecord.Repository) *RecordLogger {
	return &RecordLogger{ollama: ollama, repo: repo}
}

// defaultUnits also defines which record types may be logged via chat.
var defaultUnits = map[string]string{
	"Weight":   "kg",
	"Sleep":    "hours",
	"Activity": "minutes",
	"Water":    "L",
}

const extractPrompt = `Extract the health measurements the user wants to record.

Record types and their default units:
- Weight (kg)
- Sleep (hours)
- Activity (minutes of exercise)
- Water (L)

Reply with ONLY a JSON array, no other text. One object per measurement:
[{"record_type":"Weight","value":68.5,"unit":"kg","days_ago":0}]

"days_ago" rules:
- 0 for today — including "this morning", "tonight", or no time mentioned
- 1 ONLY when the user says "yesterday" or "last night"
- N for "N days ago"
- a time reference applies to EVERY measurement in the sentence unless
  the user gives a different one ("yesterday I slept 5 hours and drank
  2L" -> both have days_ago 1)
Skip anything without a clear numeric value.

User message: %s
JSON:`

type extractedRecord struct {
	RecordType string  `json:"record_type"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	DaysAgo    int     `json:"days_ago"`
}

// TryLog extracts and saves measurements from the message. It returns
// a confirmation reply and true on success; false means the caller
// should fall back to a normal conversational answer.
func (l *RecordLogger) TryLog(ctx context.Context, userID int64, message string) (string, bool) {
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	reply, err := l.ollama.Generate(ctx, ChatModel, fmt.Sprintf(extractPrompt, message))
	if err != nil {
		logger.Warn("[record_logger.go/TryLog]:\textraction call failed: %v", err)
		return "", false
	}

	records := parseExtractedRecords(reply)
	if len(records) == 0 {
		logger.Info("[record_logger.go/TryLog]:\tno valid records in: %s", reply)
		return "", false
	}

	var saved []string
	for _, rec := range records {
		br := bodyrecord.BodyRecord{
			UserID:     userID,
			RecordType: rec.RecordType,
			Value:      rec.Value,
			Unit:       rec.Unit,
			Timestamp:  time.Now().AddDate(0, 0, -rec.DaysAgo),
		}
		if _, err := l.repo.AddBodyRecord(ctx, br); err != nil {
			logger.Error("[record_logger.go/TryLog]:\tsaving record: %v", err)
			continue
		}
		saved = append(saved, fmt.Sprintf("%s %g %s (%s)",
			rec.RecordType, rec.Value, rec.Unit, describeDaysAgo(rec.DaysAgo)))
	}

	if len(saved) == 0 {
		return "", false
	}

	if len(saved) == 1 {
		return fmt.Sprintf("Recorded ✅ %s — see the trend on the Body record page.", saved[0]), true
	}
	return fmt.Sprintf("Recorded ✅\n- %s\nSee the trends on the Body record page.",
		strings.Join(saved, "\n- ")), true
}

// parseExtractedRecords parses the LLM reply into validated records.
// Models wrap JSON in prose, code fences, or stray quotes, so it tries
// the whole array first and then salvages individual {...} objects.
func parseExtractedRecords(reply string) []extractedRecord {
	var raw []extractedRecord

	start := strings.Index(reply, "[")
	end := strings.LastIndex(reply, "]")
	if start != -1 && end > start {
		if err := json.Unmarshal([]byte(reply[start:end+1]), &raw); err != nil {
			raw = nil
		}
	}

	if raw == nil {
		// malformed array — parse each object on its own
		for rest := reply; ; {
			s := strings.Index(rest, "{")
			if s == -1 {
				break
			}
			e := strings.Index(rest[s:], "}")
			if e == -1 {
				break
			}
			var r extractedRecord
			if err := json.Unmarshal([]byte(rest[s:s+e+1]), &r); err == nil {
				raw = append(raw, r)
			}
			rest = rest[s+e+1:]
		}
	}

	var out []extractedRecord
	for _, r := range raw {
		r.RecordType = titleCase(r.RecordType)
		defUnit, known := defaultUnits[r.RecordType]
		if !known || r.Value <= 0 || r.Value > 100000 {
			continue
		}
		if strings.TrimSpace(r.Unit) == "" {
			r.Unit = defUnit
		}
		if r.DaysAgo < 0 || r.DaysAgo > 365 {
			r.DaysAgo = 0
		}
		out = append(out, r)
	}
	return out
}

// titleCase normalizes "WEIGHT"/"weight" -> "Weight" to match the
// record type names used by the frontend.
func titleCase(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func describeDaysAgo(d int) string {
	switch d {
	case 0:
		return "today"
	case 1:
		return "yesterday"
	default:
		return fmt.Sprintf("%d days ago", d)
	}
}
