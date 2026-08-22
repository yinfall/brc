# Blender Remote Console

**Blender Remote Console** 是一款专门为 Blender 4.x 开发的高效远程 Python 代码执行插件。启用后会在 Blender 侧启动一个 HTTP 服务器（支持在 3D Viewport 的 N-Panel 侧边栏开启/关闭监听）。

项目内置了参照 `adb` 设计的极简命令行工具 **`brc`**（位于 `bin/` 目录下），你可以使用 `brc` 或 `curl` 轻松发送**简短 Python 命令**或**传输完整的 `.py` 脚本文件**，代码完全在 Blender 内部环境运行，并**实时返回完整的输出信息**（包含 `print()` 打印内容、错误异常 Traceback 及表达式返回值）。

---

## 🌟 核心优势

- 🚀 **`brc` CLI 工具（类 `adb` 体验）**：
  - **自动 Session 感知**：无需手动敲 `curl` 端口！若本机仅运行 1 个 Blender，`brc` 自动识别连接；若运行多个 Blender，像 `adb` 一样精准报错引导使用 `-s <port>`。
  - **`brc sessions`**：参照 `adb devices`，快速发现并列出本机所有活跃的 Blender 会话。
- 🔒 **主线程安全执行**：通过 Blender `bpy.app.timers` 调度队列，所有远程 Python 代码均 100% 在 Blender 主线程中安全运行，避免非主线程调用 `bpy.ops` 导致的崩溃。
- 🐍 **Console 级上下文**：
  - 默认自动注入 `bpy`, `os`, `sys`, `math` 等核心库。
  - **全局变量持久化**：上一条命令定义的变量或函数，在后续请求中均可继续调用。
  - **REPL 表达式求值**：当末行为表达式（如 `bpy.context.object` 或 `bpy.data.scenes.keys()`）时，自动返回求值结果。
- 📊 **完整输出捕获**：完整捕获 `stdout`、`stderr` 及返回值，实时返回给命令行终端。

- 🎨 **精美 N-Panel 界面**：在 3D Viewport 侧边栏提供可视化状态展示、Host/Port 端口设置、Auth Token 校验及一键复制工具。

---

## ⚡ 命令行工具 `brc` 使用指南 (`bin/`)

可以直接将插件目录下的 `bin` 文件夹添加至系统 `PATH` 环境变量，或直接调用 `.\bin\brc`。

### 1. 查看已连接的 Blender 会话 (`brc sessions`)
```bash
brc sessions
```
**输出：**
```text
List of Blender sessions attached
127.0.0.1:8182    device (Blender Remote Console)
```

### 2. 执行单行 Python 命令
```bash
brc "print(bpy.context.scene.name)"
```
**输出：**
```text
Scene
```

### 3. 传输并执行 `.py` 脚本文件
无需配置 HTTP POST，直接将文件路径传给 `brc` 即可：
```bash
brc my_script.py
```

### 4. 多会话选择 (`-s` / `--session`)
当本机开启了多个 Blender 窗口（例如端口 `8182` 和 `8183`）：
```bash
# 指定端口执行
brc -s 8183 "print('Hello Session 8183')"
```
*注：若有多个 Session 且未指定 `-s`，`brc` 将像 `adb` 一样弹出提示并列出所有可用 session。*

---

## 📦 安装与启用

### 1. 安装插件
1. 将 `blender-remote-console` 文件夹复制到 Blender 的 Addons 目录中：
   - **Windows**: `%USERPROFILE%\AppData\Roaming\Blender Foundation\Blender\4.x\scripts\addons\`
2. 打开 Blender，前往 **Edit -> Preferences -> Add-ons**。
3. 搜索 **Blender Remote Console** 并勾选启用。

### 2. 开启监听
1. 在 3D Viewport 视图中按下 **`N`** 键展开侧边栏 (Sidebar)。
2. 点击 **Remote Console** 选项卡。
3. 设置监听 Host（默认 `127.0.0.1`）、Port（默认 `8182`）和可选的 Auth Token。
4. 点击 **Start Server** 按钮，显示 `Running (127.0.0.1:8182)` 即代表启动成功。

---

## 🖥️ N-Panel 侧边栏功能说明

| 区域 | 功能描述 |
| :--- | :--- |
| **Status (状态栏)** | 动态显示当前 HTTP 服务运行状态（Running 🟢 / Stopped ⏸️）及绑定端口。 |
| **Server Settings** | 可配置 Host 绑定地址、Port 端口号、Auth Token 校验密钥及自动启动开关。 |

| **curl Quick Commands** | **Copy Short Cmd** / **Copy Script Cmd**：一键复制 ready-to-use 的 `curl` 命令模板到剪贴板。 |
| **Runtime Terminal (终端日志)** | **可滚动的终端日志视窗**，像命令行一样实时显示执行的代码（`>>>`）、标准输出及报错，配有一键复制和一键清空日志按钮。 |
| **Reset Namespace** | 点击 **Reset Console Globals** 可一键重置 Python 上下文环境，清空之前创建的全局变量。 |


---

## 📁 项目源码结构

```
blender-remote-console/
├── bin/               # CLI 辅助工具文件夹 (支持 Windows/PowerShell/CMD)
│   ├── brc.cmd        # CMD 批处理包装脚本
│   └── brc.ps1        # PowerShell 自动 Session 检测与 HTTP 传输核心逻辑
├── __init__.py        # 插件入口，注册 WindowManager/Scene 属性与组件
├── executor.py        # Python 代码执行器（REPL 表达式求值与 stdout/stderr 捕获）
├── server.py          # HTTP 服务器线程与主线程 Timer 队列调度器
├── operators.py       # Blender Operators（启动/停止、重置命名空间、复制命令等）
├── ui.py              # 3D Viewport 侧边栏 (N-Panel) 界面定义
└── README.md          # 本说明文档
```
