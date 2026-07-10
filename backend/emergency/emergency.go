// Package emergency serves curated first-aid cards. The cards are
// embedded in the binary and matched with plain keyword rules — no
// LLM, no external files — so they are instant and always available,
// even if Ollama is down.
package emergency

import (
	"embed"
	"fmt"
	"regexp"
	"strings"
)

//go:embed cards/*.md
var cardFS embed.FS

// Card is one first-aid topic.
//
// File format (cards/*.md):
//
//	# Title
//	keywords: phrase, another phrase, ...
//	<body: the steps>
type Card struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Keywords []string `json:"keywords"`
	Body     string   `json:"body"`

	patterns []*regexp.Regexp
}

var cards []Card

func init() {
	entries, err := cardFS.ReadDir("cards")
	if err != nil {
		panic(fmt.Sprintf("emergency: reading embedded cards: %v", err))
	}

	for _, e := range entries {
		raw, err := cardFS.ReadFile("cards/" + e.Name())
		if err != nil {
			panic(fmt.Sprintf("emergency: reading %s: %v", e.Name(), err))
		}
		card, err := parseCard(strings.TrimSuffix(e.Name(), ".md"), string(raw))
		if err != nil {
			panic(fmt.Sprintf("emergency: parsing %s: %v", e.Name(), err))
		}
		cards = append(cards, card)
	}
}

func parseCard(id, raw string) (Card, error) {
	lines := strings.SplitN(strings.TrimSpace(raw), "\n", 3)
	if len(lines) < 3 || !strings.HasPrefix(lines[0], "# ") || !strings.HasPrefix(lines[1], "keywords:") {
		return Card{}, fmt.Errorf("expected '# Title' then 'keywords:' then body")
	}

	card := Card{
		ID:    id,
		Title: strings.TrimSpace(strings.TrimPrefix(lines[0], "# ")),
		Body:  strings.TrimSpace(lines[2]),
	}

	for _, kw := range strings.Split(strings.TrimPrefix(lines[1], "keywords:"), ",") {
		kw = strings.TrimSpace(kw)
		if kw == "" {
			continue
		}
		card.Keywords = append(card.Keywords, kw)
		// word boundaries so "burned my" never matches "calories burned my..."
		// partially — keywords are matched as whole phrases
		card.patterns = append(card.patterns,
			regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(kw)+`\b`))
	}
	if len(card.Keywords) == 0 {
		return Card{}, fmt.Errorf("no keywords")
	}
	return card, nil
}

// All returns every card (for the Emergency page).
func All() []Card {
	return cards
}

// Match returns the first card whose keywords appear in the message,
// or nil. Deterministic and fast — safe to run on every chat message.
func Match(message string) *Card {
	for i := range cards {
		for _, p := range cards[i].patterns {
			if p.MatchString(message) {
				return &cards[i]
			}
		}
	}
	return nil
}
