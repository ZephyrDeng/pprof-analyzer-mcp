package analyzer

import (
	"fmt"
	"time"
)

// FormatSampleValue 将样本值 (如 CPU 时间或计数) 转换为人类可读的字符串。
// 注意：已导出 (首字母大写)。
func FormatSampleValue(value int64, unit string) string {
	switch unit {
	case "nanoseconds":
		d := time.Duration(value) * time.Nanosecond
		if d >= time.Second {
			return fmt.Sprintf("%.2fs", d.Seconds())
		}
		if d >= time.Millisecond {
			return fmt.Sprintf("%.2fms", float64(d.Milliseconds()))
		}
		if d >= time.Microsecond {
			return fmt.Sprintf("%.2fus", float64(d.Microseconds()))
		}
		return fmt.Sprintf("%dns", d.Nanoseconds())
	case "count":
		return fmt.Sprintf("%d", value)
	// 如果需要，可以添加其他潜在单位的处理
	default:
		return fmt.Sprintf("%d %s", value, unit) // 回退方案
	}
}

// FormatBytes 将字节数转换为人类可读的字符串 (KB, MB, GB)。
// 注意：已导出 (首字母大写)。
func FormatBytes(b int64) string {
	const unit = 1024
	if b < unit {
		return fmt.Sprintf("%d B", b)
	}
	div, exp := int64(unit), 0
	for n := b / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(b)/float64(div), "KMGTPE"[exp]) // Kilo, Mega, Giga, Tera, Peta, Exa
}

// formatNumber 格式化数字为可读字符串（带千分位）
func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	if n < 1000000 {
		return fmt.Sprintf("%.1fK", float64(n)/1000)
	}
	if n < 1000000000 {
		return fmt.Sprintf("%.1fM", float64(n)/1000000)
	}
	return fmt.Sprintf("%.1fG", float64(n)/1000000000)
}

// truncateString 截断字符串到指定长度
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
