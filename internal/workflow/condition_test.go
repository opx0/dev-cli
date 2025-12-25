package workflow

import (
	"testing"
	"time"
)

func TestConditionEvaluate(t *testing.T) {
	tests := []struct {
		name     string
		cond     *Condition
		result   *StepResult
		expected bool
	}{
		{
			name:     "nil condition returns true",
			cond:     nil,
			result:   nil,
			expected: true,
		},
		{
			name: "exit_code 0 matches success",
			cond: &Condition{
				Type:  CondExitCode,
				Value: "0",
			},
			result: &StepResult{
				ExitCode: 0,
			},
			expected: true,
		},
		{
			name: "exit_code 0 does not match failure",
			cond: &Condition{
				Type:  CondExitCode,
				Value: "0",
			},
			result: &StepResult{
				ExitCode: 1,
			},
			expected: false,
		},
		{
			name: "exit_code !0 matches failure",
			cond: &Condition{
				Type:  CondExitCode,
				Value: "!0",
			},
			result: &StepResult{
				ExitCode: 1,
			},
			expected: true,
		},
		{
			name: "output_contains matches",
			cond: &Condition{
				Type:  CondOutputContains,
				Value: "success",
			},
			result: &StepResult{
				Output: "build success completed",
			},
			expected: true,
		},
		{
			name: "output_contains does not match",
			cond: &Condition{
				Type:  CondOutputContains,
				Value: "error",
			},
			result: &StepResult{
				Output: "build success completed",
			},
			expected: false,
		},
		{
			name: "output_matches regex",
			cond: &Condition{
				Type:  CondOutputMatches,
				Value: `version \d+\.\d+`,
			},
			result: &StepResult{
				Output: "version 1.5 installed",
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.cond.Evaluate(tt.result)
			if got != tt.expected {
				t.Errorf("Condition.Evaluate() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestShouldSkip(t *testing.T) {
	results := map[string]*StepResult{
		"step1": {
			StepID:      "step1",
			ExitCode:    0,
			CompletedAt: time.Now(),
		},
	}

	tests := []struct {
		name     string
		step     *Step
		results  map[string]*StepResult
		expected bool
	}{
		{
			name: "no condition - should not skip",
			step: &Step{
				ID:      "step2",
				Command: "echo test",
			},
			results:  results,
			expected: false,
		},
		{
			name: "condition met - should not skip",
			step: &Step{
				ID:      "step2",
				Command: "echo test",
				Condition: &Condition{
					Type:  CondExitCode,
					Value: "0",
				},
			},
			results:  results,
			expected: false,
		},
		{
			name: "condition not met - should skip",
			step: &Step{
				ID:      "step2",
				Command: "echo test",
				Condition: &Condition{
					Type:  CondExitCode,
					Value: "!0",
				},
			},
			results:  results,
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShouldSkip(tt.step, tt.results)
			if got != tt.expected {
				t.Errorf("ShouldSkip() = %v, want %v", got, tt.expected)
			}
		})
	}
}
