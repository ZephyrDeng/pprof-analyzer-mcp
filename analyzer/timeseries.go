package analyzer

import (
	"encoding/json"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/google/pprof/profile"
)

// TimeSeriesData 表示单个时间点的数据
type TimeSeriesData struct {
	Timestamp   string  `json:"timestamp"`
	Label       string  `json:"label"`
	TotalBytes  int64   `json:"totalBytes"`
	TotalObjects int64  `json:"totalObjects,omitempty"`
}

// TimeSeriesAnalysisResult 表示时序分析的结果
type TimeSeriesAnalysisResult struct {
	ProfileType   string            `json:"profileType"`
	Series        []TimeSeriesData  `json:"series"`
	Trends        []ObjectTrend     `json:"trends"`
	Summary       TimeSeriesSummary  `json:"summary"`
}

// ObjectTrend 表示单个对象类型随时间的变化趋势
type ObjectTrend struct {
	TypeName        string          `json:"typeName"`
	Values          []int64         `json:"values"`
	FormattedValues []string        `json:"formattedValues"`
	GrowthBytes     int64           `json:"growthBytes"`
	GrowthPercent   float64         `json:"growthPercent"`
	GrowthRate      float64         `json:"growthRate"` // 每分钟增长率
	TrendDirection  string          `json:"trendDirection"` // "increasing", "stable", "decreasing"
}

// TimeSeriesSummary 提供时序分析的摘要
type TimeSeriesSummary struct {
	DataPoints      int     `json:"dataPoints"`
	TimeSpanMinutes float64 `json:"timeSpanMinutes"`
	TotalGrowth     int64   `json:"totalGrowth"`
	AvgGrowthRate   float64 `json:"avgGrowthRate"` // MB per minute
	GrowingObjects   int     `json:"growingObjects"`  // 持续增长的对象数量
	StableObjects    int     `json:"stableObjects"`   // 稳定的对象数量
}

// AnalyzeHeapTimeSeries 分析多个 heap profile 的时序数据
func AnalyzeHeapTimeSeries(profiles []*profile.Profile, labels []string, format string) (string, error) {
	log.Printf("Analyzing heap time series: %d data points", len(profiles))

	if len(profiles) < 3 {
		return "", fmt.Errorf("至少需要 3 个 profile 来进行时序分析，当前只有 %d 个", len(profiles))
	}

	if len(labels) != len(profiles) {
		return "", fmt.Errorf("标签数量 (%d) 与 profile 数量 (%d) 不匹配", len(labels), len(profiles))
	}

	// 1. 提取每个时间点的总体数据
	series := extractTimeSeriesData(profiles, labels)

	// 2. 分析对象级别的趋势
	trends, err := analyzeObjectTrends(profiles, labels)
	if err != nil {
		return "", fmt.Errorf("分析对象趋势失败: %w", err)
	}

	// 3. 计算摘要
	summary := computeTimeSeriesSummary(series, trends)

	// 4. 格式化输出
	if format == "json" {
		result := TimeSeriesAnalysisResult{
			ProfileType: "heap",
			Series:      series,
			Trends:      trends,
			Summary:     summary,
		}
		jsonBytes, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return "", fmt.Errorf("failed to marshal JSON: %w", err)
		}
		return string(jsonBytes), nil
	}

	// Text/Markdown 输出
	return formatTimeSeriesReport(series, trends, summary, format), nil
}

// extractTimeSeriesData 提取时序数据
func extractTimeSeriesData(profiles []*profile.Profile, labels []string) []TimeSeriesData {
	series := make([]TimeSeriesData, len(profiles))

	for i, prof := range profiles {
		// 找到 inuse_space 的值索引
		valueIndex := -1
		objectIndex := -1
		for j, st := range prof.SampleType {
			if st.Type == "inuse_space" {
				valueIndex = j
			}
			if st.Type == "inuse_objects" {
				objectIndex = j
			}
		}

		totalBytes := int64(0)
		totalObjects := int64(0)

		for _, sample := range prof.Sample {
			if valueIndex >= 0 && len(sample.Value) > valueIndex {
				totalBytes += sample.Value[valueIndex]
			}
			if objectIndex >= 0 && len(sample.Value) > objectIndex {
				totalObjects += sample.Value[objectIndex]
			}
		}

		// 使用当前时间作为时间戳（实际应用中应该从 profile 元数据读取）
		timestamp := time.Now().Add(time.Duration(i) * time.Minute).Format("2006-01-02 15:04:05")

		series[i] = TimeSeriesData{
			Timestamp:    timestamp,
			Label:        labels[i],
			TotalBytes:   totalBytes,
			TotalObjects: totalObjects,
		}
	}

	return series
}

// analyzeObjectTrends 分析对象级别的趋势
func analyzeObjectTrends(profiles []*profile.Profile, labels []string) ([]ObjectTrend, error) {
	// 聚合每个时间点的对象类型数据
	typeDataMap := make(map[string][]int64) // typeName -> []values

	for i, prof := range profiles {
		// 找到 inuse_space 的值索引
		valueIndex := -1
		for j, st := range prof.SampleType {
			if st.Type == "inuse_space" {
				valueIndex = j
				break
			}
		}

		if valueIndex == -1 {
			continue
		}

		// 按类型聚合
		typeValues := make(map[string]int64)
		for _, sample := range prof.Sample {
			if len(sample.Value) <= valueIndex {
				continue
			}

			value := sample.Value[valueIndex]

			// 获取对象类型（从 location 的 mapping 中获取）
			typeName := getObjectTypeFromSample(sample)

			typeValues[typeName] += value
		}

		// 将当前时间点的数据添加到时序中
		for typeName, value := range typeValues {
			if typeDataMap[typeName] == nil {
				typeDataMap[typeName] = make([]int64, len(profiles))
			}
			typeDataMap[typeName][i] = value
		}
	}

	// 转换为 ObjectTrend 数组
	var trends []ObjectTrend
	for typeName, values := range typeDataMap {
		formattedValues := make([]string, len(values))
		for i, v := range values {
			formattedValues[i] = FormatBytes(v)
		}

		// 计算增长
		firstVal := values[0]
		lastVal := values[len(values)-1]
		growthBytes := lastVal - firstVal
		growthPercent := 0.0
		if firstVal > 0 {
			growthPercent = float64(growthBytes) / float64(firstVal) * 100
		}

		// 计算增长率（每分钟）
		timePoints := len(values)
		growthRate := float64(growthBytes) / float64(timePoints) / 1024 / 1024 // MB per minute

		// 判断趋势方向
		trendDirection := "stable"
		if growthPercent > 10 {
			trendDirection = "increasing"
		} else if growthPercent < -10 {
			trendDirection = "decreasing"
		}

		trends = append(trends, ObjectTrend{
			TypeName:        typeName,
			Values:          values,
			FormattedValues: formattedValues,
			GrowthBytes:     growthBytes,
			GrowthPercent:   growthPercent,
			GrowthRate:      growthRate,
			TrendDirection:  trendDirection,
		})
	}

	// 按增长率排序
	sort.Slice(trends, func(i, j int) bool {
		return trends[i].GrowthPercent > trends[j].GrowthPercent
	})

	return trends, nil
}

// getObjectTypeFromSample 从样本中获取对象类型
func getObjectTypeFromSample(sample *profile.Sample) string {
	// 尝试从 location 的 mapping 中获取对象类型
	for _, loc := range sample.Location {
		for _, line := range loc.Line {
			if line.Function != nil {
				// 使用函数名作为类型标识
				return line.Function.Name
			}
		}
	}
	return "unknown"
}

// computeTimeSeriesSummary 计算时序摘要
func computeTimeSeriesSummary(series []TimeSeriesData, trends []ObjectTrend) TimeSeriesSummary {
	if len(series) < 2 {
		return TimeSeriesSummary{
			DataPoints: len(series),
		}
	}

	// 计算时间跨度（假设每个点间隔 1 分钟，实际应该从时间戳计算）
	timeSpanMinutes := float64(len(series) - 1)

	// 计算总增长
	totalGrowth := series[len(series)-1].TotalBytes - series[0].TotalBytes

	// 计算平均增长率
	avgGrowthRate := float64(totalGrowth) / timeSpanMinutes / 1024 / 1024 // MB per minute

	// 统计趋势方向
	growing := 0
	stable := 0
	for _, trend := range trends {
		if trend.TrendDirection == "increasing" {
			growing++
		} else if trend.TrendDirection == "stable" || trend.TrendDirection == "decreasing" {
			stable++
		}
	}

	return TimeSeriesSummary{
		DataPoints:      len(series),
		TimeSpanMinutes: timeSpanMinutes,
		TotalGrowth:     totalGrowth,
		AvgGrowthRate:   avgGrowthRate,
		GrowingObjects:  growing,
		StableObjects:   stable,
	}
}

// formatTimeSeriesReport 格式化时序分析报告
func formatTimeSeriesReport(series []TimeSeriesData, trends []ObjectTrend, summary TimeSeriesSummary, format string) string {
	var b strings.Builder

	if format == "markdown" {
		b.WriteString("# 内存时序分析报告\n\n")
		b.WriteString("## 概述\n\n")
		b.WriteString(fmt.Sprintf("- **数据点数**: %d\n", summary.DataPoints))
		b.WriteString(fmt.Sprintf("- **时间跨度**: %.0f 分钟\n", summary.TimeSpanMinutes))
		b.WriteString(fmt.Sprintf("- **总内存增长**: %s\n", FormatBytes(summary.TotalGrowth)))
		b.WriteString(fmt.Sprintf("- **平均增长率**: %.2f MB/分钟\n\n", summary.AvgGrowthRate))

		b.WriteString("## 时序数据\n\n")
		b.WriteString("| 时间点 | 标签 | 总内存 | 对象数 |\n")
		b.WriteString("|--------|------|--------|--------|\n")
		for _, data := range series {
			b.WriteString(fmt.Sprintf("| %s | %s | %s | %d |\n",
				data.Timestamp, data.Label, FormatBytes(data.TotalBytes), data.TotalObjects))
		}

		b.WriteString("\n## Top 增长对象类型\n\n")
		b.WriteString("| 对象类型 | 初始值 | 最终值 | 增长 | 增长率 | 趋势 |\n")
		b.WriteString("|----------|--------|--------|------|--------|------|\n")
	} else {
		b.WriteString("内存时序分析报告\n")
		b.WriteString("==================\n\n")
		b.WriteString("概述:\n")
		b.WriteString(fmt.Sprintf("  数据点数: %d\n", summary.DataPoints))
		b.WriteString(fmt.Sprintf("  时间跨度: %.0f 分钟\n", summary.TimeSpanMinutes))
		b.WriteString(fmt.Sprintf("  总内存增长: %s\n", FormatBytes(summary.TotalGrowth)))
		b.WriteString(fmt.Sprintf("  平均增长率: %.2f MB/分钟\n\n", summary.AvgGrowthRate))

		b.WriteString("时序数据:\n")
		for _, data := range series {
			b.WriteString(fmt.Sprintf("  [%s] %s: %s (%d objects)\n",
				data.Timestamp, data.Label, FormatBytes(data.TotalBytes), data.TotalObjects))
		}

		b.WriteString("\nTop 增长对象类型:\n")
		b.WriteString(strings.Repeat("-", 120) + "\n")
		b.WriteString(fmt.Sprintf("%-30s %15s %15s %12s %10s %10s\n",
			"对象类型", "初始值", "最终值", "增长", "增长率", "趋势"))
		b.WriteString(strings.Repeat("-", 120) + "\n")
	}

	// 显示增长最快的对象类型
	maxTrends := 10
	if maxTrends > len(trends) {
		maxTrends = len(trends)
	}

	for i := 0; i < maxTrends; i++ {
		trend := trends[i]
		trendIndicator := "📈"
		if trend.TrendDirection == "decreasing" {
			trendIndicator = "📉"
		} else if trend.TrendDirection == "stable" {
			trendIndicator = "➡️"
		}

		if format == "markdown" {
			b.WriteString(fmt.Sprintf("| %s `%s` | %s | %s | %s | %.1f%% | %s %s |\n",
				trendIndicator,
				truncateString(trend.TypeName, 25),
				trend.FormattedValues[0],
				trend.FormattedValues[len(trend.FormattedValues)-1],
				FormatBytes(trend.GrowthBytes),
				trend.GrowthPercent,
				trend.TrendDirection,
				trendIndicator,
			))
		} else {
			b.WriteString(fmt.Sprintf("%-30s %15s %15s %12s %9.1f%% %10s %s\n",
				truncateString(trend.TypeName, 30),
				trend.FormattedValues[0],
				trend.FormattedValues[len(trend.FormattedValues)-1],
				FormatBytes(trend.GrowthBytes),
				trend.GrowthPercent,
				trend.TrendDirection,
				trendIndicator,
			))
		}
	}

	b.WriteString("\n**建议**:\n")
	b.WriteString("- 关注增长率为正且增长率较高的对象类型\n")
	b.WriteString("- 检查是否有内存泄漏（持续增长的类型）\n")
	b.WriteString("- 优化高频分配的对象类型\n")

	if format == "markdown" {
		b.WriteString("\n```")
	}

	return b.String()
}
