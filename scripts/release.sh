#!/bin/sh
# Release nudge CLI — builds binaries, creates GitHub release, pushes Homebrew formula
#
# Usage:
#   ./scripts/release.sh v0.1.0          # release a specific version
#   ./scripts/release.sh v0.1.0 --redo   # delete existing release and redo it
#
# Prerequisites:
#   brew install goreleaser gh
#   gh secret set TAP_GITHUB_TOKEN       # store PAT in GitHub repo secrets
#   (or export TAP_GITHUB_TOKEN=ghp_...) # alternatively, set it manually

set -e

VERSION="${1:-}"
REDO="${2:-}"

if [ -z "$VERSION" ]; then
  echo "Usage: ./scripts/release.sh <version> [--redo]"
  echo ""
  echo "Examples:"
  echo "  ./scripts/release.sh v0.2.0          # new release"
  echo "  ./scripts/release.sh v0.2.0 --redo   # delete and redo existing release"
  exit 1
fi

# Validate version format
case "$VERSION" in
  v[0-9]*) ;;
  *)
    echo "Error: Version must start with 'v' (e.g., v0.1.0)"
    exit 1
    ;;
esac

# Check prerequisites
for cmd in goreleaser gh git; do
  if ! command -v "$cmd" >/dev/null 2>&1; then
    echo "Error: $cmd is not installed."
    echo "  brew install $cmd"
    exit 1
  fi
done

# Get TAP_GITHUB_TOKEN from GitHub secrets via gh CLI if not already set
if [ -z "$TAP_GITHUB_TOKEN" ]; then
  echo "TAP_GITHUB_TOKEN not set, fetching from GitHub environment..."
  TAP_GITHUB_TOKEN=$(gh secret list -R neilsanghrajka/nudge 2>/dev/null | grep -q TAP_GITHUB_TOKEN && gh auth token 2>/dev/null || true)
  if [ -z "$TAP_GITHUB_TOKEN" ]; then
    # Fall back to gh auth token — works if the gh CLI PAT has repo scope on homebrew-tap
    TAP_GITHUB_TOKEN=$(gh auth token 2>/dev/null || true)
  fi
  if [ -z "$TAP_GITHUB_TOKEN" ]; then
    echo "Error: Could not get TAP_GITHUB_TOKEN."
    echo "  Either: export TAP_GITHUB_TOKEN=ghp_..."
    echo "  Or:     gh auth login (with a token that has repo scope)"
    exit 1
  fi
  export TAP_GITHUB_TOKEN
  echo "Using token from gh auth."
fi

# Check for uncommitted changes
if [ -n "$(git status --porcelain)" ]; then
  echo "Error: You have uncommitted changes. Commit or stash them first."
  git status --short
  exit 1
fi

# Handle --redo: delete existing release and tag
if [ "$REDO" = "--redo" ]; then
  echo "Deleting existing release ${VERSION}..."
  gh release delete "$VERSION" --yes 2>/dev/null || true
  git tag -d "$VERSION" 2>/dev/null || true
  git push origin ":refs/tags/${VERSION}" 2>/dev/null || true
  echo "Cleaned up ${VERSION}"
fi

# Tag
echo ""
echo "Tagging ${VERSION}..."
git tag "$VERSION"
git push origin "$VERSION"

# Release
echo ""
echo "Running GoReleaser..."
goreleaser release --clean

echo ""
echo "Release ${VERSION} complete!"
echo ""
echo "Install methods:"
echo "  brew install neilsanghrajka/tap/nudge"
echo "  curl -sSL https://raw.githubusercontent.com/neilsanghrajka/nudge/main/install.sh | sh"
echo "  go install github.com/neilsanghrajka/nudge/cli/cmd/nudge@latest"
