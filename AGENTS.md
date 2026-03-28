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
├── scripts/                # Release and build scripts
├── skills/                 # Product skills (nudge-cli/, nudge-coach/)
├── .agents/skills/         # Development skills (agent behavior when building nudge)
├── .claude/skills/         # Per-skill symlinks → .agents/skills/*/
├── .github/workflows/      # CI (release on tag push)
├── .goreleaser.yaml        # Cross-platform build + Homebrew formula config
├── docs/                   # Documentation
└── AGENTS.md               # This file
```

## Building

```bash
cd cli && go build -o nudge ./cmd/nudge/
```

## Releasing

Releases are built with [GoReleaser](https://goreleaser.com/) — it cross-compiles for macOS/Linux (amd64 + arm64), uploads to GitHub Releases, and pushes a Homebrew formula to `neilsanghrajka/homebrew-tap`.

```bash
# Prerequisites (one-time)
brew install goreleaser gh

# Release a new version
export TAP_GITHUB_TOKEN=ghp_...
./scripts/release.sh v0.2.0

# Redo a botched release
./scripts/release.sh v0.2.0 --redo
```

Alternatively, pushing a `v*` tag triggers the same process via GitHub Actions (`.github/workflows/release.yml`).

## Installing

```bash
# Homebrew
brew install neilsanghrajka/tap/nudge

# curl
curl -sSL https://raw.githubusercontent.com/neilsanghrajka/nudge/main/scripts/install.sh | sh

# Go
go install github.com/neilsanghrajka/nudge/cli/cmd/nudge@latest
```

## CLI Design Principles

When modifying the CLI, follow the `cli-for-agents` skill:
- Every input via flags (non-interactive first)
- `--json` for machine-readable output
- Layered `--help` with real examples
- Fast actionable errors with example invocations
- Idempotent operations where possible
