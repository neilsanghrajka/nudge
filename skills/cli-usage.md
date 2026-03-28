---
name: stake-cli-usage
description: Reference for all stake CLI commands. Use when creating tasks, managing secrets, configuring punishments, or checking status.
---

# Stake CLI Usage

Stake is a command-line tool for accountability with real consequences. All commands support `--json` for machine-readable output.

## Task Commands

### Create a task
```bash
stake task add --desc "Finish the PR review" --duration 30 --why "Team is blocked" --secret-id s-1
stake task add --desc "Ship feature" --duration 60 --punishment post_to_beeper_whatsapp --targets "!roomid:..."
```

**Flags:**
- `--desc` (required): Task description
- `--duration` (default 60): Minutes until deadline
- `--why`: Why this matters (used in reminders — always ask for this)
- `--punishment`: Action name (default: from config or desktop_notification)
- `--targets`: Comma-separated recipient IDs
- `--secret-id`: Secret from the bank to use as punishment content
- `--custom-punishment-message`: Custom punishment text

**Returns:** Task object with `id`, `deadline`, `warning_intervals` (for scheduling reminders).

### Complete a task
```bash
stake task complete task-1
stake task done task-1      # alias
```
Sends all-clear messages via the punishment action. Idempotent — completing an already-completed task is a no-op.

### Fail a task
```bash
stake task fail task-1
```
Executes the punishment: sends the secret/message to all targets via the configured action. Falls back to desktop notification if the action fails.

### Cancel a task
```bash
stake task cancel task-1
```
No messages sent. Use only for legitimate reasons (task became irrelevant).

### Check status
```bash
stake task status           # all active tasks with time remaining
stake task status task-1    # specific task
```

### List and history
```bash
stake task list             # active tasks
stake task list --all       # include history
stake task history          # full history
stake task history --limit 5
```

## Secrets Commands

```bash
stake secrets add --secret "I cry during Disney movies" --severity medium
stake secrets list
stake secrets pick                    # least-used secret
stake secrets pick --severity spicy   # least-used spicy secret
```

Severities: `mild`, `medium`, `spicy`

## Motivation Commands

```bash
stake motivation list                       # all quotes
stake motivation list --phase reminder_late # phase-filtered
stake motivation add --quote "Ship it" --attribution "Me" --phase reminder_late
```

Phases: `task_created`, `reminder_early`, `reminder_mid`, `reminder_late`, `task_completed`, `task_failed`

## Punishment Commands

```bash
stake punishment list                       # show available actions
stake punishment health post_to_beeper_whatsapp  # test connectivity
stake punishment setup post_to_beeper_whatsapp --token abc123 --default-group "!room:..."
stake punishment setup post_to_beeper_whatsapp --add-contact "Alice=!room:..."
```

## Config Commands

```bash
stake config show
stake config set default_punishment post_to_beeper_whatsapp
```

## Cleanup

```bash
stake cleanup --yes    # cancel all active tasks
```

## Global Flags

- `--json`: JSON output envelope `{"ok": true, "command": "...", "data": {...}}`
- `--data-dir`: Override data directory (default: `~/.stake`)
