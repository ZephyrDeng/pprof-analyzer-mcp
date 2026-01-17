[简体中文](README_zh-CN.md) | English

# Pprof Analyzer MCP Server

[![smithery badge](https://smithery.ai/badge/@ZephyrDeng/pprof-analyzer-mcp)](https://smithery.ai/server/@ZephyrDeng/pprof-analyzer-mcp)
![](https://badge.mcpx.dev?type=server&features=tools 'MCP server with tools')
[![Build Status](https://github.com/ZephyrDeng/pprof-analyzer-mcp/actions/workflows/release.yml/badge.svg)](https://github.com/ZephyrDeng/pprof-analyzer-mcp/actions)
[![License](https://img.shields.io/badge/license-MIT-blue)]()
[![Go Version](https://img.shields.io/github/go-mod/go-version/ZephyrDeng/pprof-analyzer-mcp)](https://golang.org)
[![GoDoc](https://pkg.go.dev/badge/github.com/ZephyrDeng/pprof-analyzer-mcp)](https://pkg.go.dev/github.com/ZephyrDeng/pprof-analyzer-mcp)

This is a Model Context Protocol (MCP) server implemented in Go, providing a tool to analyze Go pprof performance profiles. Built with the official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk).

## Features

*   **`analyze_pprof` Tool:**
    *   Analyzes the specified Go pprof file and returns serialized analysis results (e.g., Top N list or flame graph JSON).
    *   Supported Profile Types:
        *   `cpu`: Analyzes CPU time consumption during code execution to find hot spots.
        *   `heap`: Analyzes the current memory usage (heap allocations) to find objects and functions with high memory consumption. Enhanced with object count, allocation site, and type information.
        *   `goroutine`: Displays stack traces of all current goroutines, used for diagnosing deadlocks, leaks, or excessive goroutine usage.
        *   `allocs`: Analyzes memory allocations (including freed ones) during program execution to locate code with frequent allocations. Provides detailed allocation site and object count information.
        *   `mutex`: Analyzes contention on mutexes to find locks causing blocking. Provides detailed statistics including contention counts, delay times, and percentages.
        *   `block`: Analyzes operations causing goroutine blocking (e.g., channel waits, system calls). Provides comprehensive blocking statistics with average delay calculations.
    *   Supported Output Formats: `text`, `markdown`, `json` (Top N list), `flamegraph-json` (hierarchical flame graph data, default).
        *   `text`, `markdown`: Human-readable text or Markdown format.
        *   `json`: Outputs Top N results in structured JSON format (implemented for `cpu`, `heap`, `goroutine`, `allocs`, `mutex`, `block`).
        *   `flamegraph-json`: Outputs hierarchical flame graph data in JSON format, compatible with d3-flame-graph (implemented for `cpu`, `heap`, `allocs`, default format). Output is compact.
    *   Configurable number of Top N results (`top_n`, defaults to 5, effective for `text`, `markdown`, `json` formats).
*   **`generate_flamegraph` Tool:**
    *   Uses `go tool pprof` to generate a flame graph (SVG format) for the specified pprof file, saves it to the specified path, and returns the path and SVG content.
    *   Supported Profile Types: `cpu`, `heap`, `allocs`, `goroutine`, `mutex`, `block`.
    *   Requires the user to specify the output SVG file path.
    *   **Important:** This feature depends on [Graphviz](#dependencies) being installed.
*   **`open_interactive_pprof` Tool (macOS Only):**
    *   Attempts to launch the `go tool pprof` interactive web UI in the background for the specified pprof file. Uses port `:8081` by default if `http_address` is not provided.
    *   Returns the Process ID (PID) of the background `pprof` process upon successful launch.
    *   **macOS Only:** This tool will only work on macOS.
    *   **Dependencies:** Requires the `go` command to be available in the system's PATH.
    *   **Limitations:** Errors from the background `pprof` process are not captured by the server. Temporary files downloaded from remote URLs are not automatically cleaned up until the process is terminated (either manually via `disconnect_pprof_session` or when the MCP server exits).
*   **`detect_memory_leaks` Tool:**
    *   Compares two heap profile snapshots to identify potential memory leaks.
    *   Analyzes memory growth by object type and allocation site.
    *   Provides detailed statistics on memory growth, including absolute and percentage changes.
    *   Configurable growth threshold and result limit.
    *   Helps identify memory leaks by comparing profiles taken at different points in time.
*   **`disconnect_pprof_session` Tool:**
    *   Attempts to terminate a background `pprof` process previously started by `open_interactive_pprof`, using its PID.
    *   Sends an Interrupt signal first, then a Kill signal if Interrupt fails.
*   **`compare_profiles` Tool:**
    *   Compares two profile files (e.g., baseline vs. target) to identify performance regressions or improvements.
    *   Supports all profile types (cpu, heap, allocs, mutex, block).
    *   Provides detailed diff statistics including improved/regressed functions, added/removed functions.
    *   Visual indicators: 🔴 regression, 🟢 improvement, 🆕 added, ❌ removed.
    *   Supports text, markdown, and JSON output formats.
*   **`analyze_heap_time_series` Tool:**
    *   Analyzes multiple heap profiles over time to identify memory growth trends and potential leaks.
    *   Requires at least 3 heap profiles provided in chronological order.
    *   Calculates growth rates (bytes, percentage, MB per minute).
    *   Identifies trending object types with directional indicators (📈 increasing, 📉 decreasing, ➡️ stable).
    *   Supports custom labels for each time point or auto-generates default labels.
    *   Supports text, markdown, and JSON output formats.

## Installation (As a Library/Tool)

You can install this package directly using `go install`:

```bash
go install github.com/ZephyrDeng/pprof-analyzer-mcp@latest
```
This will install the `pprof-analyzer-mcp` executable to your `$GOPATH/bin` or `$HOME/go/bin` directory. Ensure this directory is in your system's PATH to run the command directly.

## Building from Source

Ensure you have a Go environment installed (Go 1.18 or higher recommended).

In the project root directory (`pprof-analyzer-mcp`), run:

```bash
go build
```

This will generate an executable file named `pprof-analyzer-mcp` (or `pprof-analyzer-mcp.exe` on Windows) in the current directory.

### Using `go install` (Recommended)

You can also use `go install` to install the executable into your `$GOPATH/bin` or `$HOME/go/bin` directory. This allows you to run `pprof-analyzer-mcp` directly from the command line (if the directory is added to your system's PATH environment variable).

```bash
# Installs the executable using the module path defined in go.mod
go install .
# Or directly using the GitHub path (recommended after publishing)
# go install github.com/ZephyrDeng/pprof-analyzer-mcp@latest
```

## Running with Docker

Using Docker is a convenient way to run the server, as it bundles the necessary Graphviz dependency.

1.  **Build the Docker Image:**
    In the project root directory (where the `Dockerfile` is located), run:
    ```bash
    docker build -t pprof-analyzer-mcp .
    ```

2.  **Run the Docker Container:**
    ```bash
    docker run -i --rm pprof-analyzer-mcp
    ```
    *   The `-i` flag keeps STDIN open, which is required for the stdio transport used by this MCP server.
    *   The `--rm` flag automatically removes the container when it exits.

3.  **Configure MCP Client for Docker:**
    To connect your MCP client (like Roo Cline) to the server running inside Docker, update your `.roo/mcp.json`:
    ```json
    {
      "mcpServers": {
        "pprof-analyzer-docker": {
          "command": "docker run -i --rm pprof-analyzer-mcp"
        }
      }
    }
    ```
    Make sure the `pprof-analyzer-mcp` image has been built locally before the client tries to run this command.


## Releasing (Automated via GitHub Actions)

This project uses [GoReleaser](https://goreleaser.com/) and GitHub Actions to automate the release process. Releases are triggered automatically when a Git tag matching the pattern `v*` (e.g., `v0.1.0`, `v1.2.3`) is pushed to the repository.

**Pre-release Checklist:**

Before creating a release tag, ensure:
- ✅ All tests pass: `go test ./...`
- ✅ Code compiles successfully: `go build`
- ✅ Documentation is up to date (README, CHANGELOG, etc.)
- ✅ Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/) format

**Release Steps:**

1.  **Make Changes:** Develop new features or fix bugs.
2.  **Commit Changes:** Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat: ...`, `fix: ...`, `docs: ...`). This is important for automatic changelog generation.
    ```bash
    git add .
    git commit -m "feat: Add awesome new feature"
    # or
    git commit -m "fix: Resolve issue #42"
    # or
    git commit -m "docs: Update README for new feature"
    ```
3.  **Push Changes:** Push your commits to the main branch on GitHub.
    ```bash
    git push origin main
    ```
4.  **Run Pre-release Tests:** Optionally, run tests locally before tagging:
    ```bash
    go test ./... -v
    go build -v
    ```
5.  **Create and Push Tag:** When ready to release, create a new Git tag and push it to GitHub.
    ```bash
    # Example: Create tag v0.2.0
    git tag v0.2.0

    # Push the tag to GitHub
    git push origin v0.2.0
    ```
6.  **Automatic Release:** Pushing the tag will trigger the `GoReleaser` GitHub Action defined in `.github/workflows/release.yml`. This action will:
    *   Build binaries for Linux, macOS, and Windows (amd64 & arm64).
    *   Generate a changelog based on Conventional Commits since the last tag.
    *   Create a new GitHub Release with the changelog and attach the built binaries and checksums as assets.

**Monitoring the Release:**

You can view the release workflow progress in the "Actions" tab of the GitHub repository. Once complete, the release will be available at:
```
https://github.com/ZephyrDeng/pprof-analyzer-mcp/releases
```

## Configuring the MCP Client

This server uses the `stdio` transport protocol. You need to configure it in your MCP client (e.g., Roo Cline extension for VS Code).

Typically, this involves adding the following configuration to the `.roo/mcp.json` file in your project root:

```json
{
  "mcpServers": {
    "pprof-analyzer": {
      "command": "pprof-analyzer-mcp"
    }
  }
}
```

**Note:** Adjust the `command` value based on your build method (`go build` or `go install`) and the actual location of the executable. Ensure the MCP client can find and execute this command.

After configuration, reload or restart your MCP client, and it should automatically connect to the `PprofAnalyzer` server.

## Dependencies

*   **Graphviz**: The `generate_flamegraph` tool requires Graphviz to generate SVG flame graphs (the `go tool pprof` command calls `dot` when generating SVG). Ensure Graphviz is installed on your system and the `dot` command is available in your system's PATH environment variable.

    **Installing Graphviz:**
    *   **macOS (using Homebrew):**
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
        # or
        sudo dnf install graphviz
        ```
    *   **Windows (using Chocolatey):**
        ```bash
        choco install graphviz
        ```
    *   **Other Systems:** Refer to the [Graphviz official download page](https://graphviz.org/download/).

## Usage Examples (via MCP Client)

Once the server is connected, you can call the `analyze_pprof` and `generate_flamegraph` tools using `file://`, `http://`, or `https://` URIs for the profile file.

**Example: Analyze CPU Profile (Text format, Top 5)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu"
  }
}
```

**Example: Analyze Heap Profile (Markdown format, Top 10)**

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

**Example: Analyze Goroutine Profile (Text format, Top 5)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/goroutine.pprof",
    "profile_type": "goroutine"
  }
}
```

**Example: Generate Flame Graph for CPU Profile**

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

**Example: Generate Flame Graph for Heap Profile (inuse_space)**

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

**Example: Analyze CPU Profile (JSON format, Top 3)**

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

**Example: Analyze CPU Profile (Default Flame Graph JSON format)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "file:///path/to/your/cpu.pprof",
    "profile_type": "cpu"
    // output_format defaults to "flamegraph-json"
  }
}
```

**Example: Analyze Heap Profile (Explicitly Flame Graph JSON format)**

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

**Example: Analyze Remote CPU Profile (from HTTP URL)**

```json
{
  "tool_name": "analyze_pprof",
  "arguments": {
    "profile_uri": "https://example.com/profiles/cpu.pprof",
    "profile_type": "cpu"
  }
}
```

**Example: Analyze Online CPU Profile (from GitHub Raw URL)**

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

**Example: Generate Flame Graph for Online Heap Profile (from GitHub Raw URL)**

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

**Example: Open Interactive Pprof UI for Online CPU Profile (macOS Only)**

```json
{
  "tool_name": "open_interactive_pprof",
  "arguments": {
    "profile_uri": "https://raw.githubusercontent.com/google/pprof/refs/heads/main/profile/testdata/gobench.cpu"
    // Optional: "http_address": ":8082" // Example of overriding the default port
  }
}
```

**Example: Detect Memory Leaks Between Two Heap Profiles**

```json
{
  "tool_name": "detect_memory_leaks",
  "arguments": {
    "old_profile_uri": "file:///path/to/your/heap_before.pprof",
    "new_profile_uri": "file:///path/to/your/heap_after.pprof",
    "threshold": 0.05,  // 5% growth threshold
    "limit": 15         // Show top 15 potential leaks
  }
}
```

**Example: Disconnect a Pprof Session**

```json
{
  "tool_name": "disconnect_pprof_session",
  "arguments": {
    "pid": 12345 // Replace 12345 with the actual PID returned by open_interactive_pprof
  }
}
```

## Future Improvements (TODO)

*   Add MIME type handling in MCP results based on `output_format`.
*   Add more robust error handling and logging level control.
*   Add integration tests for end-to-end MCP tool interactions.
*   Performance optimizations for large profile files (>1GB).

## Recently Completed (v0.3.0)

*   ✅ ~~Implement differential flame graphs to visualize changes between profiles.~~ (Done - `compare_profiles` tool)
*   ✅ ~~Add time-series analysis for memory profiles to track growth over multiple snapshots.~~ (Done - `analyze_heap_time_series` tool)
*   ✅ Add automated CI/CD with GitHub Actions testing on every PR.
*   ✅ ~~Implement full analysis logic for `mutex`, `block` profiles.~~ (Done in v0.2.0)
*   ✅ ~~Implement `json` output format for `mutex`, `block` profile types.~~ (Done in v0.2.0)
*   ✅ Migrated to official [Model Context Protocol Go SDK](https://github.com/modelcontextprotocol/go-sdk).
*   ✅ ~~Consider supporting remote pprof file URIs (e.g., `http://`, `https://`).~~ (Done in v0.2.0)
*   ✅ ~~Implement full analysis logic for `allocs` profiles.~~ (Done in v0.2.0)
*   ✅ ~~Implement `json` output format for `allocs` profile type.~~ (Done in v0.2.0)
*   ✅ ~~Add memory leak detection capabilities.~~ (Done in v0.2.0)