# Blender Remote Console

**Blender Remote Console** 是一款专门为 Blender 4.x 开发的高效远程 Python 代码执行插件。该项目采用了类似 `adb` 的守护进程 (Daemon) 架构。

项目内置了用 Go 语言编写的极简命令行工具 **`brc`**（位于 `cli/` 目录下），你可以使用 `brc` 轻松发送**简短 Python 命令**或**传输完整的 `.py` 脚本文件**，代码完全在 Blender 内部主线程环境运行，并**实时返回完整的输出信息**（包含 `print()` 打印内容、错误异常 Traceback 及表达式返回值）。

---

## 🌟 核心优势

- 🚀 **`brc` CLI 工具与守护进程架构（类 `adb` 体验）**：
  - **永远自动保活重连**：插件启动后会自动寻找并连接后台守护进程 (`127.0.0.1:8082`)，中途断开会自动重连，如果守护进程未启动还会自动尝试静默拉起。
  - **`brc sessions`**：参照 `adb devices`，快速发现并列出本机所有活跃的 Blender 会话 (PID)。
  - **精确会话投递**：使用 `brc -s <pid> exec <code>` 精准将代码投递给指定的 Blender 进程。
- 🔒 **主线程安全执行**：通过 Blender `bpy.app.timers` 调度队列，所有远程 Python 代码均 100% 在 Blender 主线程中安全运行，避免非主线程调用 `bpy.ops` 导致的崩溃。
- 🐍 **Console 级上下文**：
  - 默认自动注入 `bpy`, `os`, `sys`, `math` 等核心库。
  - **全局变量持久化**：上一条命令定义的变量或函数，在后续请求中均可继续调用。
  - **REPL 表达式求值**：当末行为表达式（如 `bpy.context.object` 或 `bpy.data.scenes.keys()`）时，自动返回求值结果。
- 📊 **完整输出捕获**：完整捕获 `stdout`、`stderr` 及返回值，实时返回给命令行终端。
- 🎨 **精美 N-Panel 界面**：在 3D Viewport 侧边栏提供可视化状态展示、快捷命令复制以及执行日志滚动查看面板。

---

## 📦 一键安装 (One-Line Install)

发布后，用户可以通过以下一行命令自动安装 `brc` CLI 与 Blender 插件：

### Windows (PowerShell)
```powershell
irm https://raw.githubusercontent.com/yinfall/blender-remote-console/main/scripts/install.ps1 | iex
```

### Linux / macOS (Bash)
```bash
curl -fsSL https://raw.githubusercontent.com/yinfall/blender-remote-console/main/scripts/install.sh | bash
```

---

## ⚡ 命令行工具 `brc` 使用指南

安装后确保 `brc` 位于环境变量 `PATH` 中：

### 1. 查看已连接的 Blender 会话 (`brc sessions`)
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

### 3. 传输并执行 `.py` 脚本文件
直接将文件路径传给 `brc exec` 即可：
```bash
brc exec my_script.py
```

### 4. 多会话选择 (`-s` / `--session`)
当本机开启了多个 Blender 窗口时，需要通过 PID 指定特定的进程：
```bash
brc -s 12345 exec "print('Hello Session 12345')"
```

---

## 🖥️ N-Panel 侧边栏功能说明

| 区域 | 功能描述 |
| :--- | :--- |
| **Status (状态栏)** | 动态显示当前连接状态及本进程 PID。异常时下方会展示红色错误提示框。 |
| **brc Quick Commands** | **Copy Short Cmd** / **Copy Script Cmd**：一键复制包含了当前 PID 的 `brc` 命令模板到剪贴板。 |
| **Runtime Terminal (终端日志)** | **可滚动的终端日志视窗**，像命令行一样实时显示执行的代码（`>>>`）、标准输出及报错，配有一键复制和一键清空日志按钮。 |
| **Reset Namespace** | 点击 **Reset Console Globals** 可一键重置 Python 上下文环境，清空之前创建的全局变量。 |

---

## 📁 项目源码结构

```
blender-remote-console/
├── blender-addon/             # 【Blender 插件目录】
│   ├── __init__.py            # 插件入口与持久化保活定时器
│   ├── client.py              # 守护进程客户端，维持 TCP 长连接与主线程调度
│   ├── executor.py            # Python 代码执行器（REPL 表达式求值与 stdout/stderr 捕获）
│   ├── operators.py           # Blender Operators（重置命名空间、复制命令等）
│   └── ui.py                  # 3D Viewport 侧边栏 (N-Panel) 界面定义
│
├── cli/                       # 【CLI 与 Daemon 核心源码 (Go)】
│   ├── main.go                # Daemon 后台服务与 brc CLI 实现
│   └── go.mod
│
├── scripts/                   # 【安装与部署脚本】
│   ├── install.ps1            # Windows 一键安装脚本
│   └── install.sh             # Linux/macOS 一键安装脚本
│
├── .github/
│   └── workflows/
│       └── release.yml        # GitHub Actions 自动交叉编译与发版
│
├── .gitignore
└── README.md
```
