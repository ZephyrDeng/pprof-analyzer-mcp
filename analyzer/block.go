package analyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// BlockContentionStat 代表 Block 阻塞的统计信息
type BlockContentionStat struct {
	FunctionName     string  `json:"functionName"`
	Contentions      int64   `json:"contentions"`       // 阻塞次数
	DelayNanos       int64   `json:"delayNanos"`        // 总延迟时间（纳秒）
	DelayFormatted   string  `json:"delayFormatted"`    // 格式化后的延迟时间
	ContentionsPct   float64 `json:"contentionsPct"`    // 阻塞次数占比
	DelayPct         float64 `json:"delayPct"`          // 延迟时间占比
	AvgDelayNanos    int64   `json:"avgDelayNanos"`     // 平均每次阻塞的延迟（纳秒）
	AvgDelayFormatted string `json:"avgDelayFormatted"` // 格式化后的平均延迟
}

// BlockAnalysisResult 代表 Block 分析的整体结果 (JSON)
type BlockAnalysisResult struct {
	ProfileType         string                `json:"profileType"`
	TotalContentions    int64                 `json:"totalContentions"`
	TotalDelayNanos     int64                 `json:"totalDelayNanos"`
	TotalDelayFormatted string                `json:"totalDelayFormatted"`
	TopN                int                   `json:"topN"`
	Blocks              []BlockContentionStat `json:"blocks"`
}

// AnalyzeBlockProfile 分析 Block profile 文件并返回格式化结果。
func AnalyzeBlockProfile(p *profile.Profile, topN int, format string) (string, error) {
	log.Printf("Analyzing Block profile (Top %d, Format: %s)", topN, format)

	// --- 1. 确定用于分析的值的索引 ---
	// Block profile 有两个样本类型：
	// - contentions (count): 阻塞次数
	// - delay (nanoseconds): 阻塞等待的总延迟时间
	contentionIndex := -1
	delayIndex := -1

	for i, st := range p.SampleType {
		switch st.Type {
		case "contentions":
			contentionIndex = i
		case "delay":
			delayIndex = i
		}
	}

	if contentionIndex == -1 || delayIndex == -1 {
		return "", fmt.Errorf("无法从 profile 中找到必需的样本类型 (contentions, delay)")
	}

	log.Printf("使用索引 %d (contentions) 和 %d (delay) 进行 Block 分析", contentionIndex, delayIndex)

	// --- 2. 按函数聚合阻塞统计 ---
	blockData := make(map[string]*BlockContentionStat)
	totalContentions := int64(0)
	totalDelay := int64(0)

	for _, s := range p.Sample {
		if len(s.Location) == 0 || len(s.Value) <= max(contentionIndex, delayIndex) {
			continue
		}

		contentions := s.Value[contentionIndex]
		delay := s.Value[delayIndex]

		// 获取最顶层函数名
		loc := s.Location[0]
		functionName := ""
		for _, line := range loc.Line {
			if line.Function != nil {
				functionName = line.Function.Name
				break
			}
		}

		if functionName == "" {
			functionName = "unknown"
		}

		// 聚合统计数据
		if stat, exists := blockData[functionName]; exists {
			stat.Contentions += contentions
			stat.DelayNanos += delay
		} else {
			blockData[functionName] = &BlockContentionStat{
				FunctionName:  functionName,
				Contentions:   contentions,
				DelayNanos:    delay,
				AvgDelayNanos: delay / contentions, // 计算平均延迟
			}
		}

		totalContentions += contentions
		totalDelay += delay
	}

	if totalContentions == 0 {
		return "Block profile 分析完成：未发现阻塞操作。", nil
	}

	// --- 3. 按延迟时间排序（优先显示延迟最长的函数）---
	stats := make([]*BlockContentionStat, 0, len(blockData))
	for _, stat := range blockData {
		// 计算百分比
		stat.ContentionsPct = float64(stat.Contentions) / float64(totalContentions) * 100
		stat.DelayPct = float64(stat.DelayNanos) / float64(totalDelay) * 100
		// 格式化时间
		stat.DelayFormatted = formatNanos(stat.DelayNanos)
		stat.AvgDelayFormatted = formatNanos(stat.AvgDelayNanos)
		stats = append(stats, stat)
	}

	sort.Slice(stats, func(i, j int) bool {
		return stats[i].DelayNanos > stats[j].DelayNanos // 按延迟降序
	})

	// --- 4. 格式化输出 ---
	if format == "json" {
		// 将指针切片转换为值切片
		blocks := make([]BlockContentionStat, len(stats))
		for i, stat := range stats {
			blocks[i] = *stat
		}
		result := BlockAnalysisResult{
			ProfileType:         "block",
			TotalContentions:    totalContentions,
			TotalDelayNanos:     totalDelay,
			TotalDelayFormatted: formatNanos(totalDelay),
			TopN:                topN,
			Blocks:              blocks,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Text/Markdown 输出
	var b strings.Builder

	if format == "markdown" {
		b.WriteString("# Block Profile 分析报告\n\n")
		b.WriteString(fmt.Sprintf("**总阻塞次数**: %s\n", formatNumber(totalContentions)))
		b.WriteString(fmt.Sprintf("**总延迟时间**: %s\n\n", formatNanos(totalDelay)))
		b.WriteString("## Top 阻塞点\n\n")
		b.WriteString("| 排名 | 函数名 | 阻塞次数 | 阻塞占比 | 总延迟 | 延迟占比 | 平均延迟 |\n")
		b.WriteString("|------|--------|----------|----------|--------|----------|----------|\n")
	} else {
		b.WriteString("Block Profile 分析结果\n")
		b.WriteString("========================\n\n")
		b.WriteString(fmt.Sprintf("总阻塞次数: %s\n", formatNumber(totalContentions)))
		b.WriteString(fmt.Sprintf("总延迟时间: %s\n\n", formatNanos(totalDelay)))
		b.WriteString("Top 阻塞点:\n")
		b.WriteString(strings.Repeat("-", 120) + "\n")
		b.WriteString(fmt.Sprintf("%-6s %-50s %12s %10s %12s %10s %12s\n",
			"排名", "函数名", "阻塞次数", "阻塞占比", "总延迟", "延迟占比", "平均延迟"))
		b.WriteString(strings.Repeat("-", 120) + "\n")
	}

	limit := topN
	if limit > len(stats) {
		limit = len(stats)
	}

	for i := 0; i < limit; i++ {
		stat := stats[i]
		if format == "markdown" {
			b.WriteString(fmt.Sprintf("| %d | `%s` | %s | %.2f%% | %s | %.2f%% | %s |\n",
				i+1,
				truncateString(stat.FunctionName, 40),
				formatNumber(stat.Contentions),
				stat.ContentionsPct,
				stat.DelayFormatted,
				stat.DelayPct,
				stat.AvgDelayFormatted,
			))
		} else {
			b.WriteString(fmt.Sprintf("%-6d %-50s %12s %9.2f%% %12s %9.2f%% %12s\n",
				i+1,
				truncateString(stat.FunctionName, 50),
				formatNumber(stat.Contentions),
				stat.ContentionsPct,
				stat.DelayFormatted,
				stat.DelayPct,
				stat.AvgDelayFormatted,
			))
		}
	}

	b.WriteString("\n**分析建议**:\n")
	b.WriteString("- 关注总延迟时间最长的函数，这些可能是通道操作、网络 I/O 或系统调用导致的阻塞\n")
	b.WriteString("- 高阻塞次数但低延迟可能表明频繁但短暂的阻塞操作（如无缓冲通道的发送/接收）\n")
	b.WriteString("- 考虑使用带缓冲的通道、超时机制或异步处理来减少阻塞\n")
	b.WriteString("- 检查是否有 goroutine 泄漏导致资源耗尽\n")

	if format == "markdown" {
		b.WriteString("\n```")
	}

	return b.String(), nil
}

// max 返回两个整数中的最大值
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
