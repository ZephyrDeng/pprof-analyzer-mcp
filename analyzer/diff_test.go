package analyzer

import (
	"testing"

	"github.com/google/pprof/profile"
)

// TestCompareProfiles 测试 profile 比较功能
func TestCompareProfiles(t *testing.T) {
	// 创建基线 profile
	baseline := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{100000000}, // 100ms
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.slowFunction"},
							},
						},
					},
				},
			},
			{
				Value: []int64{50000000}, // 50ms
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.fastFunction"},
							},
						},
					},
				},
			},
		},
	}

	// 创建目标 profile（改进版本）
	target := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{80000000}, // 80ms - 改进！
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.slowFunction"},
							},
						},
					},
				},
			},
			{
				Value: []int64{30000000}, // 30ms - 改进！
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.fastFunction"},
							},
						},
					},
				},
			},
			{
				Value: []int64{20000000}, // 20ms - 新函数
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.newFunction"},
							},
						},
					},
				},
			},
		},
	}

	tests := []struct {
		name         string
		topN         int
		format       string
		wantContains []string
	}{
		{
			name:   "Text format - basic comparison",
			topN:   5,
			format: "text",
			wantContains: []string{
				"Profile 差异分析报告",
				"main.slowFunction",
				"main.fastFunction",
				"main.newFunction",
				"性能提升",
				"新增函数",
			},
		},
		{
			name:   "Markdown format",
			topN:   5,
			format: "markdown",
			wantContains: []string{
				"# Profile 差异分析报告",
				"main.slowFunction",
				"总体摘要",
			},
		},
		{
			name:   "JSON format",
			topN:   5,
			format: "json",
			wantContains: []string{
				`"profileType": "cpu"`,
				`"functionName": "main.slowFunction"`,
				`"baselineValue": 100000000`,
				`"targetValue": 80000000`,
				`"diffValue": -20000000`,
				`"improvedFuncs": 2`,
				`"addedFuncs": 1`,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := CompareProfiles(baseline, target, "cpu", tt.topN, tt.format)
			if err != nil {
				t.Errorf("CompareProfiles() error = %v", err)
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

// TestCompareProfilesRegressions 测试性能回归场景
func TestCompareProfilesRegressions(t *testing.T) {
	// 基线：性能良好
	baseline := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{50000000},
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.processData"},
							},
						},
					},
				},
			},
		},
	}

	// 目标：性能回归（变慢）
	target := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "cpu", Unit: "nanoseconds"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{150000000}, // 150ms - 变慢了 3 倍！
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.processData"},
							},
						},
					},
				},
			},
		},
	}

	result, err := CompareProfiles(baseline, target, "cpu", 10, "text")
	if err != nil {
		t.Fatalf("CompareProfiles() error = %v", err)
	}

	// 应该检测到性能回归
	if !containsString(result, "性能回归") {
		t.Errorf("Expected to detect performance regression, got:\n%s", result)
	}

	// 应该显示增加的时间
	if !containsString(result, "+") {
		t.Errorf("Expected to show increase with + sign, got:\n%s", result)
	}
}

// TestCompareProfilesRemovedFunctions 测试移除函数的场景
func TestCompareProfilesRemovedFunctions(t *testing.T) {
	baseline := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{
			{
				Value: []int64{1024},
				Location: []*profile.Location{
					{
						Line: []profile.Line{
							{
								Function: &profile.Function{Name: "main.oldFunction"},
							},
						},
					},
				},
			},
		},
	}

	target := &profile.Profile{
		SampleType: []*profile.ValueType{
			{Type: "inuse_space", Unit: "bytes"},
		},
		Sample: []*profile.Sample{}, // 所有函数都被移除
	}

	result, err := CompareProfiles(baseline, target, "heap", 10, "json")
	if err != nil {
		t.Fatalf("CompareProfiles() error = %v", err)
	}

	// 应该检测到移除的函数
	if !containsString(result, `"removedFuncs": 1`) {
		t.Errorf("Expected to detect removed functions, got:\n%s", result)
	}

	if !containsString(result, "main.oldFunction") {
		t.Errorf("Expected to contain old function name, got:\n%s", result)
	}
}
