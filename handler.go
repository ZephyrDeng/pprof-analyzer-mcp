package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/pprof/profile"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/ZephyrDeng/pprof-analyzer-mcp/analyzer"
)

// AnalyzePprofArgs 定义 analyze_pprof 工具的输入参数
type AnalyzePprofArgs struct {
	ProfileURI   string  `json:"profile_uri" jsonschema:"description=要分析的 pprof 文件的 URI (支持 'file://', 'http://', 'https://' 协议)"`
	ProfileType  string  `json:"profile_type" jsonschema:"description=要分析的 pprof profile 的类型,enum=cpu,enum=heap,enum=goroutine,enum=allocs,enum=mutex,enum=block"`
	TopN         float64 `json:"top_n,omitempty" jsonschema:"description=返回结果的数量上限 (例如 Top 5, Top 10),default=5"`
	OutputFormat string  `json:"output_format,omitempty" jsonschema:"description=分析结果的输出格式,enum=text,enum=markdown,enum=json,enum=flamegraph-json,default=flamegraph-json"`
}

// handleAnalyzePprof 处理分析 pprof 文件的请求。
func handleAnalyzePprof(_ context.Context, _ *mcp.CallToolRequest, args AnalyzePprofArgs) (*mcp.CallToolResult, any, error) {
	if args.ProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_uri")
	}
	if args.ProfileType == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_type")
	}

	// 设置默认值
	if args.TopN <= 0 {
		args.TopN = 5
	}
	if args.OutputFormat == "" {
		args.OutputFormat = "flamegraph-json"
	}

	topN := int(args.TopN)
	log.Printf("Handling analyze_pprof: URI=%s, Type=%s, TopN=%d, Format=%s", args.ProfileURI, args.ProfileType, topN, args.OutputFormat)

	filePath, cleanup, err := getProfileAsFile(args.ProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get profile file: %w", err)
	}
	defer cleanup()

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening profile file '%s': %v", filePath, err)
		return nil, nil, fmt.Errorf("failed to open profile file '%s': %w", filePath, err)
	}
	defer file.Close()

	prof, err := profile.Parse(file)
	if err != nil {
		log.Printf("Error parsing profile file '%s': %v", filePath, err)
		return nil, nil, fmt.Errorf("failed to parse profile file '%s': %w", filePath, err)
	}
	log.Printf("Successfully parsed profile file from path: %s", filePath)

	var analysisResult string
	var analysisErr error

	switch args.ProfileType {
	case "cpu":
		analysisResult, analysisErr = analyzer.AnalyzeCPUProfile(prof, topN, args.OutputFormat)
	case "heap":
		analysisResult, analysisErr = analyzer.AnalyzeHeapProfile(prof, topN, args.OutputFormat)
	case "goroutine":
		analysisResult, analysisErr = analyzer.AnalyzeGoroutineProfile(prof, topN, args.OutputFormat)
	case "allocs":
		analysisResult, analysisErr = analyzer.AnalyzeAllocsProfile(prof, topN, args.OutputFormat)
	case "mutex":
		analysisResult, analysisErr = analyzer.AnalyzeMutexProfile(prof, topN, args.OutputFormat)
	case "block":
		analysisResult, analysisErr = analyzer.AnalyzeBlockProfile(prof, topN, args.OutputFormat)
	default:
		analysisErr = fmt.Errorf("unsupported profile type: '%s'", args.ProfileType)
	}

	if analysisErr != nil {
		log.Printf("Analysis error for type '%s': %v", args.ProfileType, analysisErr)
		return nil, nil, analysisErr
	}

	log.Printf("Analysis successful for type '%s'. Result length: %d", args.ProfileType, len(analysisResult))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				
				Text: analysisResult,
			},
		},
	}, nil, nil
}

// GenerateFlamegraphArgs 定义 generate_flamegraph 工具的输入参数
type GenerateFlamegraphArgs struct {
	ProfileURI    string `json:"profile_uri" jsonschema:"description=要生成火焰图的 pprof 文件的 URI (支持 'file://', 'http://', 'https://' 协议)"`
	ProfileType   string `json:"profile_type" jsonschema:"description=要生成火焰图的 pprof profile 的类型,enum=cpu,enum=heap,enum=allocs,enum=goroutine,enum=mutex,enum=block"`
	OutputSVGPath string `json:"output_svg_path" jsonschema:"description=生成的 SVG 火焰图文件的保存路径 (必须是绝对路径或相对于工作区的路径)"`
}

// handleGenerateFlamegraph 处理生成火焰图的请求。
func handleGenerateFlamegraph(_ context.Context, _ *mcp.CallToolRequest, args GenerateFlamegraphArgs) (*mcp.CallToolResult, any, error) {
	if args.ProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_uri")
	}
	if args.ProfileType == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_type")
	}
	if args.OutputSVGPath == "" {
		return nil, nil, fmt.Errorf("missing required argument: output_svg_path")
	}

	log.Printf("Handling generate_flamegraph: URI=%s, Type=%s, Output=%s", args.ProfileURI, args.ProfileType, args.OutputSVGPath)

	inputFilePath, cleanup, err := getProfileAsFile(args.ProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get profile file for flamegraph: %w", err)
	}
	defer cleanup()

	if !filepath.IsAbs(args.OutputSVGPath) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Printf("无法获取当前工作目录: %v", err)
		} else {
			args.OutputSVGPath = filepath.Join(cwd, args.OutputSVGPath)
			log.Printf("将相对输出路径转换为绝对路径: %s", args.OutputSVGPath)
		}
	}

	cmdArgs := []string{"tool", "pprof"}
	switch args.ProfileType {
	case "heap":
		cmdArgs = append(cmdArgs, "-inuse_space")
	case "allocs":
		cmdArgs = append(cmdArgs, "-alloc_space")
	case "cpu", "goroutine", "mutex", "block":
		// No extra flags needed
	default:
		return nil, nil, fmt.Errorf("unsupported profile type for flamegraph: '%s'", args.ProfileType)
	}
	cmdArgs = append(cmdArgs, "-svg", "-output", args.OutputSVGPath, inputFilePath)

	log.Printf("Executing command: go %s", strings.Join(cmdArgs, " "))

	_, err = exec.LookPath("dot")
	if err != nil {
		errMsg := "Graphviz (dot 命令) 未找到或不在 PATH 中。生成 SVG 火焰图需要 Graphviz。\n" +
			"请先安装 Graphviz。常见安装方式：\n" +
			"- macOS (Homebrew): brew install graphviz\n" +
			"- Debian/Ubuntu: sudo apt-get update && sudo apt-get install graphviz\n" +
			"- CentOS/Fedora: sudo yum install graphviz 或 sudo dnf install graphviz\n" +
			"- Windows (Chocolatey): choco install graphviz"
		log.Println(errMsg)
		return nil, nil, fmt.Errorf(errMsg)
	}
	log.Println("Graphviz (dot) found.")

	cmd := exec.CommandContext(context.Background(), "go", cmdArgs...)
	cmdOutput, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Error executing 'go tool pprof': %v\nOutput:\n%s", err, string(cmdOutput))
		return nil, nil, fmt.Errorf("failed to generate flamegraph: %w. Output: %s", err, string(cmdOutput))
	}

	log.Printf("Successfully generated flamegraph: %s", args.OutputSVGPath)
	log.Printf("pprof output:\n%s", string(cmdOutput))

	resultText := fmt.Sprintf("火焰图已成功生成并保存到: %s", args.OutputSVGPath)

	svgBytes, readErr := os.ReadFile(args.OutputSVGPath)
	if readErr != nil {
		log.Printf("成功生成 SVG 文件 '%s' 但读取失败: %v", args.OutputSVGPath, readErr)
		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{
					
					Text: resultText,
				},
			},
		}, nil, nil
	}

	svgContentStr := string(svgBytes)
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				
				Text: resultText,
			},
			&mcp.TextContent{
				
				Text: svgContentStr,
			},
		},
	}, nil, nil
}

// DetectMemoryLeaksArgs 定义 detect_memory_leaks 工具的输入参数
type DetectMemoryLeaksArgs struct {
	OldProfileURI string  `json:"old_profile_uri" jsonschema:"description=较早的 heap profile 的 URI，支持 'file://', 'http://', 'https://' 协议"`
	NewProfileURI string  `json:"new_profile_uri" jsonschema:"description=较新的 heap profile 的 URI，支持 'file://', 'http://', 'https://' 协议"`
	Threshold     float64 `json:"threshold,omitempty" jsonschema:"description=检测内存泄漏的增长阈值 (0.1 表示 10%),default=0.1"`
	Limit         float64 `json:"limit,omitempty" jsonschema:"description=返回的潜在内存泄漏类型的最大数量,default=10"`
}

// handleDetectMemoryLeaks 处理内存泄漏检测的请求。
func handleDetectMemoryLeaks(_ context.Context, _ *mcp.CallToolRequest, args DetectMemoryLeaksArgs) (*mcp.CallToolResult, any, error) {
	if args.OldProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: old_profile_uri")
	}
	if args.NewProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: new_profile_uri")
	}

	// 设置默认值
	if args.Threshold == 0 {
		args.Threshold = 0.1 // Default 10% growth
	}
	if args.Limit == 0 {
		args.Limit = 10.0
	}

	limit := int(args.Limit)
	if limit <= 0 {
		limit = 10
	}

	log.Printf("Handling detect_memory_leaks: OldURI=%s, NewURI=%s, Threshold=%.2f, Limit=%d",
		args.OldProfileURI, args.NewProfileURI, args.Threshold, limit)

	// Get the old profile file
	oldFilePath, oldCleanup, err := getProfileAsFile(args.OldProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get old profile file: %w", err)
	}
	defer oldCleanup()

	oldFile, err := os.Open(oldFilePath)
	if err != nil {
		log.Printf("Error opening old profile file '%s': %v", oldFilePath, err)
		return nil, nil, fmt.Errorf("failed to open old profile file '%s': %w", oldFilePath, err)
	}
	defer oldFile.Close()

	oldProf, err := profile.Parse(oldFile)
	if err != nil {
		log.Printf("Error parsing old profile file '%s': %v", oldFilePath, err)
		return nil, nil, fmt.Errorf("failed to parse old profile file '%s': %w", oldFilePath, err)
	}
	log.Printf("Successfully parsed old profile file from path: %s", oldFilePath)

	// Get the new profile file
	newFilePath, newCleanup, err := getProfileAsFile(args.NewProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get new profile file: %w", err)
	}
	defer newCleanup()

	newFile, err := os.Open(newFilePath)
	if err != nil {
		log.Printf("Error opening new profile file '%s': %v", newFilePath, err)
		return nil, nil, fmt.Errorf("failed to open new profile file '%s': %w", newFilePath, err)
	}
	defer newFile.Close()

	newProf, err := profile.Parse(newFile)
	if err != nil {
		log.Printf("Error parsing new profile file '%s': %v", newFilePath, err)
		return nil, nil, fmt.Errorf("failed to parse new profile file '%s': %w", newFilePath, err)
	}
	log.Printf("Successfully parsed new profile file from path: %s", newFilePath)

	// Detect memory leaks
	result, err := analyzer.DetectPotentialMemoryLeaks(oldProf, newProf, args.Threshold, limit)
	if err != nil {
		log.Printf("Error detecting memory leaks: %v", err)
		return nil, nil, fmt.Errorf("failed to detect memory leaks: %w", err)
	}

	log.Printf("Memory leak detection completed successfully. Result length: %d", len(result))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				
				Text: result,
			},
		},
	}, nil, nil
}

// OpenInteractivePprofArgs 定义 open_interactive_pprof 工具的输入参数
type OpenInteractivePprofArgs struct {
	ProfileURI  string `json:"profile_uri" jsonschema:"description=要分析的 pprof 文件的 URI (支持 'file://', 'http://', 'https://' 或本地路径)"`
	HTTPAddress string `json:"http_address,omitempty" jsonschema:"description=指定 pprof Web UI 的监听地址和端口 (例如 ':8081')，如果省略默认为 ':8081'"`
}

// handleOpenInteractivePprof 处理打开交互式 pprof 的请求。
func handleOpenInteractivePprof(_ context.Context, _ *mcp.CallToolRequest, args OpenInteractivePprofArgs) (*mcp.CallToolResult, any, error) {
	if args.ProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_uri")
	}

	httpAddress := args.HTTPAddress
	if httpAddress == "" {
		httpAddress = ":8081" // 默认端口
		log.Printf("No http_address provided, using default: %s", httpAddress)
	}

	log.Printf("Handling open_interactive_pprof: URI=%s, Address=%s", args.ProfileURI, httpAddress)

	inputFilePath, cleanup, err := getProfileAsFile(args.ProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get profile file: %w", err)
	}
	// 注意：不能在这里 defer cleanup()，因为 pprof 进程需要持续访问文件

	cmdArgs := []string{"tool", "pprof"}
	cmdArgs = append(cmdArgs, fmt.Sprintf("-http=%s", httpAddress))
	cmdArgs = append(cmdArgs, inputFilePath)

	log.Printf("Preparing to execute command in background: go %s", strings.Join(cmdArgs, " "))

	_, err = exec.LookPath("go")
	if err != nil {
		log.Println("Error: 'go' command not found in PATH.")
		cleanup()
		return nil, nil, fmt.Errorf("'go' command not found in PATH, cannot start pprof")
	}

	cmd := exec.CommandContext(context.Background(), "go", cmdArgs...)
	err = cmd.Start()

	if err != nil {
		log.Printf("Error starting 'go tool pprof' in background: %v", err)
		cleanup()
		return nil, nil, fmt.Errorf("failed to start 'go tool pprof': %w", err)
	}

	pid := cmd.Process.Pid
	pprofMutex.Lock()
	runningPprofs[pid] = cmd.Process
	pprofMutex.Unlock()

	log.Printf("Successfully started 'go tool pprof' in background with PID: %d", pid)

	resultText := fmt.Sprintf("已成功在后台启动 'go tool pprof' (PID: %d) 来分析 '%s'", pid, inputFilePath)
	resultText += fmt.Sprintf("，监听地址约为 %s。", httpAddress)
	resultText += "\n你可以使用 'disconnect_pprof_session' 工具并提供 PID 来尝试终止此进程。"
	resultText += "\n注意：如果是远程 URL，下载的临时 pprof 文件在进程结束前不会被自动删除。"

	log.Println(resultText)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				
				Text: resultText,
			},
		},
	}, nil, nil
}

// DisconnectPprofSessionArgs 定义 disconnect_pprof_session 工具的输入参数
type DisconnectPprofSessionArgs struct {
	PID         float64 `json:"pid" jsonschema:"description=要终止的后台 pprof 进程的 PID (由 'open_interactive_pprof' 返回)"`
	HTTPAddress string  `json:"http_address,omitempty" jsonschema:"description=指定 pprof Web UI 的监听地址和端口 (例如 ':8081')，如果省略 pprof 会自动选择"`
}

// handleDisconnectPprofSession 处理断开 pprof 会话的请求。
func handleDisconnectPprofSession(_ context.Context, _ *mcp.CallToolRequest, args DisconnectPprofSessionArgs) (*mcp.CallToolResult, any, error) {
	if args.PID <= 0 {
		return nil, nil, fmt.Errorf("invalid PID: %d", int(args.PID))
	}

	pid := int(args.PID)
	log.Printf("Handling disconnect_pprof_session for PID: %d", pid)

	pprofMutex.Lock()
	process, exists := runningPprofs[pid]
	if !exists {
		pprofMutex.Unlock()
		log.Printf("PID %d not found in running pprof sessions.", pid)
		return nil, nil, fmt.Errorf("未找到 PID 为 %d 的正在运行的 pprof 会话", pid)
	}
	delete(runningPprofs, pid)
	pprofMutex.Unlock()

	log.Printf("Attempting to terminate process with PID: %d", pid)
	err := process.Signal(os.Interrupt)
	if err != nil {
		log.Printf("Failed to send Interrupt signal to PID %d: %v. Trying Kill signal.", pid, err)
		err = process.Signal(os.Kill)
		if err != nil {
			log.Printf("Failed to send Kill signal to PID %d: %v", pid, err)
			return nil, nil, fmt.Errorf("尝试终止 PID %d 失败：%w", pid, err)
		}
	}

	_, err = process.Wait()
	if err != nil && !strings.Contains(err.Error(), "wait: no child processes") && !strings.Contains(err.Error(), "signal:") {
		log.Printf("Warning: Error waiting for process PID %d after signaling: %v", pid, err)
	}

	resultText := fmt.Sprintf("已成功向 PID %d 发送终止信号。", pid)
	log.Println(resultText)

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				
				Text: resultText,
			},
		},
	}, nil, nil
}

// CompareProfilesArgs 定义 compare_profiles 工具的输入参数
type CompareProfilesArgs struct {
	BaselineProfileURI string  `json:"baseline_profile_uri" jsonschema:"description=基线 profile 的 URI (旧版本)，支持 'file://', 'http://', 'https://' 协议"`
	TargetProfileURI   string  `json:"target_profile_uri" jsonschema:"description=目标 profile 的 URI (新版本)，支持 'file://', 'http://', 'https://' 协议"`
	ProfileType        string  `json:"profile_type" jsonschema:"description=要比较的 pprof profile 的类型,enum=cpu,enum=heap,enum=allocs,enum=mutex,enum=block"`
	TopN               float64 `json:"top_n,omitempty" jsonschema:"description=返回结果的数量上限, default=10"`
	OutputFormat       string  `json:"output_format,omitempty" jsonschema:"description=输出格式,enum=text,enum=markdown,enum=json,default=markdown"`
}

// handleCompareProfiles 处理 profile 比较的请求。
func handleCompareProfiles(_ context.Context, _ *mcp.CallToolRequest, args CompareProfilesArgs) (*mcp.CallToolResult, any, error) {
	if args.BaselineProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: baseline_profile_uri")
	}
	if args.TargetProfileURI == "" {
		return nil, nil, fmt.Errorf("missing required argument: target_profile_uri")
	}
	if args.ProfileType == "" {
		return nil, nil, fmt.Errorf("missing required argument: profile_type")
	}

	// 设置默认值
	if args.TopN <= 0 {
		args.TopN = 10
	}
	if args.OutputFormat == "" {
		args.OutputFormat = "markdown"
	}

	topN := int(args.TopN)
	log.Printf("Handling compare_profiles: Baseline=%s, Target=%s, Type=%s, TopN=%d, Format=%s",
		args.BaselineProfileURI, args.TargetProfileURI, args.ProfileType, topN, args.OutputFormat)

	// 获取基线 profile
	baselinePath, baselineCleanup, err := getProfileAsFile(args.BaselineProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get baseline profile file: %w", err)
	}
	defer baselineCleanup()

	baselineFile, err := os.Open(baselinePath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open baseline profile file '%s': %w", baselinePath, err)
	}
	defer baselineFile.Close()

	baselineProf, err := profile.Parse(baselineFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse baseline profile file '%s': %w", baselinePath, err)
	}

	// 获取目标 profile
	targetPath, targetCleanup, err := getProfileAsFile(args.TargetProfileURI)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get target profile file: %w", err)
	}
	defer targetCleanup()

	targetFile, err := os.Open(targetPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to open target profile file '%s': %w", targetPath, err)
	}
	defer targetFile.Close()

	targetProf, err := profile.Parse(targetFile)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse target profile file '%s': %w", targetPath, err)
	}

	// 执行比较
	result, err := analyzer.CompareProfiles(baselineProf, targetProf, args.ProfileType, topN, args.OutputFormat)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compare profiles: %w", err)
	}

	log.Printf("Profile comparison completed successfully. Result length: %d", len(result))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, nil, nil
}

// AnalyzeHeapTimeSeriesArgs 定义 analyze_heap_time_series 工具的输入参数
type AnalyzeHeapTimeSeriesArgs struct {
	ProfileURIs []string `json:"profile_uris" jsonschema:"description=多个 heap profile 的 URI 数组（按时间顺序），支持 'file://', 'http://', 'https://' 协议"`
	Labels      []string `json:"labels,omitempty" jsonschema:"description=每个时间点的标签数组（可选），长度必须与 profile_uris 相同"`
	OutputFormat string   `json:"output_format,omitempty" jsonschema:"description=输出格式,enum=text,enum=markdown,enum=json,default=markdown"`
}

// handleAnalyzeHeapTimeSeries 处理内存时序分析的请求。
func handleAnalyzeHeapTimeSeries(_ context.Context, _ *mcp.CallToolRequest, args AnalyzeHeapTimeSeriesArgs) (*mcp.CallToolResult, any, error) {
	if len(args.ProfileURIs) < 3 {
		return nil, nil, fmt.Errorf("至少需要 3 个 profile 来进行时序分析，当前只有 %d 个", len(args.ProfileURIs))
	}

	// 设置默认值
	if args.OutputFormat == "" {
		args.OutputFormat = "markdown"
	}

	// 如果没有提供标签，生成默认标签
	labels := args.Labels
	if len(labels) == 0 {
		labels = make([]string, len(args.ProfileURIs))
		for i := range args.ProfileURIs {
			labels[i] = fmt.Sprintf("T%d", i+1)
		}
	} else if len(labels) != len(args.ProfileURIs) {
		return nil, nil, fmt.Errorf("标签数量 (%d) 与 profile 数量 (%d) 不匹配", len(labels), len(args.ProfileURIs))
	}

	log.Printf("Handling analyze_heap_time_series: profiles=%d, format=%s", len(args.ProfileURIs), args.OutputFormat)

	// 解析所有 profile
	profiles := make([]*profile.Profile, len(args.ProfileURIs))
	for i, uri := range args.ProfileURIs {
		filePath, cleanup, err := getProfileAsFile(uri)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get profile file #%d: %w", i+1, err)
		}
		defer cleanup()

		file, err := os.Open(filePath)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to open profile file #%d '%s': %w", i+1, filePath, err)
		}
		defer file.Close()

		prof, err := profile.Parse(file)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse profile file #%d '%s': %w", i+1, filePath, err)
		}

		profiles[i] = prof
		log.Printf("Successfully parsed profile #%d: %d samples", i+1, len(prof.Sample))
	}

	// 执行时序分析
	result, err := analyzer.AnalyzeHeapTimeSeries(profiles, labels, args.OutputFormat)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to analyze time series: %w", err)
	}

	log.Printf("Heap time series analysis completed successfully. Result length: %d", len(result))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			&mcp.TextContent{
				Text: result,
			},
		},
	}, nil, nil
}

// getMimeTypeForFormat 根据输出格式返回对应的 MIME 类型
func getMimeTypeForFormat(format string) string {
	switch format {
	case "text":
		return "text/plain"
	case "markdown":
		return "text/markdown"
	case "json", "flamegraph-json":
		return "application/json"
	default:
		return "text/plain"
	}
}
