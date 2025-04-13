[简体中文](README_zh-CN.md) | English

# Pprof Analyzer MCP Server

[![smithery badge](https://smithery.ai/badge/@ZephyrDeng/pprof-analyzer-mcp)](https://smithery.ai/server/@ZephyrDeng/pprof-analyzer-mcp)

This is a Model Context Protocol (MCP) server implemented in Go, providing a tool to analyze Go pprof performance profiles.

## Features

*   **`analyze_pprof` Tool:**
    *   Analyzes pprof files from local paths (`file://`) or remote URLs (`http://`, `https://`).
    *   Supported Profile Types:
        *   `cpu`: Analyzes CPU time consumption during code execution to find hot spots.
        *   `heap`: Analyzes the current memory usage (heap allocations) to find objects and functions with high memory consumption.
        *   `goroutine`: Displays stack traces of all current goroutines, used for diagnosing deadlocks, leaks, or excessive goroutine usage.
        *   `allocs`: Analyzes memory allocations (including freed ones) during program execution to locate code with frequent allocations. (*Not yet implemented*)
        *   `mutex`: Analyzes contention on mutexes to find locks causing blocking. (*Not yet implemented*)
        *   `block`: Analyzes operations causing goroutine blocking (e.g., channel waits, system calls). (*Not yet implemented*)
    *   Supported Output Formats: `text`, `markdown`, `json`.
        *   `text` and `markdown` are implemented.
        *   `json`: Outputs Top N results in structured JSON format (implemented for `cpu`, `heap`, `goroutine`).
    *   Configurable number of results to return (`top_n`, defaults to 5).
*   **`generate_flamegraph` Tool:**
    *   Uses the `go tool pprof` command to generate an SVG flame graph for the specified pprof file.
    *   Supported Profile Types: `cpu`, `heap`, `allocs`, `goroutine`, `mutex`, `block`.
    *   Requires the user to specify the output SVG file path.
    *   **Important:** This feature depends on [Graphviz](#dependencies) being installed.
*   **`open_interactive_pprof` Tool (macOS Only):**
    *   Attempts to launch the `go tool pprof` interactive web UI in the background for the specified pprof file. Uses port `:8081` by default if `http_address` is not provided.
    *   Returns the Process ID (PID) of the background `pprof` process upon successful launch.
    *   **macOS Only:** This tool will only work on macOS.
    *   **Dependencies:** Requires the `go` command to be available in the system's PATH.
    *   **Limitations:** Errors from the background `pprof` process are not captured by the server. Temporary files downloaded from remote URLs are not automatically cleaned up until the process is terminated (either manually via `disconnect_pprof_session` or when the MCP server exits).
*   **`disconnect_pprof_session` Tool:**
    *   Attempts to terminate a background `pprof` process previously started by `open_interactive_pprof`, using its PID.
    *   Sends an Interrupt signal first, then a Kill signal if Interrupt fails.

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
        "pprof-analyzer-docker": { // Use a distinct name for the Docker setup
          // Use 'docker run' as the command
          "command": "docker run -i --rm pprof-analyzer-mcp"
        }
        // ... other server configurations
      }
    }
    ```
    Make sure the `pprof-analyzer-mcp` image has been built locally before the client tries to run this command.


## Releasing (Automated via GitHub Actions)

This project uses [GoReleaser](https://goreleaser.com/) and GitHub Actions to automate the release process. Releases are triggered automatically when a Git tag matching the pattern `v*` (e.g., `v0.1.0`, `v1.2.3`) is pushed to the repository.

**Release Steps:**

1.  **Make Changes:** Develop new features or fix bugs.
2.  **Commit Changes:** Commit your changes using [Conventional Commits](https://www.conventionalcommits.org/) format (e.g., `feat: ...`, `fix: ...`). This is important for automatic changelog generation.
    ```bash
    git add .
    git commit -m "feat: Add awesome new feature"
    # or
    git commit -m "fix: Resolve issue #42"
    ```
3.  **Push Changes:** Push your commits to the main branch on GitHub.
    ```bash
    git push origin main
    ```
4.  **Create and Push Tag:** When ready to release, create a new Git tag and push it to GitHub.
    ```bash
    # Example: Create tag v0.1.0
    git tag v0.1.0

    # Push the tag to GitHub
    git push origin v0.1.0
    ```
5.  **Automatic Release:** Pushing the tag will trigger the `GoReleaser` GitHub Action defined in `.github/workflows/release.yml`. This action will:
    *   Build binaries for Linux, macOS, and Windows (amd64 & arm64).
    *   Generate a changelog based on Conventional Commits since the last tag.
    *   Create a new GitHub Release with the changelog and attach the built binaries and checksums as assets.

You can view the release workflow progress in the "Actions" tab of the GitHub repository.

## Configuring the MCP Client

This server uses the `stdio` transport protocol. You need to configure it in your MCP client (e.g., Roo Cline extension for VS Code).

Typically, this involves adding the following configuration to the `.roo/mcp.json` file in your project root:

```json
{
  "mcpServers": {
    "pprof-analyzer": { // You can choose a name for the server
      // Specify the command to start the server.
      // If you used 'go build', provide the **absolute path** or **relative path** (from project root) to the compiled executable.
      // Example: "command": "/path/to/your/pprof-analyzer-mcp/pprof-analyzer-mcp"
      // Or: "command": "./pprof-analyzer-mcp/pprof-analyzer-mcp"
      //
      // If you used 'go install github.com/ZephyrDeng/pprof-analyzer-mcp@latest'
      // and $GOPATH/bin or $HOME/go/bin is in your PATH,
      // you can use the executable name directly:
      "command": "pprof-analyzer-mcp"
      // Alternatively, if built and installed from source (go install .):
      // "command": "pprof-analyzer-mcp" (if in PATH)
      // Or specify the built path:
      // "command": "./pprof-analyzer-mcp/pprof-analyzer-mcp"
    }
    // ... other server configurations
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

*   Implement full analysis logic for `allocs`, `mutex`, `block` profiles.
*   Implement `json` output format for `allocs`, `mutex`, `block` profile types.
*   Set appropriate MIME types in MCP results based on `output_format`.
*   Add more robust error handling and logging level control.
*   ~~Consider supporting remote pprof file URIs (e.g., `http://`, `https://`).~~ (Done)