package analyzer

import (
	"testing"

	"github.com/google/pprof/profile"
)

// TestAnalyzeMutexProfile 测试 Mutex profile 分析功能
func TestAnalyzeMutexProfile(t *testing.T) {
	// 创建一个测试用的 Mutex profile
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{100, 50000000}, // 100 次竞争，50ms 延迟
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "main.lockContention",
								},
							},
						},
					},
				},
			},
			{
				Value: []int64{50, 25000000}, // 50 次竞争，25ms 延迟
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "main.anotherLock",
								},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name          string
		topN          int
		format        string
		wantContains  []string
		notWantContains []string
	}{
		{
			name:   "Text format - basic output",
			topN:   5,
			format: "text",
			wantContains: []string{
				"Mutex Profile 分析结果",
				"main.lockContention",
				"main.anotherLock",
				"150", // 总竞争次数
				"75", // 总延迟约 75ms（可能有不同格式）
			},
		},
		{
			name:   "Markdown format",
			topN:   5,
			format: "markdown",
			wantContains: []string{
				"# Mutex Profile 分析报告",
				"main.lockContention",
				"main.anotherLock",
			},
		},
		{
			name:   "JSON format",
			topN:   5,
			format: "json",
			wantContains: []string{
				`"profileType": "mutex"`,
				`"totalContentions": 150`,
				`"functionName": "main.lockContention"`,
				`"delayNanos": 50000000`,
			},
		},
		{
			name:   "Top N limiting",
			topN:   1,
			format: "text",
			wantContains: []string{
				"main.lockContention", // 应该包含第一个
			},
			notWantContains: []string{
				"main.anotherLock", // 不应该包含第二个（因为只取 Top 1）
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeMutexProfile(p, tt.topN, tt.format)
			if err != nil {
				t.Errorf("AnalyzeMutexProfile() error = %v", err)
				return
			}

			// 检查包含的字符串
			for _, want := range tt.wantContains {
				if !containsString(result, want) {
					t.Errorf("Result does not contain expected string %q\nGot:\n%s", want, result)
				}
			}

			// 检查不包含的字符串
			for _, notWant := range tt.notWantContains {
				if containsString(result, notWant) {
					t.Errorf("Result should not contain string %q\nGot:\n%s", notWant, result)
				}
			}
		})
	}
}

// TestAnalyzeMutexProfileEmpty 测试空 Mutex profile
func TestAnalyzeMutexProfileEmpty(t *testing.T) {
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{}, // 空样本
	}

	result, err := AnalyzeMutexProfile(p, 5, "text")
	if err != nil {
		t.Errorf("AnalyzeMutexProfile() error = %v", err)
		return
	}

	// 应该返回友好的消息
	if !containsString(result, "未发现锁竞争") {
		t.Errorf("Empty profile should return friendly message, got: %s", result)
	}
}

// TestAnalyzeMutexProfileInvalidSampleTypes 测试缺少样本类型的情况
func TestAnalyzeMutexProfileInvalidSampleTypes(t *testing.T) {
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"}, // 错误的类型
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{100},
			},
		},
	}

	_, err := AnalyzeMutexProfile(p, 5, "text")
	if err == nil {
		t.Error("Expected error for invalid sample types, got nil")
	}

	if !containsString(err.Error(), "contentions") && !containsString(err.Error(), "delay") {
		t.Errorf("Error should mention missing sample types, got: %v", err)
	}
}

// containsString 检查字符串是否包含子字符串
func containsString(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
