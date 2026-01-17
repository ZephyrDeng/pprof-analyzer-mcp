package analyzer

import (
	"testing"

	"github.com/google/pprof/profile"
)

// TestAnalyzeBlockProfile 测试 Block profile 分析功能
func TestAnalyzeBlockProfile(t *testing.T) {
	// 创建一个测试用的 Block profile
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{200, 100000000}, // 200 次阻塞，100ms 延迟
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "main.channelReceive",
								},
							},
						},
					},
				},
			},
			{
				Value: []int64{80, 40000000}, // 80 次阻塞，40ms 延迟
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "main.networkCall",
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
				"Block Profile 分析结果",
				"main.channelReceive",
				"main.networkCall",
				"280", // 总阻塞次数 (200 + 80)
				"140", // 总延迟约 140ms（可能有不同格式）
			},
		},
		{
			name:   "Markdown format",
			topN:   5,
			format: "markdown",
			wantContains: []string{
				"# Block Profile 分析报告",
				"main.channelReceive",
				"main.networkCall",
				"阻塞次数",
			},
		},
		{
			name:   "JSON format",
			topN:   5,
			format: "json",
			wantContains: []string{
				`"profileType": "block"`,
				`"totalContentions": 280`,
				`"functionName": "main.channelReceive"`,
				`"delayNanos": 100000000`,
			},
		},
		{
			name:   "Top N limiting",
			topN:   1,
			format: "text",
			wantContains: []string{
				"main.channelReceive", // 应该包含第一个（延迟最高的）
			},
			notWantContains: []string{
				"main.networkCall", // 不应该包含第二个（因为只取 Top 1）
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeBlockProfile(p, tt.topN, tt.format)
			if err != nil {
				t.Errorf("AnalyzeBlockProfile() error = %v", err)
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

// TestAnalyzeBlockProfileEmpty 测试空 Block profile
func TestAnalyzeBlockProfileEmpty(t *testing.T) {
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{}, // 空样本
	}

	result, err := AnalyzeBlockProfile(p, 5, "text")
	if err != nil {
		t.Errorf("AnalyzeBlockProfile() error = %v", err)
		return
	}

	// 应该返回友好的消息
	if !containsString(result, "未发现阻塞操作") {
		t.Errorf("Empty profile should return friendly message, got: %s", result)
	}
}

// TestAnalyzeBlockProfileInvalidSampleTypes 测试缺少样本类型的情况
func TestAnalyzeBlockProfileInvalidSampleTypes(t *testing.T) {
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

	_, err := AnalyzeBlockProfile(p, 5, "text")
	if err == nil {
		t.Error("Expected error for invalid sample types, got nil")
	}

	if !containsString(err.Error(), "contentions") && !containsString(err.Error(), "delay") {
		t.Errorf("Error should mention missing sample types, got: %v", err)
	}
}

// TestAnalyzeBlockProfileChannelBlocking 测试通道阻塞场景
func TestAnalyzeBlockProfileChannelBlocking(t *testing.T) {
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1000, 50000000}, // 1000 次短时间阻塞
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "runtime.chanrecv1",
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := AnalyzeBlockProfile(p, 5, "text")
	if err != nil {
		t.Errorf("AnalyzeBlockProfile() error = %v", err)
		return
	}

	// 应该包含通道相关的分析
	if !containsString(result, "runtime.chanrecv1") {
		t.Errorf("Result should contain channel blocking function, got: %s", result)
	}

	// 应该包含分析建议
	if !containsString(result, "建议") {
		t.Errorf("Result should contain analysis suggestions, got: %s", result)
	}
}

// TestAnalyzeBlockProfileAverageDelay 测试平均延迟计算
func TestAnalyzeBlockProfileAverageDelay(t *testing.T) {
	p := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "contentions", Unit: "count"},
			{Type: "delay", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{10, 100000000}, // 10 次阻塞，100ms 总延迟，平均 10ms
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{
									Name: "main.slowOperation",
								},
							},
						},
					},
				},
			},
		},
	}

	result, err := AnalyzeBlockProfile(p, 5, "json")
	if err != nil {
		t.Errorf("AnalyzeBlockProfile() error = %v", err)
		return
	}

	// JSON 输出应该包含平均延迟
	if !containsString(result, `"avgDelayNanos": 10000000`) { // 100ms / 10 = 10ms
		t.Errorf("Result should contain average delay, got: %s", result)
	}
}
