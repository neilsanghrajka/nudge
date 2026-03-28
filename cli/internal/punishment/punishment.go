// Package punishment handles executing punishment actions.
package punishment

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os/exec"
	"time"

	"github.com/neilsanghrajka/nudge/cli/internal/config"
)

type SendResult struct {
	Target string `json:"target"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

type ActionInfo struct {
	Name         string   `json:"name"`
	DisplayName  string   `json:"display_name"`
	Configured   bool     `json:"configured"`
	RequiredKeys []string `json:"required_keys"`
}

// Registry of known actions.
var actions = map[string]struct {
	displayName  string
	requiredKeys []string
	send         func(cfg *config.Config, recipient, message string) (bool, string)
	health       func(cfg *config.Config) (bool, string)
}{
	"post_to_beeper_whatsapp": {
		displayName:  "WhatsApp (via Beeper Desktop)",
		requiredKeys: []string{"token"},
		send:         sendBeeper,
		health:       healthBeeper,
	},
	"desktop_notification": {
		displayName:  "macOS Desktop Notification",
		requiredKeys: []string{},
		send:         sendDesktop,
		health:       func(_ *config.Config) (bool, string) { return true, "Always available on macOS" },
	},
}

// Execute runs a punishment action with fallback to desktop notification.
func Execute(actionName, recipient, message string) (bool, string) {
	cfg := config.Load()

	if a, ok := actions[actionName]; ok && actionName != "desktop_notification" {
		ok, detail := a.send(cfg, recipient, message)
		if ok {
			DesktopNotify(message)
			return true, detail
		}
		// Fallback
		fmt.Printf("  [warn] %s failed: %s. Falling back to desktop notification.\n", actionName, detail)
	}

	return sendDesktop(cfg, "", message)
}

// DesktopNotify sends a macOS notification.
func DesktopNotify(message string) {
	if len(message) > 200 {
		message = message[:200]
	}
	exec.Command("osascript", "-e",
		fmt.Sprintf(`display notification "%s" with title "Nudge" sound name "Funk"`, message),
	).Run()
}

// Health checks connectivity for an action.
func Health(actionName string) (bool, string) {
	a, ok := actions[actionName]
	if !ok {
		return false, fmt.Sprintf("unknown action: %s", actionName)
	}
	cfg := config.Load()
	return a.health(cfg)
}

// List returns info about all known actions.
func List() ([]ActionInfo, string) {
	cfg := config.Load()
	var result []ActionInfo
	for name, a := range actions {
		pCfg := cfg.Punishments[name]
		configured := pCfg != nil
		if configured && len(a.requiredKeys) > 0 {
			for _, k := range a.requiredKeys {
				if _, exists := pCfg[k]; !exists {
					configured = false
					break
				}
			}
		}
		result = append(result, ActionInfo{
			Name:         name,
			DisplayName:  a.displayName,
			Configured:   configured,
			RequiredKeys: a.requiredKeys,
		})
	}
	return result, cfg.DefaultPunishment
}

// --- Beeper WhatsApp ---

func sendBeeper(cfg *config.Config, recipient, message string) (bool, string) {
	pCfg := cfg.Punishments["post_to_beeper_whatsapp"]
	if pCfg == nil {
		return false, "not configured"
	}
	token, _ := pCfg["token"].(string)
	beeperURL, _ := pCfg["beeper_url"].(string)
	if beeperURL == "" {
		beeperURL = "http://localhost:23373"
	}
	if token == "" {
		return false, "no token configured"
	}

	encoded := url.PathEscape(recipient)
	apiURL := fmt.Sprintf("%s/v1/chats/%s/messages", beeperURL, encoded)

	body, _ := json.Marshal(map[string]string{"text": message})
	req, _ := http.NewRequest("POST", apiURL, bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("API error: %v", err)
	}
	defer resp.Body.Close()
	io.ReadAll(resp.Body)

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return true, "message sent"
	}
	return false, fmt.Sprintf("HTTP %d", resp.StatusCode)
}

func healthBeeper(cfg *config.Config) (bool, string) {
	pCfg := cfg.Punishments["post_to_beeper_whatsapp"]
	if pCfg == nil {
		return false, "not configured. Run: nudge punishment setup post_to_beeper_whatsapp --token <TOKEN>"
	}
	token, _ := pCfg["token"].(string)
	beeperURL, _ := pCfg["beeper_url"].(string)
	if beeperURL == "" {
		beeperURL = "http://localhost:23373"
	}
	if token == "" {
		return false, "no token"
	}

	apiURL := fmt.Sprintf("%s/v1/chats/search?q=test", beeperURL)
	req, _ := http.NewRequest("GET", apiURL, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return false, fmt.Sprintf("unreachable: %v", err)
	}
	defer resp.Body.Close()
	return true, fmt.Sprintf("reachable (HTTP %d)", resp.StatusCode)
}

// --- Desktop ---

func sendDesktop(_ *config.Config, _, message string) (bool, string) {
	DesktopNotify(message)
	return true, "notification sent"
}
