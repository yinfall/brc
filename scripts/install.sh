#!/usr/bin/env bash
set -e

REPO="your-username/blender-remote-console"
INSTALL_DIR="$HOME/.brc/bin"
RELEASE_API="https://api.github.com/repos/$REPO/releases/latest"

echo ">>> Fetching latest release info for $REPO..."
TAG=$(curl -s "$RELEASE_API" | grep -Po '"tag_name": "\K.*?(?=")' || true)

if [ -z "$TAG" ]; then
    echo "Failed to fetch latest release. Please ensure releases are published on GitHub."
    exit 1
fi

OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

mkdir -p "$INSTALL_DIR"
EXE_URL="https://github.com/$REPO/releases/download/$TAG/brc-${OS}-${ARCH}"

echo ">>> Downloading brc CLI ($TAG)..."
curl -fsSL "$EXE_URL" -o "$INSTALL_DIR/brc"
chmod +x "$INSTALL_DIR/brc"

# Download addon zip
TEMP_ZIP="/tmp/blender-remote-console.zip"
curl -fsSL "https://github.com/$REPO/releases/download/$TAG/blender-remote-console.zip" -o "$TEMP_ZIP"

if [ "$OS" = "darwin" ]; then
    BLENDER_BASE="$HOME/Library/Application Support/Blender"
else
    BLENDER_BASE="$HOME/.config/blender"
fi

if [ -d "$BLENDER_BASE" ]; then
    for ver_dir in "$BLENDER_BASE"/*; do
        if [ -d "$ver_dir" ] && [[ "$(basename "$ver_dir")" =~ ^[0-9]+\.[0-9]+ ]]; then
            ADDON_DIR="$ver_dir/scripts/addons/blender-remote-console"
            mkdir -p "$ADDON_DIR"
            unzip -o -q "$TEMP_ZIP" -d "$ADDON_DIR"
            echo "    Installed to Blender $(basename "$ver_dir")"
        fi
    done
else
    echo "    [Warning] Blender config folder not found at $BLENDER_BASE. Please manually extract the addon."
fi
rm -f "$TEMP_ZIP"

echo ""
echo "✓ Installation complete!"
echo "1. Add '$INSTALL_DIR' to your PATH in ~/.bashrc or ~/.zshrc if not already present:"
echo "   export PATH=\"\$HOME/.brc/bin:\$PATH\""
echo "2. Open Blender -> Edit -> Preferences -> Add-ons, and enable 'Blender Remote Console'"
