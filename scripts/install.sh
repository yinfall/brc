#!/usr/bin/env bash
set -e

REPO="yinfall/brc"
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
curl -fsSL "$EXE_URL" -o "$INSTALL_DIR/brc.tmp"
chmod +x "$INSTALL_DIR/brc.tmp"

# On macOS, clear quarantine and re-apply local ad-hoc signature to prevent SIGKILL on Apple Silicon
if [ "$OS" = "darwin" ]; then
    xattr -c "$INSTALL_DIR/brc.tmp" 2>/dev/null || true
    codesign --force --deep --sign - "$INSTALL_DIR/brc.tmp" 2>/dev/null || true
fi

mv -f "$INSTALL_DIR/brc.tmp" "$INSTALL_DIR/brc"

echo ">>> Downloading Blender addon package..."
curl -fsSL "$ZIP_URL" -o "$BRC_HOME/blender-remote-console.zip.tmp"
mv -f "$BRC_HOME/blender-remote-console.zip.tmp" "$BRC_HOME/blender-remote-console.zip"

echo ""
echo "🎉 brc CLI installed successfully to $INSTALL_DIR/brc"
echo ""

# Check if PATH contains ~/.brc/bin, if not print advice
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *)
        echo "💡 Note: Add '$INSTALL_DIR' to your PATH in ~/.bashrc or ~/.zshrc to use 'brc' anywhere:"
        echo "   export PATH=\"\$HOME/.brc/bin:\$PATH\""
        echo ""
        ;;
esac

# Prompt whether to run brc install-addon now
USER_INPUT=""
PROMPT="👉 Would you like to run 'brc install-addon' to configure Blender now? [Y/n]: "

if [ -t 0 ]; then
    read -r -p "$PROMPT" USER_INPUT
elif [ -c /dev/tty ]; then
    read -r -p "$PROMPT" USER_INPUT </dev/tty
else
    USER_INPUT="n"
fi

if [[ -z "$USER_INPUT" || "$USER_INPUT" =~ ^[Yy]$ ]]; then
    echo ""
    if [ -c /dev/tty ]; then
        "$INSTALL_DIR/brc" install-addon </dev/tty
    else
        "$INSTALL_DIR/brc" install-addon
    fi
else
    echo ""
    echo "You can run 'brc install-addon' anytime later to configure your Blender versions."
fi
