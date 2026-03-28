// Package secrets manages the embarrassing secrets bank.
package secrets

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/neilsanghrajka/nudge/cli/internal/store"
)

type Secret struct {
	ID        string `json:"id"`
	Secret    string `json:"secret"`
	Severity  string `json:"severity"`
	AddedAt   string `json:"added_at"`
	TimesUsed int    `json:"times_used"`
	Revealed  bool   `json:"revealed"`
}

type SecretsStore struct {
	Secrets []*Secret `json:"secrets"`
}

func Load() *SecretsStore {
	s := &SecretsStore{}
	store.LoadJSON("secrets.json", s)
	return s
}

func Save(s *SecretsStore) error {
	return store.SaveJSON("secrets.json", s)
}

// Add creates a new secret.
func Add(secret, severity string) (*Secret, error) {
	ss := Load()

	nextID := 1
	for _, s := range ss.Secrets {
		var n int
		if _, err := fmt.Sscanf(s.ID, "s-%d", &n); err == nil && n >= nextID {
			nextID = n + 1
		}
	}

	entry := &Secret{
		ID:        fmt.Sprintf("s-%d", nextID),
		Secret:    secret,
		Severity:  severity,
		AddedAt:   time.Now().UTC().Format(time.RFC3339),
		TimesUsed: 0,
	}

	ss.Secrets = append(ss.Secrets, entry)
	if err := Save(ss); err != nil {
		return nil, err
	}
	return entry, nil
}

// Pick selects a secret, preferring unrevealed and least-used. Optionally filter by severity and unused-only.
func Pick(severity string, unusedOnly bool) (*Secret, error) {
	ss := Load()
	candidates := ss.Secrets

	if severity != "" {
		var filtered []*Secret
		for _, s := range candidates {
			if s.Severity == severity {
				filtered = append(filtered, s)
			}
		}
		candidates = filtered
	}

	if unusedOnly {
		var filtered []*Secret
		for _, s := range candidates {
			if !s.Revealed {
				filtered = append(filtered, s)
			}
		}
		if len(filtered) == 0 {
			return nil, fmt.Errorf("no unused (unrevealed) secrets. Add more: nudge secrets add --secret '...'")
		}
		candidates = filtered
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no matching secrets. Add some first: nudge secrets add --secret '...'")
	}

	// Prefer unrevealed secrets first
	var unrevealed []*Secret
	for _, s := range candidates {
		if !s.Revealed {
			unrevealed = append(unrevealed, s)
		}
	}
	if len(unrevealed) > 0 {
		candidates = unrevealed
	}

	// Find minimum times_used
	minUsed := candidates[0].TimesUsed
	for _, s := range candidates {
		if s.TimesUsed < minUsed {
			minUsed = s.TimesUsed
		}
	}

	// Filter to least-used
	var leastUsed []*Secret
	for _, s := range candidates {
		if s.TimesUsed == minUsed {
			leastUsed = append(leastUsed, s)
		}
	}

	picked := leastUsed[rand.Intn(len(leastUsed))]
	return picked, nil
}

// MarkRevealed marks a secret as revealed (punishment was sent).
func MarkRevealed(secretID string) {
	ss := Load()
	for _, s := range ss.Secrets {
		if s.ID == secretID {
			s.Revealed = true
			Save(ss)
			return
		}
	}
}

// MarkUsed increments the usage counter for a secret.
func MarkUsed(secretID string) {
	ss := Load()
	for _, s := range ss.Secrets {
		if s.ID == secretID {
			s.TimesUsed++
			Save(ss)
			return
		}
	}
}

// Get returns a secret by ID.
func Get(secretID string) *Secret {
	ss := Load()
	for _, s := range ss.Secrets {
		if s.ID == secretID {
			return s
		}
	}
	return nil
}
