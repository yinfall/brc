# Blender Remote Console (`brc`)

[![Release](https://img.shields.io/github/v/release/yinfall/brc?color=blue)](https://github.com/yinfall/brc/releases)
[![License: MIT](https://img.shields.io/badge/CLI_License-MIT-green.svg)](LICENSE)
[![License: GPL v3](https://img.shields.io/badge/Addon_License-GPL_v3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0.html)
[![Blender](https://img.shields.io/badge/Blender-4.x%20%7C%205.x-orange.svg)](https://www.blender.org/)
[![Platform](https://img.shields.io/badge/Platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey.svg)]()

> **Control Blender directly from your terminal.** Send Python commands and get instant output. Designed for **AI Agents**, command-line workflows, and automation scripts.

---

## 📦 Quick Install

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/yinfall/brc/main/scripts/install.ps1 | iex
```

### Linux / macOS (Bash)
```bash
curl -fsSL https://raw.githubusercontent.com/yinfall/brc/main/scripts/install.sh | bash
```

> **First-time setup**: Run `brc install-addon` to configure your Blender, then enable **Blender Remote Console** in Blender's **Preferences -> Add-ons**.

---

## ⚡ Usage & AI Agent Integration

`brc` allows AI agents and scripts to inspect and manipulate running Blender scenes directly from the command line:

```bash
# 1. Check connected Blender instances
brc sessions
# Output: 12345   device (Blender Remote Console)

# 2. Run Python code and get output
brc exec "print(bpy.context.scene.name)"
# Output: Scene

# 3. Query scene data (returns result directly)
brc exec "bpy.data.objects.keys()"
# Output: ['Camera', 'Cube', 'Light']

# 4. Execute a script file
brc exec script.py

# 5. Target a specific Blender instance (when multiple are open)
brc -s 12345 exec "bpy.ops.mesh.primitive_cube_add()"
```

---

## 🤖 Why use `brc` with AI Agents?

- **Zero-friction**: Background service starts automatically on demand.
- **Crash-proof**: Commands execute safely on Blender's main thread.
- **Stateful**: Variables and imports persist across multiple `exec` commands.
- **Clean output**: Standard output, return values, and error stack traces are returned directly to the shell.

---

## 📚 Documentation

- [Developer & Architecture Guide](docs/development.md): Protocol details and local development.

---

## 📄 License

This project is licensed under a hybrid open-source license model:

- **`blender-addon/`**: Licensed under the [GNU General Public License v3.0 (GPLv3)](https://www.gnu.org/licenses/gpl-3.0.html) to comply with Blender's licensing terms.
- **`cli/` (`brc` binary) & `scripts/`**: Licensed under the permissive [MIT License](LICENSE).
