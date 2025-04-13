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
	"github.com/mark3labs/mcp-go/mcp"

	"github.com/ZephyrDeng/pprof-analyzer-mcp/analyzer"
)

// handleAnalyzePprof 处理分析 pprof 文件的请求。
func handleAnalyzePprof(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	profileURIStr, ok := args["profile_uri"].(string)
	if !ok || profileURIStr == "" {
		return nil, fmt.Errorf("missing or invalid required argument: profile_uri (string)")
	}
	profileType, ok := args["profile_type"].(string)
	if !ok || profileType == "" {
		return nil, fmt.Errorf("missing or invalid required argument: profile_type (string)")
	}
	outputFormat, ok := args["output_format"].(string)
	if !ok {
		outputFormat = "text"
	}
	topNFloat, ok := args["top_n"].(float64)
	if !ok {
		topNFloat = 5.0
	}
	topN := int(topNFloat)
	if topN <= 0 {
		topN = 5
	}

	log.Printf("Handling analyze_pprof: URI=%s, Type=%s, TopN=%d, Format=%s", profileURIStr, profileType, topN, outputFormat)

	filePath, cleanup, err := getProfileAsFile(profileURIStr) // Calls function from profile_utils.go
	if err != nil {
		return nil, fmt.Errorf("failed to get profile file: %w", err)
	}
	defer cleanup()

	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("Error opening profile file '%s' (might be temporary): %v", filePath, err)
		return nil, fmt.Errorf("failed to open profile file '%s': %w", filePath, err)
	}
	defer file.Close()

	prof, err := profile.Parse(file)
	if err != nil {
		log.Printf("Error parsing profile file '%s': %v", filePath, err)
		return nil, fmt.Errorf("failed to parse profile file '%s': %w", filePath, err)
	}
	log.Printf("Successfully parsed profile file from path: %s", filePath)

	var analysisResult string
	var analysisErr error

	switch profileType {
	case "cpu":
		analysisResult, analysisErr = analyzer.AnalyzeCPUProfile(prof, topN, outputFormat)
	case "heap":
		analysisResult, analysisErr = analyzer.AnalyzeHeapProfile(prof, topN, outputFormat)
	case "goroutine":
		analysisResult, analysisErr = analyzer.AnalyzeGoroutineProfile(prof, topN, outputFormat)
	case "allocs":
		analysisResult, analysisErr = analyzer.AnalyzeAllocsProfile(prof, topN, outputFormat)
	case "mutex":
		analysisResult, analysisErr = analyzer.AnalyzeMutexProfile(prof, topN, outputFormat)
	case "block":
		analysisResult, analysisErr = analyzer.AnalyzeBlockProfile(prof, topN, outputFormat)
	default:
		analysisErr = fmt.Errorf("unsupported profile type: '%s'", profileType)
	}

	if analysisErr != nil {
		log.Printf("Analysis error for type '%s': %v", profileType, analysisErr)
		return nil, analysisErr
	}

	log.Printf("Analysis successful for type '%s'. Result length: %d", profileType, len(analysisResult))
	return &mcp.CallToolResult{
		Content: []mcp.Content{
			mcp.TextContent{
				Type: "text",
				Text: analysisResult,
			},
		},
	}, nil
}

// handleGenerateFlamegraph 处理生成火焰图的请求。
func handleGenerateFlamegraph(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := request.Params.Arguments

	profileURIStr, ok := args["profile_uri"].(string)
	if !ok || profileURIStr == "" {
		return nil, fmt.Errorf("missing or invalid required argument: profile_uri (string)")
	}
	profileType, ok := args["profile_type"].(string)
	if !ok || profileType == "" {
		return nil, fmt.Errorf("missing or invalid required argument: profile_type (string)")
	}
	outputSvgPath, ok := args["output_svg_path"].(string)
	if !ok || outputSvgPath == "" {
		return nil, fmt.Errorf("missing or invalid required argument: output_svg_path (string)")
	}

	log.Printf("Handling generate_flamegraph: URI=%s, Type=%s, Output=%s", profileURIStr, profileType, outputSvgPath)

	inputFilePath, cleanup, err := getProfileAsFile(profileURIStr) // Calls function from profile_utils.go
	if err != nil {
		return nil, fmt.Errorf("failed to get profile file for flamegraph: %w", err)
	}
	defer cleanup()

	if !filepath.IsAbs(outputSvgPath) {
		cwd, err := os.Getwd()
		if err != nil {
			log.Printf("无法获取当前工作目录: %v", err)
		} else {
			outputSvgPath = filepath.Join(cwd, outputSvgPath)
			log.Printf("将相对输出路径转换为绝对路径: %s", outputSvgPath)
		}
	}

	cmdArgs := []string{"tool", "pprof"}
	switch profileType {
	case "heap":
		cmdArgs = append(cmdArgs, "-inuse_space")
	case "allocs":
		cmdArgs = append(cmdArgs, "-alloc_space")
	case "cpu", "goroutine", "mutex", "block":
		// No extra flags needed
	default:
		return nil, fmt.Errorf("unsupported profile type for flamegraph: '%s'", profileType)
	}
	cmdArgs = append(cmdArgs, "-svg", "-output", outputSvgPath, inputFilePath)

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
		return nil, fmt.Errorf(errMsg)
	}
	log.Println("Graphviz (dot) found.")

	cmd := exec.CommandContext(ctx, "go", cmdArgs...)
	cmdOutput, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Error executing 'go tool pprof': %v\nOutput:\n%s", err, string(cmdOutput))
		return nil, fmt.Errorf("failed to generate flamegraph: %w. Output: %s", err, string(cmdOutput))
	}

	log.Printf("Successfully generated flamegraph: %s", outputSvgPath)
	log.Printf("pprof output:\n%s", string(cmdOutput))

	resultText := fmt.Sprintf("火焰图已成功生成并保存到: %s", outputSvgPath)
	textContent := mcp.TextContent{
		Type: "text",
		Text: resultText,
	}

	svgBytes, readErr := os.ReadFile(outputSvgPath)
	if readErr != nil {
		log.Printf("成功生成 SVG 文件 '%s' 但读取失败: %v", outputSvgPath, readErr)
		return &mcp.CallToolResult{
			Content: []mcp.Content{textContent},
		}, nil
	}

	svgContentStr := string(svgBytes)
	svgContent := mcp.TextContent{
		Type: "text",
		Text: svgContentStr,
	}

	return &mcp.CallToolResult{
		Content: []mcp.Content{
			textContent,
			svgContent,
		},
	}, nil
}
