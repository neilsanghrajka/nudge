package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/neilsanghrajka/nudge/cli/internal/config"
	"github.com/neilsanghrajka/nudge/cli/internal/motivation"
	"github.com/neilsanghrajka/nudge/cli/internal/punishment"
	"github.com/neilsanghrajka/nudge/cli/internal/secrets"
	"github.com/neilsanghrajka/nudge/cli/internal/store"
	"github.com/neilsanghrajka/nudge/cli/internal/task"
)

var (
	jsonMode bool
	version  = "dev"
)

func main() {
	args := os.Args[1:]

	// Parse global flags
	var filtered []string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonMode = true
		case "--data-dir":
			if i+1 < len(args) {
				store.SetDataDir(args[i+1])
				i++
			}
		default:
			filtered = append(filtered, args[i])
		}
	}
	args = filtered

	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	// Force data dir init
	store.DataDir()

	resource := args[0]
	rest := args[1:]

	switch resource {
	case "task":
		handleTask(rest)
	case "secrets":
		handleSecrets(rest)
	case "motivation":
		handleMotivation(rest)
	case "punishment":
		handlePunishment(rest)
	case "config":
		handleConfig(rest)
	case "cleanup":
		handleCleanup(rest)
	case "version", "--version", "-v":
		fmt.Println("nudge " + version)
	case "help", "--help", "-h":
		printUsage()
	default:
		exitErr(resource, "unknown command: "+resource, "UNKNOWN_COMMAND")
	}
}

// ─── Task ───

func handleTask(args []string) {
	if len(args) == 0 {
		printTaskHelp()
		return
	}

	switch args[0] {
	case "add":
		taskAdd(args[1:])
	case "complete", "done":
		taskComplete(args[1:])
	case "fail":
		taskFail(args[1:])
	case "cancel":
		taskCancel(args[1:])
	case "status":
		taskStatus(args[1:])
	case "list":
		taskList(args[1:])
	case "history":
		taskHistory(args[1:])
	case "check":
		taskCheck()
	case "daemon":
		taskDaemon(args[1:])
	case "help", "--help", "-h":
		printTaskHelp()
	default:
		exitErr("task", "unknown action: "+args[0], "UNKNOWN_ACTION")
	}
}

func taskAdd(args []string) {
	var desc, why, punishAction, targets, secretID, customMsg string
	duration := 60

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--desc":
			i++
			desc = args[i]
		case "--duration", "--in":
			i++
			duration, _ = strconv.Atoi(args[i])
		case "--why":
			i++
			why = args[i]
		case "--punishment":
			i++
			punishAction = args[i]
		case "--targets":
			i++
			targets = args[i]
		case "--secret-id":
			i++
			secretID = args[i]
		case "--custom-punishment-message":
			i++
			customMsg = args[i]
		}
	}

	if desc == "" {
		exitErr("task add", "missing --desc. Example: nudge task add --desc 'Finish PR' --duration 30 --why 'Team waiting'", "MISSING_DESC")
		return
	}

	// Resolve punishment action from config if not specified
	if punishAction == "" {
		cfg := config.Load()
		punishAction = cfg.DefaultPunishment
	}

	// Resolve targets
	var targetList []string
	if targets != "" {
		targetList = strings.Split(targets, ",")
		for i := range targetList {
			targetList[i] = strings.TrimSpace(targetList[i])
		}
	} else if punishAction == "post_to_beeper_whatsapp" {
		cfg := config.Load()
		if pCfg, ok := cfg.Punishments["post_to_beeper_whatsapp"]; ok {
			if dg, ok := pCfg["default_group"].(string); ok && dg != "" {
				targetList = []string{dg}
			}
		}
	}

	// Resolve secret
	punishMsg := customMsg
	if secretID != "" {
		s := secrets.Get(secretID)
		if s == nil {
			exitErr("task add", fmt.Sprintf("secret '%s' not found", secretID), "SECRET_NOT_FOUND")
			return
		}
		if s.Revealed && !jsonMode {
			fmt.Fprintf(os.Stderr, "  [warn] Secret '%s' has already been revealed in a previous punishment. Consider using an unused secret.\n", secretID)
		}
		punishMsg = s.Secret
		secrets.MarkUsed(secretID)
	}

	t, err := task.Add(desc, duration, why, punishAction, targetList, punishMsg)
	if err == nil && secretID != "" {
		// Store secret ID on task for marking revealed on fail
		ts := task.LoadTasks()
		if active, ok := ts.Active[t.ID]; ok {
			active.SecretID = secretID
			task.SaveTasks(ts)
		}
	}
	if err != nil {
		exitErr("task add", err.Error(), "ADD_FAILED")
		return
	}

	crons := task.CronJobsForTask(t)
	outputOK("task add", map[string]any{"task": t, "crons": crons})
	if !jsonMode {
		fmt.Printf("\n  Task created: %s\n", t.ID)
		fmt.Printf("  Description: %s\n", t.Description)
		if t.Why != "" {
			fmt.Printf("  Why: %s\n", t.Why)
		}
		fmt.Printf("  Duration: %d min\n", t.DurationMinutes)
		fmt.Printf("  Deadline: %s\n", t.Deadline)
		fmt.Printf("  Punishment: %s\n", t.PunishmentAction)
		fmt.Printf("  Warnings: %d scheduled\n", len(t.WarningIntervals))
		fmt.Printf("  Create %d one-shot openclaw cron job(s):\n", len(crons))
		for _, cron := range crons {
			fmt.Printf("    - %s at %s -> %s\n", cron.Name, cron.At, cron.Command)
		}
	}
}

func taskComplete(args []string) {
	if len(args) == 0 {
		exitErr("task complete", "missing task_id. Example: nudge task complete task-1 --proof 'Strava: 18 min walk'", "MISSING_ID")
		return
	}
	taskID := args[0]
	var proof string
	for i := 1; i < len(args); i++ {
		if args[i] == "--proof" && i+1 < len(args) {
			i++
			proof = args[i]
		}
	}
	t, results, err := task.Complete(taskID, proof)
	if err != nil {
		exitErr("task complete", err.Error(), "COMPLETE_FAILED")
		return
	}
	cancelCrons := task.CancelCronNames(t)
	outputOK("task complete", map[string]any{"task": t, "messages_sent": results, "cancel_crons": cancelCrons})
	if !jsonMode {
		fmt.Printf("\n  Task '%s' completed!\n", t.ID)
		if proof != "" {
			fmt.Printf("  Proof: %s\n", proof)
		}
		fmt.Printf("  Cancel cron jobs: %s\n", strings.Join(cancelCrons, ", "))
	}
}

func taskFail(args []string) {
	if len(args) == 0 {
		exitErr("task fail", "missing task_id. Example: nudge task fail task-1 --reason 'no slides submitted'", "MISSING_ID")
		return
	}
	taskID := args[0]
	var reason string
	for i := 1; i < len(args); i++ {
		if args[i] == "--reason" && i+1 < len(args) {
			i++
			reason = args[i]
		}
	}
	t, results, err := task.Fail(taskID, reason)
	if err != nil {
		exitErr("task fail", err.Error(), "FAIL_FAILED")
		return
	}
	cancelCrons := task.CancelCronNames(t)
	outputOK("task fail", map[string]any{"task": t, "messages_sent": results, "cancel_crons": cancelCrons})
	if !jsonMode {
		fmt.Printf("\n  Task '%s' FAILED. Punishment sent.\n", t.ID)
		if reason != "" {
			fmt.Printf("  Reason: %s\n", reason)
		}
		fmt.Printf("  Cancel cron jobs: %s\n", strings.Join(cancelCrons, ", "))
	}
}

func taskCancel(args []string) {
	if len(args) == 0 {
		exitErr("task cancel", "missing task_id", "MISSING_ID")
		return
	}
	t, err := task.Cancel(args[0])
	if err != nil {
		exitErr("task cancel", err.Error(), "CANCEL_FAILED")
		return
	}
	cancelCrons := task.CancelCronNames(t)
	outputOK("task cancel", map[string]any{"task": t, "cancel_crons": cancelCrons})
	if !jsonMode {
		fmt.Printf("  Task '%s' cancelled.\n", t.ID)
		fmt.Printf("  Cancel cron jobs: %s\n", strings.Join(cancelCrons, ", "))
	}
}

func taskStatus(args []string) {
	taskID := ""
	if len(args) > 0 {
		taskID = args[0]
	}
	statuses, err := task.Status(taskID)
	if err != nil {
		exitErr("task status", err.Error(), "STATUS_FAILED")
		return
	}
	outputOK("task status", map[string]any{"tasks": statuses})
	if !jsonMode {
		if len(statuses) == 0 {
			fmt.Println("No active tasks.")
			return
		}
		for _, s := range statuses {
			mins := s["remaining_minutes"]
			fmt.Printf("  %s: %s — %v min left\n", s["id"], s["description"], mins)
			if w, ok := s["why"].(string); ok && w != "" {
				fmt.Printf("    Why: %s\n", w)
			}
		}
	}
}

func taskList(args []string) {
	showAll := false
	for _, a := range args {
		if a == "--all" {
			showAll = true
		}
	}
	ts := task.LoadTasks()
	h := task.LoadHistory()

	result := map[string]any{"active": ts.Active}
	if showAll {
		result["completed"] = lastN(h.Completed, 5)
		result["failed"] = lastN(h.Failed, 5)
		result["cancelled"] = lastN(h.Cancelled, 5)
	}
	outputOK("task list", result)
	if !jsonMode {
		fmt.Println("=== Active Tasks ===")
		if len(ts.Active) == 0 {
			fmt.Println("  (none)")
		}
		for _, t := range ts.Active {
			fmt.Printf("  %s: %s (%d min)\n", t.ID, t.Description, t.DurationMinutes)
		}
	}
}

func taskHistory(args []string) {
	limit := 10
	for i, a := range args {
		if a == "--limit" && i+1 < len(args) {
			limit, _ = strconv.Atoi(args[i+1])
		}
	}
	h := task.LoadHistory()
	type entry struct {
		*task.Task
		Result string `json:"result"`
	}
	var all []entry
	for _, t := range h.Completed {
		all = append(all, entry{t, "completed"})
	}
	for _, t := range h.Failed {
		all = append(all, entry{t, "failed"})
	}
	for _, t := range h.Cancelled {
		all = append(all, entry{t, "cancelled"})
	}
	if len(all) > limit {
		all = all[len(all)-limit:]
	}
	outputOK("task history", map[string]any{"history": all, "total": len(all)})
	if !jsonMode {
		for _, e := range all {
			icon := map[string]string{"completed": "[OK]", "failed": "[FAIL]", "cancelled": "[CANCEL]"}[e.Result]
			fmt.Printf("  %s %s: %s (%dmin)\n", icon, e.ID, e.Description, e.DurationMinutes)
		}
	}
}

func taskCheck() {
	results, err := task.Check()
	if err != nil {
		exitErr("task check", err.Error(), "CHECK_FAILED")
		return
	}

	outputOK("task check", map[string]any{"results": results, "count": len(results)})
	if !jsonMode {
		if len(results) == 0 {
			fmt.Println("All tasks OK. Nothing to fire.")
			return
		}
		for _, r := range results {
			switch r.Action {
			case "warning_fired":
				fmt.Printf("  ⏰ %s: fired warnings %s\n", r.TaskID, strings.Join(r.WarningsFired, ", "))
			case "deadline_failed":
				status := "sent"
				if !r.PunishmentSent {
					status = "FAILED: " + r.PunishmentError
				}
				fmt.Printf("  ☠️  %s: deadline passed, punishment %s\n", r.TaskID, status)
			}
		}
	}
}

func taskDaemon(args []string) {
	interval := 30
	for i := 0; i < len(args); i++ {
		if args[i] == "--interval" && i+1 < len(args) {
			interval, _ = strconv.Atoi(args[i+1])
		}
	}
	if interval < 5 {
		interval = 5
	}

	if !jsonMode {
		fmt.Printf("Nudge daemon started. Checking every %ds. Press Ctrl+C to stop.\n", interval)
	}

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	ticker := time.NewTicker(time.Duration(interval) * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	runCheck := func() {
		results, err := task.Check()
		if err != nil && !jsonMode {
			fmt.Fprintf(os.Stderr, "  check error: %v\n", err)
			return
		}
		if jsonMode && len(results) > 0 {
			b, _ := json.Marshal(map[string]any{"event": "check", "results": results})
			fmt.Println(string(b))
		} else if !jsonMode {
			for _, r := range results {
				switch r.Action {
				case "warning_fired":
					fmt.Printf("  ⏰ %s: %s\n", r.TaskID, strings.Join(r.WarningsFired, ", "))
				case "deadline_failed":
					fmt.Printf("  ☠️  %s: auto-failed\n", r.TaskID)
				}
			}
		}
	}

	runCheck()
	for {
		select {
		case <-ticker.C:
			runCheck()
		case <-sig:
			if !jsonMode {
				fmt.Println("\nDaemon stopped.")
			}
			return
		}
	}
}

// ─── Secrets ───

func handleSecrets(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nudge secrets <add|list|pick> [flags]")
		return
	}
	switch args[0] {
	case "add":
		secretsAdd(args[1:])
	case "list":
		secretsList()
	case "pick":
		secretsPick(args[1:])
	default:
		exitErr("secrets", "unknown action: "+args[0], "UNKNOWN_ACTION")
	}
}

func secretsAdd(args []string) {
	var secret, severity string
	severity = "medium"
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--secret":
			i++
			secret = args[i]
		case "--severity":
			i++
			severity = args[i]
		}
	}
	if secret == "" {
		exitErr("secrets add", "missing --secret. Example: nudge secrets add --secret 'I sleep with a nightlight' --severity medium", "MISSING_SECRET")
		return
	}
	s, err := secrets.Add(secret, severity)
	if err != nil {
		exitErr("secrets add", err.Error(), "ADD_FAILED")
		return
	}
	outputOK("secrets add", s)
	if !jsonMode {
		fmt.Printf("Secret added: %s (%s)\n", s.ID, s.Severity)
	}
}

func secretsList() {
	ss := secrets.Load()
	outputOK("secrets list", map[string]any{"secrets": ss.Secrets})
	if !jsonMode {
		if len(ss.Secrets) == 0 {
			fmt.Println("No secrets. Add one: nudge secrets add --secret '...'")
			return
		}
		for _, s := range ss.Secrets {
			revealed := ""
			if s.Revealed {
				revealed = " [REVEALED]"
			}
			fmt.Printf("  %s [%s] (used %dx)%s: %s\n", s.ID, s.Severity, s.TimesUsed, revealed, s.Secret)
		}
	}
}

func secretsPick(args []string) {
	var severity string
	unusedOnly := false
	for i := 0; i < len(args); i++ {
		if args[i] == "--severity" && i+1 < len(args) {
			severity = args[i+1]
		}
		if args[i] == "--unused" {
			unusedOnly = true
		}
	}
	s, err := secrets.Pick(severity, unusedOnly)
	if err != nil {
		exitErr("secrets pick", err.Error(), "PICK_FAILED")
		return
	}
	outputOK("secrets pick", s)
	if !jsonMode {
		fmt.Printf("Picked: %s (%s): %s\n", s.ID, s.Severity, s.Secret)
	}
}

// ─── Motivation ───

func handleMotivation(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nudge motivation <list|add> [flags]")
		return
	}
	switch args[0] {
	case "list":
		motivationList(args[1:])
	case "add":
		motivationAdd(args[1:])
	default:
		exitErr("motivation", "unknown action: "+args[0], "UNKNOWN_ACTION")
	}
}

func motivationList(args []string) {
	var phase string
	for i := 0; i < len(args); i++ {
		if args[i] == "--phase" && i+1 < len(args) {
			phase = args[i+1]
		}
	}
	quotes, err := motivation.ListByPhase(phase)
	if err != nil {
		exitErr("motivation list", err.Error(), "LIST_FAILED")
		return
	}
	outputOK("motivation list", map[string]any{"quotes": quotes})
	if !jsonMode {
		for _, q := range quotes {
			fmt.Printf("  \"%s\" — %s [%s]\n", q.Text, q.Attribution, strings.Join(q.Phase, ", "))
		}
	}
}

func motivationAdd(args []string) {
	var quote, attribution, phaseStr string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--quote":
			i++
			quote = args[i]
		case "--attribution":
			i++
			attribution = args[i]
		case "--phase":
			i++
			phaseStr = args[i]
		}
	}
	if quote == "" {
		exitErr("motivation add", "missing --quote", "MISSING_QUOTE")
		return
	}
	phases := []string{"reminder_mid"}
	if phaseStr != "" {
		phases = strings.Split(phaseStr, ",")
	}
	q, err := motivation.AddCustom(quote, attribution, phases)
	if err != nil {
		exitErr("motivation add", err.Error(), "ADD_FAILED")
		return
	}
	outputOK("motivation add", q)
	if !jsonMode {
		fmt.Printf("Quote added: \"%s\" — %s\n", q.Text, q.Attribution)
	}
}

// ─── Punishment ───

func handlePunishment(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nudge punishment <list|setup|health> [flags]")
		return
	}
	switch args[0] {
	case "list":
		punishmentList()
	case "setup":
		punishmentSetup(args[1:])
	case "health":
		punishmentHealth(args[1:])
	default:
		exitErr("punishment", "unknown action: "+args[0], "UNKNOWN_ACTION")
	}
}

func punishmentList() {
	actions, defaultAction := punishment.List()
	outputOK("punishment list", map[string]any{"actions": actions, "default": defaultAction})
	if !jsonMode {
		fmt.Println("Available punishment actions:")
		for _, a := range actions {
			status := "[not configured]"
			if a.Configured {
				status = "[configured]"
			}
			def := ""
			if defaultAction == a.Name {
				def = " (default)"
			}
			fmt.Printf("  %s: %s %s%s\n", a.Name, a.DisplayName, status, def)
		}
	}
}

func punishmentSetup(args []string) {
	if len(args) == 0 {
		exitErr("punishment setup", "missing action name. Example: nudge punishment setup post_to_beeper_whatsapp --token abc123", "MISSING_ACTION")
		return
	}
	actionName := args[0]
	cfg := config.Load()
	if cfg.Punishments[actionName] == nil {
		cfg.Punishments[actionName] = map[string]any{}
	}

	for i := 1; i < len(args); i++ {
		if strings.HasPrefix(args[i], "--") && i+1 < len(args) {
			key := strings.TrimPrefix(args[i], "--")
			key = strings.ReplaceAll(key, "-", "_")
			val := args[i+1]
			i++

			if key == "add_contact" {
				parts := strings.SplitN(val, "=", 2)
				if len(parts) == 2 {
					contacts, _ := cfg.Punishments[actionName]["contacts"].(map[string]any)
					if contacts == nil {
						contacts = map[string]any{}
					}
					contacts[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
					cfg.Punishments[actionName]["contacts"] = contacts
				}
			} else {
				cfg.Punishments[actionName][key] = val
			}
		}
	}

	if cfg.DefaultPunishment == "" {
		cfg.DefaultPunishment = actionName
	}

	config.Save(cfg)
	outputOK("punishment setup", map[string]any{"action": actionName, "config": cfg.Punishments[actionName]})
	if !jsonMode {
		fmt.Printf("Configured: %s\n", actionName)
	}
}

func punishmentHealth(args []string) {
	if len(args) == 0 {
		exitErr("punishment health", "missing action name", "MISSING_ACTION")
		return
	}
	ok, detail := punishment.Health(args[0])
	outputOK("punishment health", map[string]any{"action": args[0], "healthy": ok, "detail": detail})
	if !jsonMode {
		status := "[OK]"
		if !ok {
			status = "[FAIL]"
		}
		fmt.Printf("%s %s: %s\n", status, args[0], detail)
	}
}

// ─── Config ───

func handleConfig(args []string) {
	if len(args) == 0 {
		fmt.Println("Usage: nudge config <show|set> [flags]")
		return
	}
	switch args[0] {
	case "show":
		cfg := config.Load()
		outputOK("config show", cfg)
		if !jsonMode {
			b, _ := json.MarshalIndent(cfg, "", "  ")
			fmt.Println(string(b))
		}
	case "set":
		if len(args) < 3 {
			exitErr("config set", "usage: nudge config set <key> <value>", "MISSING_ARGS")
			return
		}
		cfg := config.Load()
		if args[1] == "default_punishment" {
			cfg.DefaultPunishment = args[2]
		}
		config.Save(cfg)
		outputOK("config set", map[string]any{"key": args[1], "value": args[2]})
		if !jsonMode {
			fmt.Printf("Set %s = %s\n", args[1], args[2])
		}
	default:
		exitErr("config", "unknown action: "+args[0], "UNKNOWN_ACTION")
	}
}

// ─── Cleanup ───

func handleCleanup(args []string) {
	yes := false
	for _, a := range args {
		if a == "--yes" {
			yes = true
		}
	}
	if !yes && !jsonMode {
		fmt.Println("This will cancel all active tasks. Run with --yes to confirm.")
		os.Exit(1)
	}

	ts := task.LoadTasks()
	count := len(ts.Active)
	ts.Active = map[string]*task.Task{}
	task.SaveTasks(ts)

	outputOK("cleanup", map[string]any{"tasks_cancelled": count})
	if !jsonMode {
		fmt.Printf("Cleanup done. %d task(s) cancelled.\n", count)
	}
}

// ─── Output helpers ───

func outputOK(command string, data any) {
	if jsonMode {
		b, _ := json.Marshal(map[string]any{"ok": true, "command": command, "data": data})
		fmt.Println(string(b))
	}
}

func exitErr(command, error, code string) {
	if jsonMode {
		b, _ := json.Marshal(map[string]any{"ok": false, "command": command, "error": error, "code": code})
		fmt.Println(string(b))
	} else {
		fmt.Fprintf(os.Stderr, "Error: %s\n", error)
	}
	os.Exit(1)
}

func printUsage() {
	fmt.Print(`nudge — motivation through consequences

Usage: nudge <command> [flags]

Commands:
  task          Manage accountability tasks
  secrets       Manage embarrassing secrets bank
  motivation    Manage motivational quotes
  punishment    Configure punishment actions
  config        View/update configuration
  cleanup       Cancel all tasks

Global flags:
  --json        Machine-readable JSON output
  --data-dir    Override data directory (default: ~/.nudge)

Examples:
  nudge task add --desc "Finish PR" --duration 30 --why "Team waiting"
  nudge task complete task-1 --proof "PR merged, tests passing"
  nudge secrets add --secret "I sleep with a nightlight" --severity medium
  nudge punishment setup post_to_beeper_whatsapp --token abc123

Run 'nudge <command> --help' for command-specific help.
`)
}

func printTaskHelp() {
	fmt.Print(`nudge task — manage accountability tasks

Usage: nudge task <action> [flags]

Actions:
  add       Create a new task with deadline and consequences
  complete  Mark task as done (sends all-clear) [--proof "..."]
  done      Alias for complete
  fail      Mark task as failed (executes punishment) [--reason "..."]
  cancel    Cancel task (no messages)
  status    Show active tasks with time remaining
  list      List tasks (--all for history)
  history   Show task history (--limit N)
  check     Check all tasks: fire overdue warnings & auto-fail past deadlines
  daemon    Convenience loop for short sprints; checks every 30s (--interval N)

Examples:
  nudge task add --desc "Ship feature" --duration 60 --why "Demo tomorrow" --secret-id s-1
  nudge task add --desc "Ship feature" --duration 60 --why "Demo tomorrow" --secret-id s-1 --json
  nudge task add --desc "Write tests" --duration 30 --punishment post_to_beeper_whatsapp --targets "!room:..."
  nudge task complete task-1 --proof "Strava: 18 min walk recorded at 4:45 PM"
  nudge task fail task-1 --reason "no slides submitted before deadline"
  nudge task status
  nudge task check
  # Preferred for reliable automation: parse the returned crons array and create one-shot openclaw jobs.
  nudge task daemon --interval 30
  nudge task list --all
  nudge task history --limit 5
`)
}

// ─── Utilities ───

func lastN(tasks []*task.Task, n int) []*task.Task {
	if len(tasks) <= n {
		return tasks
	}
	return tasks[len(tasks)-n:]
}
