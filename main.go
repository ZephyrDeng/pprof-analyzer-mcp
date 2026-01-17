package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func main() {
	// 1. 初始化 MCP 服务器
	server := mcp.NewServer(&mcp.Implementation{
		Name:    "PprofAnalyzer",
		Version: "0.3.0",
	}, nil)

	// 2. 注册工具 - 使用泛型 AddTool 函数
	// analyze_pprof 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "analyze_pprof",
		Description: "分析指定的 Go pprof 文件，并返回序列化的分析结果 (例如 Top N 列表或火焰图 JSON)。",
	}, handleAnalyzePprof)

	// generate_flamegraph 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "generate_flamegraph",
		Description: "使用 'go tool pprof' 为指定的 pprof 文件生成火焰图 (SVG 格式)，将其保存到指定路径，并返回路径和 SVG 内容。",
	}, handleGenerateFlamegraph)

	// detect_memory_leaks 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "detect_memory_leaks",
		Description: "比较两个 heap profile 文件以识别潜在的内存泄漏。",
	}, handleDetectMemoryLeaks)

	// open_interactive_pprof 工具 (仅限 macOS)
	mcp.AddTool(server, &mcp.Tool{
		Name:        "open_interactive_pprof",
		Description: "【仅限 macOS】尝试在后台启动 'go tool pprof' 交互式 Web UI。成功启动后会返回进程 PID，用于后续手动断开连接。",
	}, handleOpenInteractivePprof)

	// disconnect_pprof_session 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "disconnect_pprof_session",
		Description: "尝试终止由 'open_interactive_pprof' 启动的指定后台 pprof 进程。",
	}, handleDisconnectPprofSession)

	// compare_profiles 工具
	mcp.AddTool(server, &mcp.Tool{
		Name:        "compare_profiles",
		Description: "比较两个 profile 文件（如同一服务的不同版本），生成差异分析报告，识别性能回归或改进。",
	}, handleCompareProfiles)

	// 3. 设置信号处理程序以进行清理
	setupSignalHandler()

	// 4. 使用 stdio transport 启动服务器
	log.Println("Starting PprofAnalyzer MCP server via stdio...")
	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
