# Agents

This repo is designed to be worked on by AI coding agents. Here's how it's organized for agent consumption.

## Skills

Agent skills live in `.agents/skills/` and are symlinked to `.claude/skills/` for Claude Code compatibility.

| Skill | Purpose |
|-------|---------|
| `cli-for-agents` | Design patterns for agent-friendly CLIs (non-interactive, --json, layered help, examples) |

## Product Skills

Product-level skills that guide how agents should USE the nudge CLI (not build it) live in `skills/<skill-name>/SKILL.md`:

| Skill | Purpose |
|-------|---------|
| `nudge-cli` | CLI commands, flags, onboarding, and first-time setup |
| `nudge-coach` | Coaching psychology, messaging, verification, and strictness |

## Repository Structure

```
nudge/
├── cli/                    # Go CLI source (builds to `nudge` binary)
├── skills/                 # Product skills (nudge-cli/, nudge-coach/)
├── .agents/skills/         # Development skills (agent behavior when building nudge)
├── .claude/skills/         # Symlink → .agents/skills/
├── docs/                   # Documentation
└── AGENTS.md               # This file
```

## Building

```bash
cd cli && go build -o nudge ./cmd/nudge/
```

## CLI Design Principles

When modifying the CLI, follow the `cli-for-agents` skill:
- Every input via flags (non-interactive first)
- `--json` for machine-readable output
- Layered `--help` with real examples
- Fast actionable errors with example invocations
- Idempotent operations where possible
