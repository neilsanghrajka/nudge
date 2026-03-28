// Package motivation manages motivational quotes.
package motivation

import (
	_ "embed"
	"encoding/json"

	"github.com/neilsanghrajka/nudge/cli/internal/store"
)

//go:embed defaults.json
var defaultsJSON []byte

type Quote struct {
	Text        string   `json:"text"`
	Attribution string   `json:"attribution"`
	Category    string   `json:"category"`
	Phase       []string `json:"phase"`
}

type MotivationStore struct {
	Quotes           []Quote `json:"quotes"`
	UserCustomQuotes []Quote `json:"user_custom_quotes"`
}

// LoadAll returns built-in defaults + user custom quotes.
func LoadAll() (*MotivationStore, error) {
	// Load defaults embedded in binary
	var defaults struct {
		Quotes []Quote `json:"quotes"`
	}
	if err := json.Unmarshal(defaultsJSON, &defaults); err != nil {
		return nil, err
	}

	// Load user customs
	var user MotivationStore
	store.LoadJSON("motivation.json", &user)

	return &MotivationStore{
		Quotes:           defaults.Quotes,
		UserCustomQuotes: user.UserCustomQuotes,
	}, nil
}

// ListByPhase returns all quotes matching a phase (or all if phase is empty).
func ListByPhase(phase string) ([]Quote, error) {
	ms, err := LoadAll()
	if err != nil {
		return nil, err
	}

	all := append(ms.Quotes, ms.UserCustomQuotes...)
	if phase == "" {
		return all, nil
	}

	var filtered []Quote
	for _, q := range all {
		for _, p := range q.Phase {
			if p == phase {
				filtered = append(filtered, q)
				break
			}
		}
	}
	return filtered, nil
}

// AddCustom adds a user quote.
func AddCustom(text, attribution string, phases []string) (*Quote, error) {
	var user MotivationStore
	store.LoadJSON("motivation.json", &user)

	q := Quote{
		Text:        text,
		Attribution: attribution,
		Category:    "custom",
		Phase:       phases,
	}
	user.UserCustomQuotes = append(user.UserCustomQuotes, q)

	if err := store.SaveJSON("motivation.json", &user); err != nil {
		return nil, err
	}
	return &q, nil
}
