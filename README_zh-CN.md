简体中文 | [English](README.md)

# Pprof Analyzer MCP 服务器

[![smithery badge](https://smithery.ai/badge/@ZephyrDeng/pprof-analyzer-mcp)](https://smithery.ai/server/@ZephyrDeng/pprof-analyzer-mcp)
![](https://badge.mcpx.dev?type=server&features=tools 'MCP server with tools')
[![Build Status](https://github.com/ZephyrDeng/pprof-analyzer-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/ZephyrDeng/pprof-analyzer-mcp/actions)
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Go Version](https://img.shields.io/github/go-mod/go-version/ZephyrDeng/pprof-analyzer-mcp)](https://golang.org)
[![GoDoc](https://pkg.go.dev/badge/github.com/ZephyrDeng/pprof-analyzer-mcp)](https://pkg.go.dev/github.com/ZephyrDeng/pprof-analyzer-mcp)

这是一个基于 Go 语言实现的模型上下文协议 (MCP) 服务器，提供了一个用于分析 Go pprof 性能剖析文件的工具。使用官方 [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk) 构建。

## 功能

*   **`analyze_pprof` 工具:**
    *   分析指定的 Go pprof 文件，并返回序列化的分析结果 (例如 Top N 列表或火焰图 JSON)。
    *   支持的 Profile 类型：
        *   `cpu`: 分析代码执行的 CPU 时间消耗，找出热点函数。
        *   `heap`: 分析程序当前的内存使用情况（堆内存分配），找出内存占用高的对象和函数。增强了对象计数、分配位置和类型信息。
        *   `goroutine`: 显示所有当前 Goroutine 的堆栈信息，用于诊断死锁、泄漏或 Goroutine 过多的问题。
        *   `allocs`: 分析程序运行期间的内存分配情况（包括已释放的），用于定位频繁分配内存的代码。提供详细的分配位置和对象计数信息。
        *   `mutex`: 分析互斥锁的竞争情况，找出导致阻塞的锁。提供详细的统计信息，包括竞争次数、延迟时间和百分比。
        *   `block`: 分析导致 Goroutine 阻塞的操作（如 channel 等待、系统调用等）。提供全面的阻塞统计，包括平均延迟计算。
    *   支持的输出格式：`text`, `markdown`, `json` (Top N 列表), `flamegraph-json` (火焰图层级数据，默认)。
        *   `text`, `markdown`: 人类可读的文本或 Markdown 格式。
        *   `json`: 以结构化 JSON 格式输出 Top N 结果 (已为 `cpu`, `heap`, `goroutine`, `allocs`, `mutex`, `block` 实现)。
        *   `flamegraph-json`: 以层级化 JSON 格式输出火焰图数据，兼容 d3-flame-graph (已为 `cpu`, `heap`, `allocs` 实现，默认格式)。输出为紧凑格式。
    *   可配置 Top N 结果数量 (`top_n`, 默认为 5，对 `text`, `markdown`, `json` 格式有效)。
*   **`generate_flamegraph` 工具:**
    *   使用 `go tool pprof` 为指定的 pprof 文件生成火焰图 (SVG 格式)，将其保存到指定路径，并返回路径和 SVG 内容。
    *   支持的 Profile 类型：`cpu`, `heap`, `allocs`, `goroutine`, `mutex`, `block`。
    *   需要用户指定输出 SVG 文件的路径。
    *   **重要：** 此功能依赖于 [Graphviz](#依赖项) 的安装。
*   **`open_interactive_pprof` 工具 (仅限 macOS):**
    *   尝试在后台为指定的 pprof 文件启动 `go tool pprof` 交互式 Web UI。如果未提供 `http_address`，默认使用端口 `:8081`。
    *   成功启动后返回后台 `pprof` 进程的进程 ID (PID)。
    *   **仅限 macOS:** 此工具仅在 macOS 上有效。
    *   **依赖项：** 需要 `go` 命令在系统的 PATH 中可用。
    *   **限制：** 服务器无法捕获后台 `pprof` 进程的错误。从远程 URL 下载的临时文件在进程终止前（通过 `disconnect_pprof_session` 手动终止或 MCP 服务器退出时）不会被自动清理。
*   **`detect_memory_leaks` 工具:**
    *   比较两个堆内存剖析快照以识别潜在的内存泄漏。
    *   按对象类型和分配位置分析内存增长情况。
    *   提供详细的内存增长统计数据，包括绝对值和百分比变化。
    *   可配置增长阈值和结果数量限制。
    *   通过比较在不同时间点获取的剖析文件来帮助识别内存泄漏。
*   **`disconnect_pprof_session` 工具:**
    *   尝试使用 PID 终止先前由 `open_interactive_pprof` 启动的后台 `pprof` 进程。
    *   首先发送 Interrupt 信号，如果失败则发送 Kill 信号。
*   **`compare_profiles` 工具:**
    *   比较两个 profile 文件（例如基线版本与目标版本）以识别性能回归或改进。
    *   支持所有 profile 类型（cpu、heap、allocs、mutex、block）。
    *   提供详细的差异统计，包括改进/回归函数、新增/移除函数。
    *   视觉指示器：🔴 回归、🟢 改进、🆕 新增、❌ 移除。
    *   支持 text、markdown 和 JSON 输出格式。
*   **`analyze_heap_time_series` 工具:**
    *   分析多个 heap profile 的时序数据以识别内存增长趋势和潜在泄漏。
    *   需要至少 3 个按时间顺序提供的 heap profile。
    *   计算增长率（字节、百分比、MB 每分钟）。
    *   识别趋势对象类型，带有方向指示器（📈 增长、📉 下降、➡️ 稳定）。
    *   支持为每个时间点提供自定义标签或自动生成默认标签。
    *   支持 text、markdown 和 JSON 输出格式。

## 安装 (作为库/工具)

你可以使用 `go install` 直接安装此包：

```bash
go install github.com/ZephyrDeng/pprof-analyzer-mcp@latest
```
这会将 `pprof-analyzer-mcp` 可执行文件安装到你的 `$GOPATH/bin` 或 `$HOME/go/bin` 目录下。请确保该目录已添加到你的系统 PATH 环境变量中，以便直接运行命令。

## 从源码构建

确保你已经安装了 Go 环境 (推荐 Go 1.18 或更高版本)。

在项目根目录 (`pprof-analyzer-mcp`) 下运行：

```bash
go build
```

这将生成一个名为 `pprof-analyzer-mcp` (或 `pprof-analyzer-mcp.exe` 在 Windows 上) 的可执行文件在当前目录下。

### 使用 `go install` (推荐)

你也可以使用 `go install` 将可执行文件安装到你的 `$GOPATH/bin` 或 `$HOME/go/bin` 目录下，这样可以直接在命令行中运行 `pprof-analyzer-mcp` (如果该目录已添加到你的系统 PATH 环境变量中)。

```bash
# 使用 go.mod 中定义的模块路径安装可执行文件
go install .
# 或者直接使用 GitHub 路径 (发布后推荐)
# go install github.com/ZephyrDeng/pprof-analyzer-mcp@latest
```

## 使用 Docker 运行

使用 Docker 是一种便捷的运行服务器的方式，因为它打包了必需的 Graphviz 依赖。

1.  **构建 Docker 镜像：**
    在项目根目录（包含 `Dockerfile` 文件的目录）下运行：
    ```bash
    docker build -t pprof-analyzer-mcp .
    ```

2.  **运行 Docker 容器：**
    ```bash
    docker run -i --rm pprof-analyzer-mcp
    ```
    *   `-i` 标志保持标准输入 (STDIN) 打开，这是此 MCP 服务器使用的 stdio 传输协议所必需的。
    *   `--rm` 标志表示容器退出时自动删除。

3.  **为 Docker 配置 MCP 客户端：**
    要将你的 MCP 客户端（如 Roo Cline）连接到在 Docker 内部运行的服务器，请更新你的 `.roo/mcp.json`：
    ```json
    {
      "mcpServers": {
        "pprof-analyzer-docker": {
          "command": "docker run -i --rm pprof-analyzer-mcp"
        }
      }
    }
    ```
    在客户端尝试运行此命令之前，请确保 `pprof-analyzer-mcp` 镜像已在本地构建。


## 发布流程 (通过 GitHub Actions 自动化)

本项目使用 [GoReleaser](https://goreleaser.com/) 和 GitHub Actions 来自动化发布流程。当一个匹配 `v*` 模式（例如 `v0.1.0`, `v1.2.3`）的 Git 标签被推送到仓库时，会自动触发发布。

**发布前检查清单：**

在创建发布标签前，请确保：
- ✅ 所有测试通过：`go test ./...`
- ✅ 代码编译成功：`go build`
- ✅ 文档是最新的（README、CHANGELOG 等）
- ✅ 提交消息遵循 [Conventional Commits](https://www.conventionalcommits.org/) 格式

**发布步骤：**

1.  **进行更改：** 开发新功能或修复 Bug。
2.  **提交更改：** 使用 [Conventional Commits](https://www.conventionalcommits.org/) 格式提交你的更改 (例如 `feat: ...`, `fix: ...`, `docs: ...`)。这对自动生成 Changelog 很重要。
    ```bash
    git add .
    git commit -m "feat: 添加了很棒的新功能"
    # 或者
    git commit -m "fix: 解决了问题 #42"
    # 或者
    git commit -m "docs: 更新 README 说明新功能"
    ```
3.  **推送更改：** 将你的提交推送到 GitHub 的主分支。
    ```bash
    git push origin main
    ```
4.  **运行发布前测试：** 可选，在打标签前本地运行测试：
    ```bash
    go test ./... -v
    go build -v
    ```
5.  **创建并推送标签：** 准备好发布时，创建一个新的 Git 标签并将其推送到 GitHub。
    ```bash
    # 示例：创建标签 v0.2.0
    git tag v0.2.0

    # 推送标签到 GitHub
    git push origin v0.2.0
    ```
6.  **自动发布：** 推送标签将触发 `.github/workflows/release.yml` 中定义的 `GoReleaser` GitHub Action。此 Action 将会：
    *   为 Linux、macOS 和 Windows (amd64 & arm64) 构建二进制文件。
    *   基于自上一个标签以来的 Conventional Commits 生成 Changelog。
    *   在 GitHub 上创建一个新的 Release，包含 Changelog，并将构建好的二进制文件和校验和文件作为附件上传。

**监控发布进度：**

你可以在 GitHub 仓库的 "Actions" 标签页查看发布工作流的进度。完成后，发布将在以下地址可用：
```
https://github.com/ZephyrDeng/pprof-analyzer-mcp/releases
```

## 配置 MCP 客户端

本服务器使用 `stdio` 传输协议。你需要在你的 MCP 客户端 (例如 VS Code 的 Roo Cline 扩展) 中配置它。

通常，这需要在项目根目录的 `.roo/mcp.json` 文件中添加如下配置：

```json
{
  "mcpServers": {
    "pprof-analyzer": {
      "command": "pprof-analyzer-mcp"
    }
  }
}
```

**注意：** `command` 的值需要根据你的构建方式（`go build` 或 `go install`）和可执行文件的实际位置进行调整。确保 MCP 客户端能够找到并执行这个命令。

配置完成后，重新加载或重启你的 MCP 客户端，它应该会自动连接到 `PprofAnalyzer` 服务器。

## 依赖项

*   **Graphviz**: `generate_flamegraph` 工具需要 Graphviz 来生成 SVG 火焰图 (`go tool pprof` 在生成 SVG 时会调用 `dot` 命令)。请确保你的系统已经安装了 Graphviz 并且 `dot` 命令在系统的 PATH 环境变量中。

    **安装 Graphviz:**
    *   **macOS (使用 Homebrew):**
        ```bash
        brew install graphviz
        ```
    *   **Debian/Ubuntu:**
        ```bash
        sudo apt-get update && sudo apt-get install graphviz
        ```
    *   **CentOS/Fedora:**
        ```bash
        sudo yum install graphviz
        # 或者
        sudo dnf install graphviz
        ```
    *   **Windows (使用 Chocolatey):**
        ```bash
        choco install graphviz
        ```
    *   **其他系统：** 请参考 [Graphviz 官方下载页面](https://graphviz.org/download/)。

## 使用示例 (通过 MCP 客户端)

一旦服务器连接成功，你就可以使用 `file://`, `http://`, 或 `https://` URI 来调用 `analyze_pprof` 和 `generate_flamegraph` 工具了。

**示例：分析 CPU Profile (文本格式，Top 5)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu"
  }
}
```

**示例：分析 Heap Profile (Markdown 格式，Top 10)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/heap.pprof",
    "profile_type": "heap",
    "top_n": 10,
    "output_format": "markdown"
  }
}
```

**示例：分析 Goroutine Profile (文本格式，Top 5)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/goroutine.pprof",
    "profile_type": "goroutine"
  }
}
```

**示例：生成 CPU Profile 的火焰图**

```json
{
  "tool_name": "generate_flamegraph",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu",
    "output_svg_path": "/path/to/save/cpu_flamegraph.svg"
  }
}
```

**示例：生成 Heap Profile (inuse_space) 的火焰图**

```json
{
  "tool_name": "generate_flamegraph",
  "arguments": {
    "profile_uri": "file:///path/to/your/heap.pprof",
    "profile_type": "heap",
    "output_svg_path": "/path/to/save/heap_flamegraph.svg"
  }
}
```

**示例：分析 CPU Profile (JSON 格式，Top 3)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu",
    "top_n": 3,
    "output_format": "json"
  }
}
```

**示例：分析 CPU Profile (默认火焰图 JSON 格式)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu"
    // output_format 默认为 "flamegraph-json"
  }
}
```

**示例：分析 Heap Profile (显式指定火焰图 JSON 格式)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/heap.pprof",
    "profile_type": "heap",
    "output_format": "flamegraph-json"
  }
}
```

**示例：分析远程 CPU Profile (来自 HTTP URL)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "https://example.com/profiles/cpu.pprof",
    "profile_type": "cpu"
  }
}
```

**示例：分析在线 CPU Profile (来自 GitHub Raw URL)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "https://raw.githubusercontent.com/google/pprof/refs/heads/main/profile/testdata/gobench.cpu",
    "profile_type": "cpu",
    "top_n": 5
  }
}
```

**示例：生成在线 Heap Profile 的火焰图 (来自 GitHub Raw URL)**

```json
{
  "tool_name": "generate_flamegraph",
  "arguments": {
    "profile_uri": "https://raw.githubusercontent.com/google/pprof/refs/heads/main/profile/testdata/gobench.heap",
    "profile_type": "heap",
    "output_svg_path": "./online_heap_flamegraph.svg"
  }
}
```

**示例：为在线 CPU Profile 打开交互式 Pprof UI (仅限 macOS)**

```json
{
  "tool_name": "open_interactive_pprof",
  "arguments": {
    "profile_uri": "https://raw.githubusercontent.com/google/pprof/refs/heads/main/profile/testdata/gobench.cpu"
    // 可选："http_address": ":8082" // 覆盖默认端口的示例
  }
}
```

**示例：检测两个堆内存剖析文件之间的内存泄漏**

```json
{
  "tool_name": "detect_memory_leaks",
  "arguments": {
    "old_profile_uri": "file:///path/to/your/heap_before.pprof",
    "new_profile_uri": "file:///path/to/your/heap_after.pprof",
    "threshold": 0.05,  // 5% 增长阈值
    "limit": 15         // 显示前 15 个潜在泄漏点
  }
}
```

**示例：断开 Pprof 会话连接**

```json
{
  "tool_name": "disconnect_pprof_session",
  "arguments": {
    "pid": 12345 // 将 12345 替换为 open_interactive_pprof 返回的实际 PID
  }
}
```

## 未来改进 (TODO)

*   为内存剖析添加时序分析功能，以跟踪多个快照的增长情况。
*   实现差异火焰图以可视化剖析文件之间的变化。
*   在 MCP 结果中根据 `output_format` 设置合适的 MIME 类型。
*   增加更健壮的错误处理和日志级别控制。

## 最近完成 (v0.3.0)

*   ✅ ~~实现差异火焰图以可视化剖析文件之间的变化。~~ (已完成 - `compare_profiles` 工具)
*   ✅ ~~为内存剖析添加时序分析功能，以跟踪多个快照的增长情况。~~ (已完成 - `analyze_heap_time_series` 工具)
*   ✅ 添加自动化 CI/CD，在每次 PR 时运行 GitHub Actions 测试。
*   ✅ ~~实现 `mutex`, `block` profile 的完整分析逻辑。~~ (v0.2.0 已完成)
*   ✅ ~~为 `mutex`, `block` profile 类型实现 `json` 输出格式。~~ (v0.2.0 已完成)
*   ✅ 迁移到官方 [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk)。
*   ✅ ~~考虑支持远程 pprof 文件 URI (例如 `http://`, `https://`)。~~ (v0.2.0 已完成)
*   ✅ ~~实现 `allocs` profile 的完整分析逻辑。~~ (v0.2.0 已完成)
*   ✅ ~~为 `allocs` profile 类型实现 `json` 输出格式。~~ (v0.2.0 已完成)
*   ✅ ~~添加内存泄漏检测功能。~~ (v0.2.0 已完成)