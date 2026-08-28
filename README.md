# Blender Remote Console

**Blender Remote Console (BRC)** 是一款专为 Blender 4.x / 5.x 设计的远程 Python 代码执行工具。它采用了类似于 Android `adb` 的守护进程架构，让你可以随时在外部命令行、脚本或自动化流水线中向正在运行的 Blender 发送指令并实时获取完整的返回输出。

---

## 🌟 核心特性

- 🚀 **极简类 `adb` 交互**：
  - **`brc sessions`**：一键发现并列出本机所有活跃的 Blender 会话及其进程 PID。
  - **`brc exec <代码|文件>`**：单会话自动识别，多会话通过 `-s <PID>` 精准路由。
- 🔄 **零配置自动保活**：Blender 插件在后台始终自动寻找并连接守护进程，中途守护进程断开或未启动会自动静默拉起。
- 🔒 **100% 主线程安全**：所有远程 Python 代码均通过 Blender 主线程调度队列执行，杜绝非主线程调用 `bpy.ops` 引发的软件崩溃。
- 🐍 **完整的交互式 Console 体验**：
  - 支持 **全局变量持久化**（上一条命令定义的变量、类、函数在后续调用中依然有效）。
  - 支持 **REPL 表达式求值**（例如输入 `bpy.data.objects.keys()` 即可直接回传返回值）。
  - 完整捕获 `stdout` 打印、`stderr` 报错堆栈及返回值。
- 🎨 **可视化 N-Panel 侧边栏**：3D Viewport 侧边栏提供实时连接状态、一键命令复制与可滚动的终端执行日志。

---

## 📦 一键安装 (One-Line Install)

选择你的操作系统，在终端运行以下一行命令即可自动完成 `brc` CLI 的安装：

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/yinfall/blender-remote-console/main/scripts/install.ps1 | iex
```

### Linux / macOS (Bash)
```bash
curl -fsSL https://raw.githubusercontent.com/yinfall/blender-remote-console/main/scripts/install.sh | bash
```

> **手动安装 / 离线安装**：
> 在 [Releases](https://github.com/yinfall/blender-remote-console/releases) 页面下载对应系统的 `brc` 可执行文件加入 `PATH`，以及 `blender-remote-console.zip` 放入 `~/.brc/` 即可。

---

## ⚡ 快速上手指南

### 1. 配置 Blender 插件 (`brc install-addon`)
安装完成后，在终端运行 `brc install-addon`，即可交互式选择要安装的 Blender 版本：
```bash
brc install-addon
```
安装后在 Blender 中打开 **Edit -> Preferences -> Add-ons** 启用 **Blender Remote Console**。

### 2. 环境与连接自检 (`brc doctor`)
```bash
brc doctor
```
**输出：**
```text
=== Blender Remote Console Doctor ===
✓ CLI Binary: ...
ℹ️ Daemon: Not running (will start automatically on first `brc exec` or `brc sessions`)

Blender Installations & Addon Status:
  ✓ [Installed]   .../Blender/4.5/scripts/addons/blender-remote-console
```

### 2. 查看活跃的 Blender 会话 (`brc sessions`)
```bash
brc sessions
```
**输出：**
```text
List of Blender sessions attached
12345    device (Blender Remote Console)
```
*(注：左侧数字为 Blender 对应进程的 PID)*

### 2. 执行单行 Python 命令
```bash
brc exec "print(bpy.context.scene.name)"
```
**输出：**
```text
Scene
```

### 3. 获取表达式返回值
```bash
brc exec "bpy.data.objects.keys()"
```
**输出：**
```text
['Camera', 'Cube', 'Light']
```

### 4. 传输并执行 `.py` 脚本文件
```bash
brc exec path/to/my_script.py
```

### 5. 多会话指定目标 (`-s` / `--session`)
当本机同时打开了多个 Blender 窗口时，可通过 `-s <PID>` 将指令发送给特定的实例：
```bash
brc -s 12345 exec "bpy.ops.mesh.primitive_cube_add()"
```

---

## 🖥️ Blender 界面功能 (N-Panel)

在 3D Viewport 视图中按下 **`N`** 键展开侧边栏，点击 **Remote Console** 选项卡：

| 面板区域 | 功能说明 |
| :--- | :--- |
| **Status (状态栏)** | 动态显示连接状态与当前进程 PID。异常时展示直观的错误提示。 |
| **brc Quick Commands** | **Copy Short Cmd** / **Copy Script Cmd**：一键复制携带当前 PID 的命令行示例到剪贴板。 |
| **Runtime Terminal (执行日志)** | 实时显示外部执行的代码（`>>>`）、标准输出及报错异常，配有一键清空和一键复制日志按钮。 |
| **Reset Namespace** | 点击 **Reset Console Globals** 可一键重置 Python 上下文环境，清空之前定义的临时变量。 |

---

## 📚 开发者文档

如果你想了解底层协议设计、本地开发调试流程或参与贡献，请参考：
* [开发者与架构设计文档 (docs/development.md)](docs/development.md)

---

## 📄 开源许可

本项目基于 MIT License 开源。
