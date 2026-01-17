package analyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strings"

	"github.com/google/pprof/profile"
)

// DiffResult 表示两个 profile 之间的差异结果
type DiffResult struct {
	ProfileType     string           `json:"profileType"`
	BaselineURI     string           `json:"baselineUri"`
	TargetURI       string           `json:"targetUri"`
	TopN            int              `json:"topN"`
	Functions       []FunctionDiff   `json:"functions"`
	Summary         DiffSummary      `json:"summary"`
}

// FunctionDiff 表示单个函数的差异统计
type FunctionDiff struct {
	FunctionName       string  `json:"functionName"`
	BaselineValue      int64   `json:"baselineValue"`
	TargetValue        int64   `json:"targetValue"`
	DiffValue          int64   `json:"diffValue"`
	DiffPercentage     float64 `json:"diffPercentage"`
	BaselineFormatted  string  `json:"baselineFormatted"`
	TargetFormatted    string  `json:"targetFormatted"`
	DiffFormatted      string  `json:"diffFormatted"`
}

// DiffSummary 提供差异分析的总体摘要
type DiffSummary struct {
	BaselineTotal      int64   `json:"baselineTotal"`
	TargetTotal        int64   `json:"targetTotal"`
	TotalDiff          int64   `json:"totalDiff"`
	TotalDiffPercent   float64 `json:"totalDiffPercent"`
	ImprovedFuncs      int     `json:"improvedFuncs"`    // 性能提升的函数数量
	RegressedFuncs     int     `json:"regressedFuncs"`   // 性能回归的函数数量
	AddedFuncs         int     `json:"addedFuncs"`       // 新增的函数
	RemovedFuncs       int     `json:"removedFuncs"`     // 移除的函数
}

// CompareProfiles 比较两个 profile 并生成差异分析
func CompareProfiles(baseline, target *profile.Profile, profileTypeName string, topN int, format string) (string, error) {
	log.Printf("Comparing profiles: type=%s, baseline samples=%d, target samples=%d",
		profileTypeName, len(baseline.Sample), len(target.Sample))

	// 确定要比较的值索引
	valueIndex, err := getValueIndex(baseline, profileTypeName)
	if err != nil {
		return "", err
	}

	// 聚合 baseline 和 target 的函数级统计
	baselineFuncs := aggregateFunctionValues(baseline, valueIndex)
	targetFuncs := aggregateFunctionValues(target, valueIndex)

	// 计算差异
	diffs := computeFunctionDiffs(baselineFuncs, targetFuncs)

	// 按差异绝对值排序（最大的变化排在前面）
	sort.Slice(diffs, func(i, j int) bool {
		return math.Abs(diffs[i].DiffPercentage) > math.Abs(diffs[j].DiffPercentage)
	})

	// 计算总体摘要
	summary := computeDiffSummary(baselineFuncs, targetFuncs, diffs)

	// 格式化输出
	if format == "json" {
		result := DiffResult{
			ProfileType: profileTypeName,
			BaselineURI: "baseline",
			TargetURI:   "target",
			TopN:        topN,
			Functions:   diffs,
			Summary:     summary,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Text/Markdown 输出
	return formatDiffReport(diffs, summary, profileTypeName, topN, format), nil
}

// getValueIndex 根据profile类型获取值的索引
func getValueIndex(p *profile.Profile, profileType string) (int, error) {
	for i, st := range p.SampleType {
		switch profileType {
		case "cpu":
			if st.Type == "cpu" || (st.Type == "samples" && st.Unit == "nanoseconds") {
				return i, nil
			}
		case "heap", "allocs":
			if st.Type == "inuse_space" || st.Type == "alloc_space" {
				return i, nil
			}
		case "mutex", "block":
			if st.Type == "delay" {
				return i, nil
			}
		}
	}

	// 如果没找到特定的，使用第二个值（通常是延迟/空间）
	if len(p.SampleType) > 1 {
		return 1, nil
	}
	return 0, nil
}

// aggregateFunctionValues 聚合函数级别的值
func aggregateFunctionValues(p *profile.Profile, valueIndex int) map[string]int64 {
	result := make(map[string]int64)

	for _, sample := range p.Sample {
		if len(sample.Location) == 0 || len(sample.Value) <= valueIndex {
			continue
		}

		value := sample.Value[valueIndex]

		// 获取最顶层函数名
		loc := sample.Location[0]
		functionName := "unknown"
		for _, line := range loc.Line {
			if line.Function != nil {
				functionName = line.Function.Name
				break
			}
		}

		result[functionName] += value
	}

	return result
}

// computeFunctionDiffs 计算函数差异
func computeFunctionDiffs(baselineFuncs, targetFuncs map[string]int64) []FunctionDiff {
	var diffs []FunctionDiff

	// 收集所有函数名
	allFuncs := make(map[string]bool)
	for name := range baselineFuncs {
		allFuncs[name] = true
	}
	for name := range targetFuncs {
		allFuncs[name] = true
	}

	for name := range allFuncs {
		baselineVal := baselineFuncs[name]
		targetVal := targetFuncs[name]

		diff := targetVal - baselineVal
		var diffPercent float64
		if baselineVal > 0 {
			diffPercent = float64(diff) / float64(baselineVal) * 100
		} else if targetVal > 0 {
			diffPercent = 100.0 // 新增的函数
		}

		diffs = append(diffs, FunctionDiff{
			FunctionName:      name,
			BaselineValue:     baselineVal,
			TargetValue:       targetVal,
			DiffValue:         diff,
			DiffPercentage:    diffPercent,
			BaselineFormatted: formatValue(baselineVal),
			TargetFormatted:   formatValue(targetVal),
			DiffFormatted:     formatDiffValue(diff),
		})
	}

	return diffs
}

// computeDiffSummary 计算总体摘要
func computeDiffSummary(baselineFuncs, targetFuncs map[string]int64, diffs []FunctionDiff) DiffSummary {
	baselineTotal := int64(0)
	targetTotal := int64(0)

	for _, v := range baselineFuncs {
		baselineTotal += v
	}
	for _, v := range targetFuncs {
		targetTotal += v
	}

	totalDiff := targetTotal - baselineTotal
	totalDiffPercent := 0.0
	if baselineTotal > 0 {
		totalDiffPercent = float64(totalDiff) / float64(baselineTotal) * 100
	}

	improved := 0
	regressed := 0
	added := 0
	removed := 0

	for _, diff := range diffs {
		if diff.BaselineValue == 0 && diff.TargetValue > 0 {
			added++
		} else if diff.TargetValue == 0 && diff.BaselineValue > 0 {
			removed++
		} else if diff.DiffValue < 0 {
			improved++
		} else if diff.DiffValue > 0 {
			regressed++
		}
	}

	return DiffSummary{
		BaselineTotal:    baselineTotal,
		TargetTotal:      targetTotal,
		TotalDiff:        totalDiff,
		TotalDiffPercent: totalDiffPercent,
		ImprovedFuncs:    improved,
		RegressedFuncs:   regressed,
		AddedFuncs:       added,
		RemovedFuncs:     removed,
	}
}

// formatDiffReport 格式化差异报告
func formatDiffReport(diffs []FunctionDiff, summary DiffSummary, profileType string, topN int, format string) string {
	var b strings.Builder

	if format == "markdown" {
		b.WriteString(fmt.Sprintf("# Profile 差异分析报告 (%s)\n\n", profileType))
		b.WriteString("## 总体摘要\n\n")
		b.WriteString(fmt.Sprintf("- **Baseline 总值**: %s\n", formatValue(summary.BaselineTotal)))
		b.WriteString(fmt.Sprintf("- **Target 总值**: %s\n", formatValue(summary.TargetTotal)))
		b.WriteString(fmt.Sprintf("- **总差异**: %s (%.2f%%)\n\n", formatDiffValue(summary.TotalDiff), summary.TotalDiffPercent))
		b.WriteString(fmt.Sprintf("- **性能提升**: %d 个函数\n", summary.ImprovedFuncs))
		b.WriteString(fmt.Sprintf("- **性能回归**: %d 个函数\n", summary.RegressedFuncs))
		b.WriteString(fmt.Sprintf("- **新增函数**: %d 个\n", summary.AddedFuncs))
		b.WriteString(fmt.Sprintf("- **移除函数**: %d 个\n\n", summary.RemovedFuncs))
		b.WriteString("## Top 变化函数\n\n")
		b.WriteString("| 排名 | 函数名 | Baseline | Target | 差异 | 变化%% |\n")
		b.WriteString("|------|--------|----------|--------|------|-------|\n")
	} else {
		b.WriteString(fmt.Sprintf("Profile 差异分析报告 (%s)\n", profileType))
		b.WriteString("==============================\n\n")
		b.WriteString("总体摘要:\n")
		b.WriteString(fmt.Sprintf("  Baseline 总值: %s\n", formatValue(summary.BaselineTotal)))
		b.WriteString(fmt.Sprintf("  Target 总值:   %s\n", formatValue(summary.TargetTotal)))
		b.WriteString(fmt.Sprintf("  总差异:         %s (%.2f%%)\n\n", formatDiffValue(summary.TotalDiff), summary.TotalDiffPercent))
		b.WriteString(fmt.Sprintf("  性能提升: %d 个函数\n", summary.ImprovedFuncs))
		b.WriteString(fmt.Sprintf("  性能回归: %d 个函数\n", summary.RegressedFuncs))
		b.WriteString(fmt.Sprintf("  新增函数: %d 个\n", summary.AddedFuncs))
		b.WriteString(fmt.Sprintf("  移除函数: %d 个\n\n", summary.RemovedFuncs))
		b.WriteString("Top 变化函数:\n")
		b.WriteString(strings.Repeat("-", 140) + "\n")
		b.WriteString(fmt.Sprintf("%-6s %-50s %15s %15s %15s %10s\n",
			"排名", "函数名", "Baseline", "Target", "差异", "变化%"))
		b.WriteString(strings.Repeat("-", 140) + "\n")
	}

	limit := topN
	if limit > len(diffs) {
		limit = len(diffs)
	}

	for i := 0; i < limit; i++ {
		diff := diffs[i]
		if format == "markdown" {
			indicator := "🟢"
			if diff.DiffValue > 0 {
				indicator = "🔴"
			} else if diff.BaselineValue == 0 && diff.TargetValue > 0 {
				indicator = "🆕"
			} else if diff.TargetValue == 0 && diff.BaselineValue > 0 {
				indicator = "❌"
			}

			b.WriteString(fmt.Sprintf("| %d | %s `%s` | %s | %s | %s | %.2f%% |\n",
				i+1, indicator, truncateString(diff.FunctionName, 40),
				diff.BaselineFormatted, diff.TargetFormatted,
				diff.DiffFormatted, diff.DiffPercentage))
		} else {
			indicator := ""
			if diff.DiffValue > 0 {
				indicator = " ⬆"
			} else if diff.DiffValue < 0 {
				indicator = " ⬇"
			}

			b.WriteString(fmt.Sprintf("%-6d %-50s %15s %15s %15s %9.2f%%%s\n",
				i+1, truncateString(diff.FunctionName, 50),
				diff.BaselineFormatted, diff.TargetFormatted,
				diff.DiffFormatted, diff.DiffPercentage, indicator))
		}
	}

	b.WriteString("\n**符号说明**:\n")
	b.WriteString("- 🔴/⬆ : 性能回归（增加）\n")
	b.WriteString("- 🟢/⬇ : 性能提升（减少）\n")
	b.WriteString("- 🆕 : 新增函数\n")
	b.WriteString("- ❌ : 移除函数\n")

	if format == "markdown" {
		b.WriteString("\n```")
	}

	return b.String()
}

// formatValue 格式化值
func formatValue(value int64) string {
	if value < 1024 {
		return fmt.Sprintf("%d", value)
	}
	return FormatBytes(value)
}

// formatDiffValue 格式化差异值
func formatDiffValue(diff int64) string {
	if diff > 0 {
		return "+" + formatValue(diff)
	}
	return formatValue(diff)
}
