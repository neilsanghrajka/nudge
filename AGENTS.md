# Agents

This repo is designed to be worked on by AI coding agents. Here's how it's organized for agent consumption.

## Skills

Agent skills live in `.agents/skills/` and are symlinked to `.claude/skills/` for Claude Code compatibility.

| Skill | Purpose |
|-------|---------|
| `cli-for-agents` | Design patterns for agent-friendly CLIs (non-interactive, --json, layered help, examples) |

## Product Skills

Product-level skills that guide how agents should USE the stake CLI (not build it) live in `skills/`:

| Skill | Purpose |
|-------|---------|
| `cli-usage` | Reference for all `stake` CLI commands |
| `coaching` | Psychology-backed motivation strategy (SDT, loss aversion, identity) |
| `verification` | How to verify task completion — what counts as proof, red flags |
| `strictness` | How to handle avoidance — common cheats, when to push back |
| `onboarding` | First-time setup flow for new users |

## Repository Structure

```
stake-ai/
├── cli/                    # Go CLI source (builds to `stake` binary)
├── skills/                 # Product skills (agent behavior when using stake)
├── .agents/skills/         # Development skills (agent behavior when building stake)
├── .claude/skills/         # Symlink → .agents/skills/
├── docs/                   # Documentation
└── AGENTS.md               # This file
```

## Building

```bash
cd cli && go build -o stake ./cmd/stake/
```

## CLI Design Principles

When modifying the CLI, follow the `cli-for-agents` skill:
- Every input via flags (non-interactive first)
- `--json` for machine-readable output
- Layered `--help` with real examples
- Fast actionable errors with example invocations
- Idempotent operations where possible
