package analyzer

import (
	"testing"

	"github.com/google/pprof/profile"
)

// TestAnalyzeHeapTimeSeries 测试内存时序分析功能
func TestAnalyzeHeapTimeSeries(t *testing.T) {
	// 创建 3 个时间点的 profile
	profiles := make([]*profile.Profile, 3)
	labels := []string{"T1", "T2", "T3"}

	// T1: 初始状态
	profiles[0] = &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
			{Type: "inuse_objects", Unit: "count"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024 * 1024 * 10, 100}, // 10MB, 100 objects
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{Function: &profile.Function{Name: "main.growingCache"}},
						},
					},
				},
			},
		},
	}

	// T2: 中间状态（增长）
	profiles[1] = &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
			{Type: "inuse_objects", Unit: "count"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024 * 1024 * 20, 200}, // 20MB, 200 objects
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{Function: &profile.Function{Name: "main.growingCache"}},
						},
					},
				},
			},
		},
	}

	// T3: 最终状态（继续增长）
	profiles[2] = &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
			{Type: "inuse_objects", Unit: "count"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024 * 1024 * 50, 500}, // 50MB, 500 objects
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{Function: &profile.Function{Name: "main.growingCache"}},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name         string
		format       string
		wantContains []string
	}{
		{
			name:   "Text format",
			format: "text",
			wantContains: []string{
				"内存时序分析报告",
				"数据点数: 3",
				"main.growingCache",
				"总内存增长",
				"10.00 MB",
				"建议",
			},
		},
		{
			name:   "Markdown format",
			format: "markdown",
			wantContains: []string{
				"# 内存时序分析报告",
				"## 概述",
				"## 时序数据",
				"## Top 增长对象类型",
				"T1",
				"T2",
				"T3",
			},
		},
		{
			name:   "JSON format",
			format: "json",
			wantContains: []string{
				`"profileType": "heap"`,
				`"dataPoints": 3`,
				`"totalGrowth":`,
				`"typeName": "main.growingCache"`,
				`"trendDirection": "increasing"`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := AnalyzeHeapTimeSeries(profiles, labels, tt.format)
			if err != nil {
				t.Errorf("AnalyzeHeapTimeSeries() error = %v", err)
				return
			}

			for _, want := range tt.wantContains {
				if !containsString(result, want) {
					t.Errorf("Result does not contain expected string %q\nGot:\n%s", want, result)
				}
			}
		})
	}
}

// TestAnalyzeHeapTimeSeriesInsufficientData 测试数据点不足的情况
func TestAnalyzeHeapTimeSeriesInsufficientData(t *testing.T) {
	profiles := make([]*profile.Profile, 2) // 只有 2 个数据点
	labels := []string{"T1", "T2"}

	profiles[0] = &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024 * 1024},
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{Function: &profile.Function{Name: "main.test"}},
						},
					},
				},
			},
		},
	}

	profiles[1] = &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024 * 1024 * 2},
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{Function: &profile.Function{Name: "main.test"}},
						},
					},
				},
			},
		},
	}

	_, err := AnalyzeHeapTimeSeries(profiles, labels, "text")
	if err == nil {
		t.Error("Expected error for insufficient data points, got nil")
	}

	if !containsString(err.Error(), "至少需要 3 个") {
		t.Errorf("Error should mention minimum 3 profiles required, got: %v", err)
	}
}

// TestAnalyzeHeapTimeSeriesLabelMismatch 测试标签数量不匹配的情况
func TestAnalyzeHeapTimeSeriesLabelMismatch(t *testing.T) {
	profiles := make([]*profile.Profile, 3)
	labels := []string{"T1", "T2"} // 只有 2 个标签，但有 3 个 profile

	for i := range profiles {
		profiles[i] = &profile.Profile{
			SampleType: []*profile.ValueType{
				{Type: "inuse_space", Unit: "bytes"},
			},
			Sample: []*profile.Sample{
				{
					Value: []int64{1024 * 1024},
				},
			},
		}
	}

	_, err := AnalyzeHeapTimeSeries(profiles, labels, "text")
	if err == nil {
		t.Error("Expected error for label count mismatch, got nil")
	}

	if !containsString(err.Error(), "不匹配") {
		t.Errorf("Error should mention label count mismatch, got: %v", err)
	}
}

// TestAnalyzeHeapTimeSeriesMultipleTypes 测试多种对象类型的时序分析
func TestAnalyzeHeapTimeSeriesMultipleTypes(t *testing.T) {
	profiles := make([]*profile.Profile, 3)
	labels := []string{"Initial", "After 5min", "After 10min"}

	// 创建包含多种对象类型的 profile
	for i := range profiles {
		baseValue := int64(1024 * 1024) * int64(i+1) * 10
		profiles[i] = &profile.Profile{
			SampleType: []*profile.ValueType{
				{Type: "inuse_space", Unit: "bytes"},
				{Type: "inuse_objects", Unit: "count"},
			},
			Sample: []*profile.Sample{
				{
					Value: []int64{baseValue, int64(i + 1) * 100},
					Location: []*profile.Location{
						{
							Line: []profile.Line{
								{Function: &profile.Function{Name: "main.cache"}},
							},
						},
					},
				},
				{
					Value: []int64{baseValue / 2, int64(i + 1) * 50},
					Location: []*profile.Location{
						{
							Line: []profile.Line{
								{Function: &profile.Function{Name: "main.buffer"}},
							},
						},
					},
				},
			},
		}
	}

	result, err := AnalyzeHeapTimeSeries(profiles, labels, "text")
	if err != nil {
		t.Fatalf("AnalyzeHeapTimeSeries() error = %v", err)
	}

	// 应该包含两种对象类型
	if !containsString(result, "main.cache") {
		t.Error("Result should contain main.cache type")
	}
	if !containsString(result, "main.buffer") {
		t.Error("Result should contain main.buffer type")
	}

	// 应该显示增长趋势
	if !containsString(result, "增长率") {
		t.Error("Result should show growth rate")
	}
}
