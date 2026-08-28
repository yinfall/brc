#!/usr/bin/env bash
set -e

REPO="yinfall/blender-remote-console"
BRC_HOME="$HOME/.brc"
INSTALL_DIR="$BRC_HOME/bin"

echo ">>> Detecting platform and architecture..."
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    linux|darwin) ;;
    *) echo "❌ Unsupported operating system: $OS"; exit 1 ;;
esac

# Try to get latest tag name (using POSIX sed for macOS & Linux compatibility)
echo ">>> Checking for latest release..."
TAG=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" 2>/dev/null | sed -n 's/.*"tag_name": *"\([^"]*\)".*/\1/p' || true)

if [ -n "$TAG" ]; then
    echo "    Found latest version: $TAG"
    EXE_URL="https://github.com/$REPO/releases/download/$TAG/brc-${OS}-${ARCH}"
    ZIP_URL="https://github.com/$REPO/releases/download/$TAG/blender-remote-console.zip"
else
    # Fallback to direct latest release assets (bypasses GitHub API rate limits)
    echo "    (Using latest release assets directly)"
    EXE_URL="https://github.com/$REPO/releases/latest/download/brc-${OS}-${ARCH}"
    ZIP_URL="https://github.com/$REPO/releases/latest/download/blender-remote-console.zip"
fi

mkdir -p "$INSTALL_DIR"

echo ">>> Downloading brc CLI..."
curl -fsSL "$EXE_URL" -o "$INSTALL_DIR/brc"
chmod +x "$INSTALL_DIR/brc"

echo ">>> Downloading Blender addon package..."
curl -fsSL "$ZIP_URL" -o "$BRC_HOME/blender-remote-console.zip"

echo ">>> Configuring Blender addon..."
"$INSTALL_DIR/brc" install-addon --all

echo ""
echo "✓ Installation complete!"
echo "1. Add '$INSTALL_DIR' to your PATH in ~/.bashrc or ~/.zshrc if not already present:"
echo "   export PATH=\"\$HOME/.brc/bin:\$PATH\""
echo "2. Open Blender -> Edit -> Preferences -> Add-ons, and enable 'Blender Remote Console'"
