#!/bin/sh
# Install script for nudge CLI
# Usage: curl -sSL https://raw.githubusercontent.com/neilsanghrajka/nudge/main/scripts/install.sh | sh

set -e

REPO="neilsanghrajka/nudge"
BINARY="nudge"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  darwin) OS="darwin" ;;
  linux)  OS="linux" ;;
  *)
    echo "Error: Unsupported OS: $OS"
    echo "  Supported: macOS (darwin), Linux"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  amd64)   ARCH="amd64" ;;
  arm64)   ARCH="arm64" ;;
  aarch64) ARCH="arm64" ;;
  *)
    echo "Error: Unsupported architecture: $ARCH"
    echo "  Supported: x86_64 (amd64), arm64 (aarch64)"
    exit 1
    ;;
esac

# Get latest release tag
echo "Fetching latest release..."
TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | head -1 | sed 's/.*"tag_name": *"//;s/".*//')

if [ -z "$TAG" ]; then
  echo "Error: Could not determine latest release."
  echo "  Check: https://github.com/${REPO}/releases"
  exit 1
fi

echo "Latest release: ${TAG}"

# Download
URL="https://github.com/${REPO}/releases/download/${TAG}/${BINARY}_${OS}_${ARCH}.tar.gz"
TMP=$(mktemp -d)
echo "Downloading ${BINARY}_${OS}_${ARCH}.tar.gz..."

if ! curl -sSL "$URL" -o "${TMP}/${BINARY}.tar.gz"; then
  echo "Error: Download failed."
  echo "  URL: $URL"
  rm -rf "$TMP"
  exit 1
fi

# Extract
tar -xzf "${TMP}/${BINARY}.tar.gz" -C "$TMP"

if [ ! -f "${TMP}/${BINARY}" ]; then
  echo "Error: Binary not found in archive."
  rm -rf "$TMP"
  exit 1
fi

chmod +x "${TMP}/${BINARY}"

# Install
INSTALL_DIR="/usr/local/bin"
if [ -w "$INSTALL_DIR" ]; then
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
elif command -v sudo >/dev/null 2>&1; then
  echo "Installing to ${INSTALL_DIR} (requires sudo)..."
  sudo mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
  INSTALL_DIR="${HOME}/.local/bin"
  mkdir -p "$INSTALL_DIR"
  mv "${TMP}/${BINARY}" "${INSTALL_DIR}/${BINARY}"
  echo "Installed to ${INSTALL_DIR}/${BINARY}"
  echo "Make sure ${INSTALL_DIR} is in your PATH."
fi

rm -rf "$TMP"

# Verify
echo ""
echo "Installed successfully!"
${INSTALL_DIR}/${BINARY} version
