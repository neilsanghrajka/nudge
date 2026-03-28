#!/usr/bin/env python3
"""
Accountability Coach CLI — motivation through consequences.

Usage:
  accountability.py task add --desc "..." --duration 60 --why "..." [--punishment ACTION] [--targets "id1,id2"] [--secret-id S-1]
  accountability.py task complete <task_id>
  accountability.py task fail <task_id>
  accountability.py task cancel <task_id>
  accountability.py task status [task_id]
  accountability.py task list [--all]
  accountability.py task history [--limit 10]
  accountability.py secrets add --secret "..." [--severity mild|medium|spicy]
  accountability.py secrets list
  accountability.py secrets pick [--severity spicy]
  accountability.py motivation list [--phase reminder_mid]
  accountability.py motivation add --quote "..." --attribution "..." [--phase task_created,reminder_early]
  accountability.py punishment list
  accountability.py punishment setup <action_name> [--key value ...]
  accountability.py punishment health <action_name>
  accountability.py config show
  accountability.py config set <key> <value>
  accountability.py cleanup [--yes]

Global flags:
  --json         Machine-readable JSON output
  --data-dir     Override data directory (default: ~/.accountability)

Examples:
  accountability.py task add --desc "Finish API refactor" --duration 120 --why "Team is blocked on me" --json
  accountability.py secrets add --secret "I cried watching a dog food commercial" --severity medium
  accountability.py punishment setup post_to_beeper_whatsapp --token abc123 --beeper-url http://localhost:23373
  accountability.py punishment health post_to_beeper_whatsapp
  accountability.py task complete task-1 --json
"""

import argparse
import json
import os
import random
import subprocess
import sys
import urllib.parse
import urllib.request
from datetime import datetime, timedelta, timezone
from pathlib import Path

# ─── Paths ───

DEFAULT_DATA_DIR = Path.home() / ".accountability"

def get_data_dir(override=None):
    if override:
        return Path(override)
    return Path(os.environ.get("ACCOUNTABILITY_DATA_DIR", str(DEFAULT_DATA_DIR)))

# ─── JSON output ───

_json_mode = False

def set_json_mode(enabled):
    global _json_mode
    _json_mode = enabled

def output(command, data=None, error=None, code=None):
    """Output result. In JSON mode: envelope to stdout. Otherwise: caller handles printing."""
    if _json_mode:
        result = {"ok": error is None, "command": command}
        if data is not None:
            result["data"] = data
        if error is not None:
            result["error"] = error
        if code is not None:
            result["code"] = code
        print(json.dumps(result, default=str))
        if error:
            sys.exit(1)
    elif error:
        print(f"Error: {error}", file=sys.stderr)
        if code:
            print(f"  Code: {code}", file=sys.stderr)
        sys.exit(1)

def output_ok(command, data):
    output(command, data=data)

def output_err(command, error, code=None):
    output(command, error=error, code=code)

# ─── State Management ───

def load_json(path, default=None):
    if default is None:
        default = {}
    if path.exists():
        with open(path) as f:
            return json.load(f)
    return default

def save_json(path, data):
    path.parent.mkdir(parents=True, exist_ok=True)
    with open(path, "w") as f:
        json.dump(data, f, indent=2, default=str)

def load_tasks(data_dir):
    return load_json(data_dir / "tasks.json", {"active": {}, "next_id": 1})

def save_tasks(data_dir, tasks):
    save_json(data_dir / "tasks.json", tasks)

def load_config(data_dir):
    return load_json(data_dir / "config.json", {
        "punishments": {},
        "default_punishment": None
    })

def save_config(data_dir, config):
    save_json(data_dir / "config.json", config)

def load_secrets(data_dir):
    return load_json(data_dir / "secrets.json", {"secrets": []})

def save_secrets(data_dir, data):
    save_json(data_dir / "secrets.json", data)

def load_motivation(data_dir):
    # Load defaults from shipped file, merge with user customizations
    defaults_path = Path(__file__).resolve().parent.parent / "data" / "motivation_defaults.json"
    user_path = data_dir / "motivation.json"

    defaults = load_json(defaults_path, {"quotes": []})
    user_data = load_json(user_path, {"quotes": [], "user_custom_quotes": []})

    return {
        "quotes": defaults.get("quotes", []),
        "user_custom_quotes": user_data.get("user_custom_quotes", [])
    }

def save_motivation(data_dir, data):
    save_json(data_dir / "motivation.json", data)

def load_history(data_dir):
    return load_json(data_dir / "history.json", {"completed": [], "failed": [], "cancelled": []})

def save_history(data_dir, history):
    save_json(data_dir / "history.json", history)

# ─── Punishment Actions ───

def action_post_to_beeper_whatsapp(config, recipient_id, message):
    """Send message via Beeper Desktop WhatsApp bridge."""
    settings = config.get("punishments", {}).get("post_to_beeper_whatsapp", {})
    token = settings.get("token", "")
    beeper_url = settings.get("beeper_url", "http://localhost:23373")

    if not token:
        return False, "No token configured for post_to_beeper_whatsapp"

    encoded = urllib.parse.quote(recipient_id, safe="")
    url = f"{beeper_url}/v1/chats/{encoded}/messages"
    headers = {
        "Authorization": f"Bearer {token}",
        "Content-Type": "application/json"
    }
    body = json.dumps({"text": message}).encode()
    req = urllib.request.Request(url, data=body, headers=headers, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=10) as resp:
            return True, "Message sent"
    except Exception as e:
        return False, f"Beeper API error: {e}"

def action_desktop_notification(config, recipient_id, message):
    """Send macOS desktop notification. Always available."""
    try:
        # Escape quotes for osascript
        safe_msg = message.replace('"', '\\"')[:200]
        subprocess.run([
            "osascript", "-e",
            f'display notification "{safe_msg}" with title "Accountability Coach" sound name "Funk"'
        ], capture_output=True, timeout=5)
        return True, "Notification sent"
    except Exception as e:
        return False, f"Notification error: {e}"

def action_health_post_to_beeper_whatsapp(config):
    """Check if Beeper API is reachable."""
    settings = config.get("punishments", {}).get("post_to_beeper_whatsapp", {})
    token = settings.get("token", "")
    beeper_url = settings.get("beeper_url", "http://localhost:23373")

    if not token:
        return False, "No token configured. Run: accountability.py punishment setup post_to_beeper_whatsapp --token <TOKEN>"

    url = f"{beeper_url}/v1/chats/search?q=test"
    headers = {"Authorization": f"Bearer {token}"}
    req = urllib.request.Request(url, headers=headers)
    try:
        with urllib.request.urlopen(req, timeout=5) as resp:
            return True, f"Beeper API reachable (HTTP {resp.status})"
    except Exception as e:
        return False, f"Beeper API unreachable: {e}"

# Registry of punishment actions
PUNISHMENT_ACTIONS = {
    "post_to_beeper_whatsapp": {
        "send": action_post_to_beeper_whatsapp,
        "health": action_health_post_to_beeper_whatsapp,
        "display_name": "WhatsApp (via Beeper Desktop)",
        "required_keys": ["token"],
        "optional_keys": ["beeper_url", "contacts", "default_group"],
    },
    "desktop_notification": {
        "send": action_desktop_notification,
        "health": lambda config: (True, "Always available on macOS"),
        "display_name": "macOS Desktop Notification",
        "required_keys": [],
        "optional_keys": [],
    },
    # Future: post_to_x, email, etc.
}

def execute_punishment(data_dir, action_name, recipient_id, message):
    """Execute a punishment action. Falls back to desktop_notification."""
    config = load_config(data_dir)
    action = PUNISHMENT_ACTIONS.get(action_name)

    if action and action_name != "desktop_notification":
        ok, detail = action["send"](config, recipient_id, message)
        if ok:
            # Always also notify locally
            action_desktop_notification(config, "", message)
            return True, detail
        # Fall through to desktop notification on failure
        print(f"  [warn] {action_name} failed: {detail}. Falling back to desktop notification.", file=sys.stderr)

    return action_desktop_notification(config, "", message)

# ─── Commands: Task ───

def cmd_task_add(args, data_dir):
    config = load_config(data_dir)
    tasks = load_tasks(data_dir)

    task_id = f"task-{tasks['next_id']}"
    tasks["next_id"] += 1

    # Resolve punishment action
    punishment_action = args.punishment or config.get("default_punishment") or "desktop_notification"
    if punishment_action not in PUNISHMENT_ACTIONS:
        output_err("task add", f"Unknown punishment action: {punishment_action}. Available: {', '.join(PUNISHMENT_ACTIONS.keys())}", "UNKNOWN_ACTION")
        return

    # Resolve targets
    targets = []
    if args.targets:
        targets = [t.strip() for t in args.targets.split(",")]
    elif punishment_action == "post_to_beeper_whatsapp":
        p_config = config.get("punishments", {}).get("post_to_beeper_whatsapp", {})
        default_group = p_config.get("default_group", "")
        if default_group:
            targets = [default_group]

    # Resolve secret
    secret_message = None
    if args.secret_id:
        secrets = load_secrets(data_dir)
        for s in secrets["secrets"]:
            if s["id"] == args.secret_id:
                secret_message = s["secret"]
                s["times_used"] = s.get("times_used", 0) + 1
                save_secrets(data_dir, secrets)
                break
        if not secret_message:
            output_err("task add", f"Secret '{args.secret_id}' not found", "SECRET_NOT_FOUND")
            return
    elif args.custom_punishment_message:
        secret_message = args.custom_punishment_message

    now = datetime.now(timezone.utc)
    deadline = now + timedelta(minutes=args.duration)
    duration = args.duration

    # Calculate warning intervals
    warning_intervals = []
    half = duration // 2
    three_q = (duration * 3) // 4
    ten_left = duration - 10
    five_left = duration - 5

    if half > 0:
        warning_intervals.append({"name": "halfway", "minutes_from_start": half, "minutes_remaining": duration - half, "phase": "reminder_early"})
    if three_q > half:
        warning_intervals.append({"name": "75_percent", "minutes_from_start": three_q, "minutes_remaining": duration - three_q, "phase": "reminder_mid"})
    if ten_left > three_q and ten_left > 0:
        warning_intervals.append({"name": "10_min_left", "minutes_from_start": ten_left, "minutes_remaining": 10, "phase": "reminder_late"})
    if five_left > 0 and five_left > max(ten_left, 0):
        warning_intervals.append({"name": "5_min_left", "minutes_from_start": five_left, "minutes_remaining": 5, "phase": "reminder_late"})

    task = {
        "id": task_id,
        "description": args.desc,
        "why": args.why or None,
        "duration_minutes": duration,
        "punishment_action": punishment_action,
        "punishment_message": secret_message,
        "targets": targets,
        "status": "active",
        "created_at": now.isoformat(),
        "deadline": deadline.isoformat(),
        "warning_intervals": warning_intervals,
    }

    tasks["active"][task_id] = task
    save_tasks(data_dir, tasks)

    if _json_mode:
        output_ok("task add", task)
    else:
        print(f"\n  Task created: {task_id}")
        print(f"   Description: {args.desc}")
        if args.why:
            print(f"   Why: {args.why}")
        print(f"   Duration: {duration} min")
        print(f"   Deadline: {deadline.strftime('%H:%M:%S %Z')}")
        print(f"   Punishment: {punishment_action}")
        if targets:
            print(f"   Targets: {len(targets)} recipient(s)")
        print(f"   Warnings: {len(warning_intervals)} scheduled")

def cmd_task_complete(args, data_dir):
    config = load_config(data_dir)
    tasks = load_tasks(data_dir)
    history = load_history(data_dir)

    task = tasks["active"].get(args.task_id)
    if not task:
        # Idempotent: check if already completed
        for t in history.get("completed", []):
            if t["id"] == args.task_id:
                if _json_mode:
                    output_ok("task complete", {"already_completed": True, "task": t})
                else:
                    print(f"Task '{args.task_id}' already completed.")
                return
        output_err("task complete", f"No active task '{args.task_id}'", "TASK_NOT_FOUND")
        return

    # Send all-clear messages
    send_results = []
    if task.get("punishment_action") and task["punishment_action"] != "desktop_notification" and task.get("targets"):
        action = PUNISHMENT_ACTIONS.get(task["punishment_action"])
        if action:
            all_clear_msg = f"Task completed! '{task['description']}' — finished in time. No punishment today -- Accountability Coach"
            for target in task["targets"]:
                ok, detail = action["send"](config, target, all_clear_msg)
                send_results.append({"target": target[:30], "ok": ok, "detail": detail})

    action_desktop_notification(config, "", f"PASSED: {task['description']}")

    task["status"] = "completed"
    task["completed_at"] = datetime.now(timezone.utc).isoformat()
    history["completed"].append(task)
    save_history(data_dir, history)

    del tasks["active"][args.task_id]
    save_tasks(data_dir, tasks)

    if _json_mode:
        output_ok("task complete", {"task": task, "messages_sent": send_results})
    else:
        print(f"\n  Task '{args.task_id}' marked complete!")
        for r in send_results:
            status = "sent" if r["ok"] else f"failed: {r['detail']}"
            print(f"   All-clear → {r['target']}... ({status})")

def cmd_task_fail(args, data_dir):
    config = load_config(data_dir)
    tasks = load_tasks(data_dir)
    history = load_history(data_dir)

    task = tasks["active"].get(args.task_id)
    if not task:
        output_err("task fail", f"No active task '{args.task_id}'", "TASK_NOT_FOUND")
        return

    # Execute punishment
    send_results = []
    punishment_msg = task.get("punishment_message", f"TIME'S UP! Failed to complete: {task['description']}")
    if task.get("targets"):
        for target in task["targets"]:
            ok, detail = execute_punishment(data_dir, task.get("punishment_action", "desktop_notification"), target, punishment_msg)
            send_results.append({"target": target[:30], "ok": ok, "detail": detail})
    else:
        ok, detail = execute_punishment(data_dir, "desktop_notification", "", punishment_msg)
        send_results.append({"target": "local", "ok": ok, "detail": detail})

    task["status"] = "failed"
    task["failed_at"] = datetime.now(timezone.utc).isoformat()
    history["failed"].append(task)
    save_history(data_dir, history)

    del tasks["active"][args.task_id]
    save_tasks(data_dir, tasks)

    if _json_mode:
        output_ok("task fail", {"task": task, "messages_sent": send_results})
    else:
        print(f"\n  Task '{args.task_id}' FAILED. Punishment sent.")
        for r in send_results:
            status = "sent" if r["ok"] else f"failed: {r['detail']}"
            print(f"   Punishment → {r['target']}... ({status})")

def cmd_task_cancel(args, data_dir):
    tasks = load_tasks(data_dir)
    history = load_history(data_dir)

    task = tasks["active"].get(args.task_id)
    if not task:
        output_err("task cancel", f"No active task '{args.task_id}'", "TASK_NOT_FOUND")
        return

    task["status"] = "cancelled"
    task["cancelled_at"] = datetime.now(timezone.utc).isoformat()
    history["cancelled"].append(task)
    save_history(data_dir, history)

    del tasks["active"][args.task_id]
    save_tasks(data_dir, tasks)

    if _json_mode:
        output_ok("task cancel", {"task": task})
    else:
        print(f"  Task '{args.task_id}' cancelled. No punishment sent.")

def cmd_task_status(args, data_dir):
    tasks = load_tasks(data_dir)

    if args.task_id:
        task = tasks["active"].get(args.task_id)
        if not task:
            output_err("task status", f"No active task '{args.task_id}'", "TASK_NOT_FOUND")
            return
        deadline = datetime.fromisoformat(task["deadline"])
        now = datetime.now(timezone.utc)
        remaining_secs = max(0, int((deadline - now).total_seconds()))
        remaining_mins = remaining_secs // 60
        task_with_time = {**task, "remaining_minutes": remaining_mins, "remaining_seconds": remaining_secs}
        if _json_mode:
            output_ok("task status", task_with_time)
        else:
            print(json.dumps(task_with_time, indent=2))
        return

    active = tasks.get("active", {})
    statuses = []
    for tid, t in active.items():
        deadline = datetime.fromisoformat(t["deadline"])
        now = datetime.now(timezone.utc)
        remaining = max(0, int((deadline - now).total_seconds() / 60))
        statuses.append({**t, "remaining_minutes": remaining})

    if _json_mode:
        output_ok("task status", {"tasks": statuses})
    else:
        if not statuses:
            print("No active tasks.")
            return
        for s in statuses:
            icon = "green" if s["remaining_minutes"] > 10 else "yellow" if s["remaining_minutes"] > 5 else "red"
            icons = {"green": "G", "yellow": "Y", "red": "R"}
            print(f"[{icons[icon]}] {s['id']}: {s['description']} — {s['remaining_minutes']} min left")
            if s.get("why"):
                print(f"     Why: {s['why']}")

def cmd_task_list(args, data_dir):
    tasks = load_tasks(data_dir)
    history = load_history(data_dir)

    active = list(tasks.get("active", {}).values())

    if _json_mode:
        result = {"active": active}
        if args.all:
            result["completed"] = history.get("completed", [])[-5:]
            result["failed"] = history.get("failed", [])[-5:]
            result["cancelled"] = history.get("cancelled", [])[-5:]
        output_ok("task list", result)
    else:
        print("=== Active Tasks ===")
        if not active:
            print("  (none)")
        for t in active:
            print(f"  {t['id']}: {t['description']} ({t['duration_minutes']} min)")

        if args.all:
            completed = history.get("completed", [])
            print(f"\n=== Completed ({len(completed)}) ===")
            for t in completed[-5:]:
                print(f"  {t['id']}: {t['description']}")
            failed = history.get("failed", [])
            print(f"\n=== Failed ({len(failed)}) ===")
            for t in failed[-5:]:
                print(f"  {t['id']}: {t['description']}")

def cmd_task_history(args, data_dir):
    history = load_history(data_dir)
    all_items = []
    for t in history.get("completed", []):
        all_items.append({**t, "_result": "completed"})
    for t in history.get("failed", []):
        all_items.append({**t, "_result": "failed"})
    for t in history.get("cancelled", []):
        all_items.append({**t, "_result": "cancelled"})

    all_items.sort(key=lambda x: x.get("created_at", ""), reverse=True)
    limited = all_items[:args.limit]

    if _json_mode:
        output_ok("task history", {"history": limited, "total": len(all_items)})
    else:
        icons = {"completed": "[OK]", "failed": "[FAIL]", "cancelled": "[CANCEL]"}
        for t in limited:
            print(f"{icons.get(t['_result'], '?')} {t['id']}: {t['description']} ({t.get('duration_minutes', '?')}min) — {t['_result']}")

# ─── Commands: Secrets ───

def cmd_secrets_add(args, data_dir):
    data = load_secrets(data_dir)
    next_id = max((int(s["id"].split("-")[1]) for s in data["secrets"]), default=0) + 1

    entry = {
        "id": f"s-{next_id}",
        "secret": args.secret,
        "severity": args.severity or "medium",
        "added_at": datetime.now(timezone.utc).isoformat(),
        "times_used": 0
    }
    data["secrets"].append(entry)
    save_secrets(data_dir, data)

    if _json_mode:
        output_ok("secrets add", entry)
    else:
        print(f"Secret added: {entry['id']} ({entry['severity']})")

def cmd_secrets_list(args, data_dir):
    data = load_secrets(data_dir)
    secrets = data.get("secrets", [])

    if _json_mode:
        output_ok("secrets list", {"secrets": secrets})
    else:
        if not secrets:
            print("No secrets in bank. Add one: accountability.py secrets add --secret '...'")
            return
        for s in secrets:
            sev = {"mild": "[mild]", "medium": "[med]", "spicy": "[SPICY]"}.get(s.get("severity", ""), "[?]")
            print(f"  {s['id']} {sev} (used {s.get('times_used', 0)}x): {s['secret'][:80]}")

def cmd_secrets_pick(args, data_dir):
    data = load_secrets(data_dir)
    secrets = data.get("secrets", [])

    if args.severity:
        secrets = [s for s in secrets if s.get("severity") == args.severity]

    if not secrets:
        output_err("secrets pick", "No matching secrets. Add some first.", "NO_SECRETS")
        return

    # Prefer least-used secrets
    min_used = min(s.get("times_used", 0) for s in secrets)
    candidates = [s for s in secrets if s.get("times_used", 0) == min_used]
    picked = random.choice(candidates)

    if _json_mode:
        output_ok("secrets pick", picked)
    else:
        print(f"Picked: {picked['id']} ({picked['severity']}): {picked['secret']}")

# ─── Commands: Motivation ───

def cmd_motivation_list(args, data_dir):
    data = load_motivation(data_dir)
    all_quotes = data.get("quotes", []) + data.get("user_custom_quotes", [])

    if args.phase:
        all_quotes = [q for q in all_quotes if args.phase in q.get("phase", [])]

    if _json_mode:
        output_ok("motivation list", {"quotes": all_quotes})
    else:
        if not all_quotes:
            print("No quotes found.")
            return
        for q in all_quotes:
            phases = ", ".join(q.get("phase", []))
            print(f"  \"{q['text']}\" — {q.get('attribution', 'Unknown')} [{phases}]")

def cmd_motivation_add(args, data_dir):
    data = load_motivation(data_dir)
    phases = [p.strip() for p in args.phase.split(",")] if args.phase else ["reminder_mid"]

    entry = {
        "text": args.quote,
        "attribution": args.attribution or "Custom",
        "category": "custom",
        "phase": phases
    }

    # Save to user file only
    user_path = data_dir / "motivation.json"
    user_data = load_json(user_path, {"quotes": [], "user_custom_quotes": []})
    user_data["user_custom_quotes"].append(entry)
    save_json(user_path, user_data)

    if _json_mode:
        output_ok("motivation add", entry)
    else:
        print(f"Quote added: \"{args.quote}\" — {entry['attribution']}")

# ─── Commands: Punishment Setup ───

def cmd_punishment_list(args, data_dir):
    config = load_config(data_dir)
    configured = config.get("punishments", {})

    actions = []
    for name, info in PUNISHMENT_ACTIONS.items():
        action_config = configured.get(name, {})
        is_configured = bool(action_config) and all(
            action_config.get(k) for k in info["required_keys"]
        )
        actions.append({
            "name": name,
            "display_name": info["display_name"],
            "configured": is_configured,
            "required_keys": info["required_keys"],
        })

    if _json_mode:
        output_ok("punishment list", {"actions": actions, "default": config.get("default_punishment")})
    else:
        print("Available punishment actions:")
        for a in actions:
            status = "[configured]" if a["configured"] else "[not configured]"
            default = " (default)" if config.get("default_punishment") == a["name"] else ""
            print(f"  {a['name']}: {a['display_name']} {status}{default}")
            if not a["configured"] and a["required_keys"]:
                print(f"    Setup: accountability.py punishment setup {a['name']} --{' --'.join(a['required_keys'])} <value>")

def cmd_punishment_setup(args, data_dir):
    config = load_config(data_dir)
    action_name = args.action_name

    if action_name not in PUNISHMENT_ACTIONS:
        output_err("punishment setup", f"Unknown action: {action_name}. Available: {', '.join(PUNISHMENT_ACTIONS.keys())}", "UNKNOWN_ACTION")
        return

    config.setdefault("punishments", {}).setdefault(action_name, {})

    # Apply key-value pairs from extra args
    if args.settings:
        for setting in args.settings:
            if "=" in setting:
                k, v = setting.split("=", 1)
            elif setting.startswith("--"):
                # Handle --key value format (already parsed by argparse remainder)
                continue
            else:
                continue
            config["punishments"][action_name][k.lstrip("-").replace("-", "_")] = v

    # Handle explicit flags
    if hasattr(args, "token") and args.token:
        config["punishments"][action_name]["token"] = args.token
    if hasattr(args, "beeper_url") and args.beeper_url:
        config["punishments"][action_name]["beeper_url"] = args.beeper_url
    if hasattr(args, "default_group") and args.default_group:
        config["punishments"][action_name]["default_group"] = args.default_group
    if hasattr(args, "add_contact") and args.add_contact:
        name, cid = args.add_contact.split("=", 1)
        config["punishments"][action_name].setdefault("contacts", {})[name.strip()] = cid.strip()

    # Set as default if it's the first configured action
    if not config.get("default_punishment"):
        config["default_punishment"] = action_name

    save_config(data_dir, config)

    if _json_mode:
        output_ok("punishment setup", {"action": action_name, "config": config["punishments"][action_name]})
    else:
        print(f"Configured: {action_name}")
        print(json.dumps(config["punishments"][action_name], indent=2))

def cmd_punishment_health(args, data_dir):
    config = load_config(data_dir)
    action_name = args.action_name

    if action_name not in PUNISHMENT_ACTIONS:
        output_err("punishment health", f"Unknown action: {action_name}", "UNKNOWN_ACTION")
        return

    ok, detail = PUNISHMENT_ACTIONS[action_name]["health"](config)

    if _json_mode:
        output_ok("punishment health", {"action": action_name, "healthy": ok, "detail": detail})
    else:
        status = "[OK]" if ok else "[FAIL]"
        print(f"{status} {action_name}: {detail}")

# ─── Commands: Config ───

def cmd_config_show(args, data_dir):
    config = load_config(data_dir)
    # Redact tokens for display
    display = json.loads(json.dumps(config, default=str))
    for action_name, action_config in display.get("punishments", {}).items():
        if isinstance(action_config, dict) and "token" in action_config:
            token = action_config["token"]
            if len(token) > 8:
                action_config["token"] = token[:8] + "..."

    if _json_mode:
        output_ok("config show", config)  # Full config in JSON mode
    else:
        print(json.dumps(display, indent=2))

def cmd_config_set(args, data_dir):
    config = load_config(data_dir)
    key, value = args.key, args.value

    if key == "default_punishment":
        if value not in PUNISHMENT_ACTIONS:
            output_err("config set", f"Unknown action: {value}", "UNKNOWN_ACTION")
            return
        config["default_punishment"] = value
    else:
        config[key] = value

    save_config(data_dir, config)

    if _json_mode:
        output_ok("config set", {"key": key, "value": value})
    else:
        print(f"Set {key} = {value}")

# ─── Commands: Cleanup ───

def cmd_cleanup(args, data_dir):
    if not args.yes and not _json_mode:
        print("This will cancel all active tasks and remove temp files.")
        print("Run with --yes to confirm, or use --json mode.")
        sys.exit(1)

    tasks = load_tasks(data_dir)
    cancelled_count = len(tasks.get("active", {}))
    tasks["active"] = {}
    save_tasks(data_dir, tasks)

    # Clean temp files
    for f in ["/tmp/accountability-cancel", "/tmp/accountability-ws-events.jsonl"]:
        if os.path.exists(f):
            os.remove(f)

    if _json_mode:
        output_ok("cleanup", {"tasks_cancelled": cancelled_count})
    else:
        print(f"Cleanup done. {cancelled_count} task(s) cancelled.")

# ─── CLI Parser ───

def main():
    parser = argparse.ArgumentParser(
        description="Accountability Coach — motivation through consequences",
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    parser.add_argument("--json", action="store_true", help="Machine-readable JSON output")
    parser.add_argument("--data-dir", help=f"Override data directory (default: {DEFAULT_DATA_DIR})")

    sub = parser.add_subparsers(dest="resource", help="Resource to manage")

    # ─── task ───
    task_parser = sub.add_parser("task", help="Manage accountability tasks")
    task_sub = task_parser.add_subparsers(dest="action", required=True)

    p = task_sub.add_parser("add", help="Create a new accountability task",
        epilog="Examples:\n"
               "  accountability.py task add --desc 'Finish API' --duration 60 --why 'Team is blocked'\n"
               "  accountability.py task add --desc 'Write tests' --duration 30 --punishment post_to_beeper_whatsapp --secret-id s-1 --json",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--desc", required=True, help="Task description")
    p.add_argument("--duration", type=int, default=60, help="Minutes (default: 60)")
    p.add_argument("--why", help="Why this task matters to you (used in motivational reminders)")
    p.add_argument("--punishment", help="Punishment action name (default: from config or desktop_notification)")
    p.add_argument("--targets", help="Comma-separated recipient IDs for the punishment action")
    p.add_argument("--secret-id", help="ID of secret to reveal as punishment (from secrets bank)")
    p.add_argument("--custom-punishment-message", help="Custom punishment message text")

    p = task_sub.add_parser("complete", help="Mark task as completed (sends all-clear)")
    p.add_argument("task_id", help="Task ID (e.g., task-1)")

    p = task_sub.add_parser("fail", help="Mark task as failed (sends punishment)")
    p.add_argument("task_id", help="Task ID")

    p = task_sub.add_parser("cancel", help="Cancel task (no messages sent)")
    p.add_argument("task_id", help="Task ID")

    p = task_sub.add_parser("status", help="Show active tasks with time remaining")
    p.add_argument("task_id", nargs="?", help="Specific task ID (omit for all)")

    p = task_sub.add_parser("list", help="List tasks")
    p.add_argument("--all", action="store_true", help="Include completed/failed/cancelled")

    p = task_sub.add_parser("history", help="Show task history")
    p.add_argument("--limit", type=int, default=10, help="Max results (default: 10)")

    # ─── secrets ───
    secrets_parser = sub.add_parser("secrets", help="Manage embarrassing secrets bank")
    secrets_sub = secrets_parser.add_subparsers(dest="action", required=True)

    p = secrets_sub.add_parser("add", help="Add a secret to the bank",
        epilog="Examples:\n"
               "  accountability.py secrets add --secret 'I cried during a dog food commercial' --severity medium",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--secret", required=True, help="The embarrassing secret")
    p.add_argument("--severity", choices=["mild", "medium", "spicy"], default="medium")

    p = secrets_sub.add_parser("list", help="List all secrets")

    p = secrets_sub.add_parser("pick", help="Pick a secret (prefers least-used)")
    p.add_argument("--severity", choices=["mild", "medium", "spicy"], help="Filter by severity")

    # ─── motivation ───
    motivation_parser = sub.add_parser("motivation", help="Manage motivational quotes")
    motivation_sub = motivation_parser.add_subparsers(dest="action", required=True)

    p = motivation_sub.add_parser("list", help="List quotes")
    p.add_argument("--phase", help="Filter by phase (task_created, reminder_early, reminder_mid, reminder_late, task_completed, task_failed)")

    p = motivation_sub.add_parser("add", help="Add a custom quote",
        epilog="Examples:\n"
               "  accountability.py motivation add --quote 'Ship it or regret it' --attribution 'Me' --phase reminder_late",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("--quote", required=True, help="The quote text")
    p.add_argument("--attribution", help="Who said it")
    p.add_argument("--phase", help="Comma-separated phases: task_created,reminder_early,reminder_mid,reminder_late,task_completed,task_failed")

    # ─── punishment ───
    punishment_parser = sub.add_parser("punishment", help="Configure punishment actions")
    punishment_sub = punishment_parser.add_subparsers(dest="action", required=True)

    punishment_sub.add_parser("list", help="List available punishment actions")

    p = punishment_sub.add_parser("setup", help="Configure a punishment action",
        epilog="Examples:\n"
               "  accountability.py punishment setup post_to_beeper_whatsapp --token abc123 --default-group '!roomid:...'\n"
               "  accountability.py punishment setup post_to_beeper_whatsapp --add-contact 'Alice=!roomid:...'",
        formatter_class=argparse.RawDescriptionHelpFormatter)
    p.add_argument("action_name", help="Action to configure")
    p.add_argument("--token", help="API token")
    p.add_argument("--beeper-url", help="Beeper API URL (default: http://localhost:23373)")
    p.add_argument("--default-group", help="Default group/chat ID")
    p.add_argument("--add-contact", help="Add contact as name=chat_id")
    p.add_argument("settings", nargs="*", help="Additional key=value settings")

    p = punishment_sub.add_parser("health", help="Test connectivity for an action")
    p.add_argument("action_name", help="Action to test")

    # ─── config ───
    config_parser = sub.add_parser("config", help="View/update configuration")
    config_sub = config_parser.add_subparsers(dest="action", required=True)

    config_sub.add_parser("show", help="Show current configuration")

    p = config_sub.add_parser("set", help="Set a config value")
    p.add_argument("key", help="Config key")
    p.add_argument("value", help="Config value")

    # ─── cleanup ───
    p = sub.add_parser("cleanup", help="Cancel all tasks and clean up")
    p.add_argument("--yes", action="store_true", help="Skip confirmation")

    args = parser.parse_args()
    if not args.resource:
        parser.print_help()
        sys.exit(1)

    set_json_mode(args.json)
    data_dir = get_data_dir(args.data_dir)
    data_dir.mkdir(parents=True, exist_ok=True)

    # Route commands
    handlers = {
        ("task", "add"): cmd_task_add,
        ("task", "complete"): cmd_task_complete,
        ("task", "fail"): cmd_task_fail,
        ("task", "cancel"): cmd_task_cancel,
        ("task", "status"): cmd_task_status,
        ("task", "list"): cmd_task_list,
        ("task", "history"): cmd_task_history,
        ("secrets", "add"): cmd_secrets_add,
        ("secrets", "list"): cmd_secrets_list,
        ("secrets", "pick"): cmd_secrets_pick,
        ("motivation", "list"): cmd_motivation_list,
        ("motivation", "add"): cmd_motivation_add,
        ("punishment", "list"): cmd_punishment_list,
        ("punishment", "setup"): cmd_punishment_setup,
        ("punishment", "health"): cmd_punishment_health,
        ("config", "show"): cmd_config_show,
        ("config", "set"): cmd_config_set,
    }

    action = getattr(args, "action", None)
    if args.resource == "cleanup":
        cmd_cleanup(args, data_dir)
    elif (args.resource, action) in handlers:
        handlers[(args.resource, action)](args, data_dir)
    else:
        parser.print_help()
        sys.exit(1)

if __name__ == "__main__":
    main()
