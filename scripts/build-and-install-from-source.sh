#!/usr/bin/env bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
BRC_HOME="$HOME/.brc"
INSTALL_DIR="$BRC_HOME/bin"

echo "=== Installing Blender Remote Console locally from source ==="
echo "Project Root: $PROJECT_ROOT"

# 1. Ensure ~/.brc/bin directory exists
mkdir -p "$INSTALL_DIR"

# 2. Build Go CLI directly to ~/.brc/bin/brc
echo ""
echo ">>> [1/3] Building brc locally from source..."
cd "$PROJECT_ROOT/cli"
go build -ldflags="-s -w" -o "$INSTALL_DIR/brc" .
chmod +x "$INSTALL_DIR/brc"
echo "✓ Successfully built: $INSTALL_DIR/brc"

# 3. Package local blender-addon directory to ~/.brc/blender-remote-console.zip
echo ""
echo ">>> [2/3] Packaging local addon files into zip..."
cd "$PROJECT_ROOT/blender-addon"
rm -f "$BRC_HOME/blender-remote-console.zip"
zip -r -q "$BRC_HOME/blender-remote-console.zip" .
echo "✓ Successfully created: $BRC_HOME/blender-remote-console.zip"

# 4. Run brc install-addon
echo ""
echo ">>> [3/3] Running 'brc install-addon'..."
"$INSTALL_DIR/brc" install-addon

echo ""
echo "✓ Local installation & setup completed!"
echo "Make sure '$INSTALL_DIR' is in your PATH in ~/.bashrc or ~/.zshrc:"
echo "   export PATH=\"\$HOME/.brc/bin:\$PATH\""
