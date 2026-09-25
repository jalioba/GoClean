#!/usr/bin/env bash
# GoClean cross-platform installer for Linux and macOS
# Usage: curl -fsSL https://raw.githubusercontent.com/jalioba/GoClean/main/install.sh | bash

set -e

REPO="jalioba/GoClean"
BINARY_NAME="goclean"

# 1. Detect OS
OS="$(uname -s)"
case "$OS" in
  Linux*)  TARGET_OS="linux" ;;
  Darwin*) TARGET_OS="darwin" ;;
  *)
    echo "❌ Unsupported operating system: $OS"
    exit 1
    ;;
esac

# 2. Detect Architecture
ARCH="$(uname -m)"
case "$ARCH" in
  x86_64|amd64) TARGET_ARCH="amd64" ;;
  aarch64|arm64) TARGET_ARCH="arm64" ;;
  *)
    echo "❌ Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "🔍 Detected platform: ${TARGET_OS}_${TARGET_ARCH}"

# 3. Fetch latest release version
echo "📡 Checking latest release from GitHub..."
LATEST_TAG=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" 2>/dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/' || true)

if [ -z "$LATEST_TAG" ]; then
  echo "⚠️  Could not determine latest release via GitHub API, falling back to git..."
  LATEST_TAG="v1.0.0"
fi

VERSION="${LATEST_TAG#v}"
TARBALL="${BINARY_NAME}_${VERSION}_${TARGET_OS}_${TARGET_ARCH}.tar.gz"
DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${LATEST_TAG}/${TARBALL}"

# 4. Download and extract
TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT

echo "⬇️  Downloading GoClean (${LATEST_TAG})..."
if ! curl -sSL --fail "$DOWNLOAD_URL" -o "${TMP_DIR}/${TARBALL}"; then
  echo "❌ Download failed from: $DOWNLOAD_URL"
  echo "Please check if the release asset exists for your platform."
  exit 1
fi

tar -xzf "${TMP_DIR}/${TARBALL}" -C "$TMP_DIR"
chmod +x "${TMP_DIR}/${BINARY_NAME}"

# 5. Determine destination directory
INSTALL_DIR="/usr/local/bin"
if [ ! -w "$INSTALL_DIR" ]; then
  if command -v sudo >/dev/null 2>&1 && [ -t 0 ]; then
    echo "🔐 Requesting sudo permissions to install to $INSTALL_DIR..."
    sudo mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
  else
    INSTALL_DIR="$HOME/.local/bin"
    mkdir -p "$INSTALL_DIR"
    mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
    
    # Check if ~/.local/bin is in PATH
    if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
      echo "⚠️  $INSTALL_DIR is not in your PATH."
      echo "   Add it with: export PATH=\"\$HOME/.local/bin:\$PATH\""
    fi
  fi
else
  mv "${TMP_DIR}/${BINARY_NAME}" "${INSTALL_DIR}/${BINARY_NAME}"
fi

echo "✅ GoClean ${LATEST_TAG} installed successfully to ${INSTALL_DIR}/${BINARY_NAME}!"
echo "🚀 Run 'goclean --help' or 'goclean -i' to get started."
