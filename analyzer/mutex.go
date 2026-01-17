package analyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// MutexContentionStat 代表 Mutex 竞争的统计信息
type MutexContentionStat struct {
	FunctionName     string  `json:"functionName"`
	Contentions      int64   `json:"contentions"`       // 竞争次数
	DelayNanos       int64   `json:"delayNanos"`        // 总延迟时间（纳秒）
	DelayFormatted   string  `json:"delayFormatted"`    // 格式化后的延迟时间
	ContentionsPct   float64 `json:"contentionsPct"`    // 竞争次数占比
	DelayPct         float64 `json:"delayPct"`          // 延迟时间占比
	AvgDelayNanos    int64   `json:"avgDelayNanos"`     // 平均每次竞争的延迟（纳秒）
	AvgDelayFormatted string `json:"avgDelayFormatted"` // 格式化后的平均延迟
}

// MutexAnalysisResult 代表 Mutex 分析的整体结果 (JSON)
type MutexAnalysisResult struct {
	ProfileType         string                `json:"profileType"`
	TotalContentions    int64                 `json:"totalContentions"`
	TotalDelayNanos     int64                 `json:"totalDelayNanos"`
	TotalDelayFormatted string                `json:"totalDelayFormatted"`
	TopN                int                   `json:"topN"`
	Contentions         []MutexContentionStat `json:"contentions"`
}

// AnalyzeMutexProfile 分析 Mutex profile 文件并返回格式化结果。
func AnalyzeMutexProfile(p *profile.Profile, topN int, format string) (string, error) {
	log.Printf("Analyzing Mutex profile (Top %d, Format: %s)", topN, format)

	// --- 1. 确定用于分析的值的索引 ---
	// Mutex profile 有两个样本类型：
	// - contentions (count): 锁竞争次数
	// - delay (nanoseconds): 等待锁的总延迟时间
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

	log.Printf("使用索引 %d (contentions) 和 %d (delay) 进行 Mutex 分析", contentionIndex, delayIndex)

	// --- 2. 按函数聚合竞争统计 ---
	contentionData := make(map[string]*MutexContentionStat)
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
		if stat, exists := contentionData[functionName]; exists {
			stat.Contentions += contentions
			stat.DelayNanos += delay
		} else {
			contentionData[functionName] = &MutexContentionStat{
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
		return "Mutex profile 分析完成：未发现锁竞争。", nil
	}

	// --- 3. 按延迟时间排序（优先显示延迟最长的函数）---
	stats := make([]*MutexContentionStat, 0, len(contentionData))
	for _, stat := range contentionData {
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
		contentions := make([]MutexContentionStat, len(stats))
		for i, stat := range stats {
			contentions[i] = *stat
		}
		result := MutexAnalysisResult{
			ProfileType:         "mutex",
			TotalContentions:    totalContentions,
			TotalDelayNanos:     totalDelay,
			TotalDelayFormatted: formatNanos(totalDelay),
			TopN:                topN,
			Contentions:         contentions,
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
		b.WriteString("# Mutex Profile 分析报告\n\n")
		b.WriteString(fmt.Sprintf("**总竞争次数**: %s\n", formatNumber(totalContentions)))
		b.WriteString(fmt.Sprintf("**总延迟时间**: %s\n\n", formatNanos(totalDelay)))
		b.WriteString("## Top Mutex 竞争点\n\n")
		b.WriteString("| 排名 | 函数名 | 竞争次数 | 竞争占比 | 总延迟 | 延迟占比 | 平均延迟 |\n")
		b.WriteString("|------|--------|----------|----------|--------|----------|----------|\n")
	} else {
		b.WriteString("Mutex Profile 分析结果\n")
		b.WriteString("========================\n\n")
		b.WriteString(fmt.Sprintf("总竞争次数: %s\n", formatNumber(totalContentions)))
		b.WriteString(fmt.Sprintf("总延迟时间: %s\n\n", formatNanos(totalDelay)))
		b.WriteString("Top Mutex 竞争点:\n")
		b.WriteString(strings.Repeat("-", 120) + "\n")
		b.WriteString(fmt.Sprintf("%-6s %-50s %12s %10s %12s %10s %12s\n",
			"排名", "函数名", "竞争次数", "竞争占比", "总延迟", "延迟占比", "平均延迟"))
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
	b.WriteString("- 关注总延迟时间最长的函数，这些是性能瓶颈的根源\n")
	b.WriteString("- 高竞争次数但低延迟可能表明锁粒度过小，频繁获取/释放\n")
	b.WriteString("- 考虑使用细粒度锁、读写锁 (sync.RWMutex) 或无锁数据结构来减少竞争\n")

	if format == "markdown" {
		b.WriteString("\n```")
	}

	return b.String(), nil
}

// formatNanos 将纳秒数格式化为可读的时间字符串
func formatNanos(nanos int64) string {
	if nanos < 1000 {
		return fmt.Sprintf("%d ns", nanos)
	}
	micros := nanos / 1000
	if micros < 1000 {
		return fmt.Sprintf("%.2f μs", float64(nanos)/1000)
	}
	millis := micros / 1000
	if millis < 1000 {
		return fmt.Sprintf("%.2f ms", float64(micros)/1000)
	}
	seconds := millis / 1000
	if seconds < 60 {
		return fmt.Sprintf("%.2f s", float64(millis)/1000)
	}
	minutes := seconds / 60
	secondsRemainder := seconds % 60
	return fmt.Sprintf("%d m %d s", minutes, secondsRemainder)
}
