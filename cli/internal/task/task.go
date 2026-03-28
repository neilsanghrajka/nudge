// Package task manages the accountability task lifecycle.
package task

import (
	"fmt"
	"time"

	"github.com/neilsanghrajka/nudge/cli/internal/punishment"
	"github.com/neilsanghrajka/nudge/cli/internal/store"
)

type Task struct {
	ID                string            `json:"id"`
	Description       string            `json:"description"`
	Why               string            `json:"why,omitempty"`
	DurationMinutes   int               `json:"duration_minutes"`
	PunishmentAction  string            `json:"punishment_action"`
	PunishmentMessage string            `json:"punishment_message,omitempty"`
	Targets           []string          `json:"targets,omitempty"`
	Status            string            `json:"status"`
	CreatedAt         string            `json:"created_at"`
	Deadline          string            `json:"deadline"`
	WarningIntervals  []WarningInterval `json:"warning_intervals"`
	CompletedAt       string            `json:"completed_at,omitempty"`
	FailedAt          string            `json:"failed_at,omitempty"`
	CancelledAt       string            `json:"cancelled_at,omitempty"`
	Proof             string            `json:"proof,omitempty"`
	FailReason        string            `json:"fail_reason,omitempty"`
	SecretID          string            `json:"secret_id,omitempty"`
}

type WarningInterval struct {
	Name             string `json:"name"`
	MinutesFromStart int    `json:"minutes_from_start"`
	MinutesRemaining int    `json:"minutes_remaining"`
	Phase            string `json:"phase"`
	Fired            bool   `json:"fired"`
}

type TaskStore struct {
	Active map[string]*Task `json:"active"`
	NextID int              `json:"next_id"`
}

type History struct {
	Completed []*Task `json:"completed"`
	Failed    []*Task `json:"failed"`
	Cancelled []*Task `json:"cancelled"`
}

func LoadTasks() *TaskStore {
	ts := &TaskStore{Active: map[string]*Task{}, NextID: 1}
	store.LoadJSON("tasks.json", ts)
	if ts.Active == nil {
		ts.Active = map[string]*Task{}
	}
	return ts
}

func SaveTasks(ts *TaskStore) error {
	return store.SaveJSON("tasks.json", ts)
}

func LoadHistory() *History {
	h := &History{}
	store.LoadJSON("history.json", h)
	return h
}

func SaveHistory(h *History) error {
	return store.SaveJSON("history.json", h)
}

// CalculateWarnings returns warning intervals for a given duration.
func CalculateWarnings(durationMinutes int) []WarningInterval {
	var warnings []WarningInterval
	d := durationMinutes

	half := d / 2
	threeQ := (d * 3) / 4
	tenLeft := d - 10
	fiveLeft := d - 5

	if half > 0 {
		warnings = append(warnings, WarningInterval{
			Name: "halfway", MinutesFromStart: half, MinutesRemaining: d - half, Phase: "reminder_early",
		})
	}
	if threeQ > half {
		warnings = append(warnings, WarningInterval{
			Name: "75_percent", MinutesFromStart: threeQ, MinutesRemaining: d - threeQ, Phase: "reminder_mid",
		})
	}
	if tenLeft > threeQ && tenLeft > 0 {
		warnings = append(warnings, WarningInterval{
			Name: "10_min_left", MinutesFromStart: tenLeft, MinutesRemaining: 10, Phase: "reminder_late",
		})
	}
	if fiveLeft > 0 && fiveLeft > max(tenLeft, 0) {
		warnings = append(warnings, WarningInterval{
			Name: "5_min_left", MinutesFromStart: fiveLeft, MinutesRemaining: 5, Phase: "reminder_late",
		})
	}
	return warnings
}

// Add creates a new task and saves it.
func Add(desc string, durationMin int, why string, punishAction string, targets []string, punishMsg string) (*Task, error) {
	ts := LoadTasks()
	id := fmt.Sprintf("task-%d", ts.NextID)
	ts.NextID++

	if punishAction == "" {
		punishAction = "desktop_notification"
	}

	now := time.Now().UTC()
	deadline := now.Add(time.Duration(durationMin) * time.Minute)

	t := &Task{
		ID:                id,
		Description:       desc,
		Why:               why,
		DurationMinutes:   durationMin,
		PunishmentAction:  punishAction,
		PunishmentMessage: punishMsg,
		Targets:           targets,
		Status:            "active",
		CreatedAt:         now.Format(time.RFC3339),
		Deadline:          deadline.Format(time.RFC3339),
		WarningIntervals:  CalculateWarnings(durationMin),
	}

	ts.Active[id] = t
	if err := SaveTasks(ts); err != nil {
		return nil, err
	}
	return t, nil
}

// Complete marks a task as done, sends all-clear messages, moves to history.
func Complete(taskID string, proof string) (*Task, []punishment.SendResult, error) {
	ts := LoadTasks()
	t, ok := ts.Active[taskID]
	if !ok {
		// Idempotent check
		h := LoadHistory()
		for _, c := range h.Completed {
			if c.ID == taskID {
				return c, nil, nil
			}
		}
		return nil, nil, fmt.Errorf("no active task '%s'", taskID)
	}

	t.Proof = proof

	// Send all-clear
	var results []punishment.SendResult
	if t.PunishmentAction != "" && t.PunishmentAction != "desktop_notification" && len(t.Targets) > 0 {
		msg := fmt.Sprintf("✅ Task completed! '%s' — finished in time. No punishment today.", t.Description)
		if proof != "" {
			msg += fmt.Sprintf(" — Verified: %s", proof)
		}
		msg += " -- Nudge"
		for _, target := range t.Targets {
			ok, detail := punishment.Execute(t.PunishmentAction, target, msg)
			results = append(results, punishment.SendResult{Target: target, OK: ok, Detail: detail})
		}
	}
	punishment.DesktopNotify(fmt.Sprintf("PASSED: %s", t.Description))

	t.Status = "completed"
	t.CompletedAt = time.Now().UTC().Format(time.RFC3339)

	h := LoadHistory()
	h.Completed = append(h.Completed, t)
	SaveHistory(h)

	delete(ts.Active, taskID)
	SaveTasks(ts)

	return t, results, nil
}

// Fail marks a task as failed and executes punishment.
func Fail(taskID string, reason string) (*Task, []punishment.SendResult, error) {
	ts := LoadTasks()
	t, ok := ts.Active[taskID]
	if !ok {
		return nil, nil, fmt.Errorf("no active task '%s'", taskID)
	}

	t.FailReason = reason

	punishMsg := t.PunishmentMessage
	if punishMsg == "" {
		punishMsg = fmt.Sprintf("☠️ TIME'S UP! Failed to complete: %s", t.Description)
	}
	if reason != "" {
		punishMsg += fmt.Sprintf(" — Verified: %s", reason)
	}

	var results []punishment.SendResult
	if len(t.Targets) > 0 {
		for _, target := range t.Targets {
			ok, detail := punishment.Execute(t.PunishmentAction, target, punishMsg)
			results = append(results, punishment.SendResult{Target: target, OK: ok, Detail: detail})
		}
	} else {
		ok, detail := punishment.Execute("desktop_notification", "", punishMsg)
		results = append(results, punishment.SendResult{Target: "local", OK: ok, Detail: detail})
	}

	t.Status = "failed"
	t.FailedAt = time.Now().UTC().Format(time.RFC3339)

	h := LoadHistory()
	h.Failed = append(h.Failed, t)
	SaveHistory(h)

	delete(ts.Active, taskID)
	SaveTasks(ts)

	return t, results, nil
}

// Cancel removes a task without punishment.
func Cancel(taskID string) (*Task, error) {
	ts := LoadTasks()
	t, ok := ts.Active[taskID]
	if !ok {
		return nil, fmt.Errorf("no active task '%s'", taskID)
	}

	t.Status = "cancelled"
	t.CancelledAt = time.Now().UTC().Format(time.RFC3339)

	h := LoadHistory()
	h.Cancelled = append(h.Cancelled, t)
	SaveHistory(h)

	delete(ts.Active, taskID)
	SaveTasks(ts)

	return t, nil
}

// CheckResult captures what happened during a check cycle.
type CheckResult struct {
	TaskID          string   `json:"task_id"`
	Description     string   `json:"description"`
	Action          string   `json:"action"` // "warning_fired", "deadline_failed"
	WarningsFired   []string `json:"warnings_fired,omitempty"`
	PunishmentSent  bool     `json:"punishment_sent,omitempty"`
	PunishmentError string   `json:"punishment_error,omitempty"`
}

// Check inspects all active tasks, fires overdue warnings and auto-fails past-deadline tasks.
func Check() ([]CheckResult, error) {
	ts := LoadTasks()
	now := time.Now().UTC()
	var results []CheckResult

	for _, t := range ts.Active {
		created, err := time.Parse(time.RFC3339, t.CreatedAt)
		if err != nil {
			continue
		}
		deadline, err := time.Parse(time.RFC3339, t.Deadline)
		if err != nil {
			continue
		}

		// Check warnings
		var firedNames []string
		for i := range t.WarningIntervals {
			w := &t.WarningIntervals[i]
			if w.Fired {
				continue
			}
			warningTime := created.Add(time.Duration(w.MinutesFromStart) * time.Minute)
			if now.After(warningTime) {
				// Fire this warning
				remaining := int(time.Until(deadline).Minutes())
				if remaining < 0 {
					remaining = 0
				}
				msg := fmt.Sprintf("⏰ %s — %d min left. %s", t.Description, remaining, t.Why)
				punishment.DesktopNotify(msg)
				w.Fired = true
				firedNames = append(firedNames, w.Name)
			}
		}
		if len(firedNames) > 0 {
			results = append(results, CheckResult{
				TaskID:        t.ID,
				Description:   t.Description,
				Action:        "warning_fired",
				WarningsFired: firedNames,
			})
		}

		// Check deadline
		if now.After(deadline) {
			// Auto-fail: execute punishment directly
			punishMsg := t.PunishmentMessage
			if punishMsg == "" {
				punishMsg = fmt.Sprintf("☠️ TIME'S UP! Failed to complete: %s", t.Description)
			}
			punishMsg += " — auto-failed by nudge"

			var punishOK bool
			var punishErr string
			if len(t.Targets) > 0 {
				for _, target := range t.Targets {
					ok, detail := punishment.Execute(t.PunishmentAction, target, punishMsg)
					if !ok {
						punishErr = detail
					}
					punishOK = punishOK || ok
				}
			} else {
				ok, detail := punishment.Execute("desktop_notification", "", punishMsg)
				punishOK = ok
				if !ok {
					punishErr = detail
				}
			}

			t.Status = "failed"
			t.FailReason = "deadline passed (auto-check)"
			t.FailedAt = now.Format(time.RFC3339)

			h := LoadHistory()
			h.Failed = append(h.Failed, t)
			SaveHistory(h)
			delete(ts.Active, t.ID)

			results = append(results, CheckResult{
				TaskID:          t.ID,
				Description:     t.Description,
				Action:          "deadline_failed",
				PunishmentSent:  punishOK,
				PunishmentError: punishErr,
			})
		}
	}

	// Save updated warning states (and removed tasks)
	if err := SaveTasks(ts); err != nil {
		return results, err
	}
	return results, nil
}

// Status returns active tasks with remaining time.
func Status(taskID string) ([]map[string]any, error) {
	ts := LoadTasks()

	if taskID != "" {
		t, ok := ts.Active[taskID]
		if !ok {
			return nil, fmt.Errorf("no active task '%s'", taskID)
		}
		return []map[string]any{taskWithRemaining(t)}, nil
	}

	var result []map[string]any
	for _, t := range ts.Active {
		result = append(result, taskWithRemaining(t))
	}
	return result, nil
}

func taskWithRemaining(t *Task) map[string]any {
	deadline, _ := time.Parse(time.RFC3339, t.Deadline)
	remaining := int(time.Until(deadline).Seconds())
	if remaining < 0 {
		remaining = 0
	}
	return map[string]any{
		"id":                t.ID,
		"description":       t.Description,
		"why":               t.Why,
		"duration_minutes":  t.DurationMinutes,
		"punishment_action": t.PunishmentAction,
		"targets":           t.Targets,
		"status":            t.Status,
		"created_at":        t.CreatedAt,
		"deadline":          t.Deadline,
		"remaining_minutes": remaining / 60,
		"remaining_seconds": remaining,
		"warning_intervals": t.WarningIntervals,
	}
}
