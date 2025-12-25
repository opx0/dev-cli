package workflow

import (
	"os"
	"regexp"
	"strings"
)

// Evaluate checks if a condition is met based on the last step result.
// If no condition is specified, returns true (step should run).
func (c *Condition) Evaluate(lastResult *StepResult) bool {
	if c == nil {
		return true
	}

	switch c.Type {
	case CondExitCode:
		if lastResult == nil {
			return c.Value == "0"
		}
		return matchExitCode(lastResult.ExitCode, c.Value)

	case CondOutputContains:
		if lastResult == nil {
			return false
		}
		return strings.Contains(lastResult.Output, c.Value)

	case CondOutputMatches:
		if lastResult == nil {
			return false
		}
		matched, _ := regexp.MatchString(c.Value, lastResult.Output)
		return matched

	case CondFileExists:
		_, err := os.Stat(c.Value)
		return err == nil

	case CondEnvSet:
		_, exists := os.LookupEnv(c.Value)
		return exists

	default:
		return true
	}
}

// matchExitCode checks if an exit code matches the condition value.
// Supports: "0", "!0" (non-zero), or specific code like "1".
func matchExitCode(exitCode int, value string) bool {
	if value == "!0" {
		return exitCode != 0
	}

	// Parse as integer
	var expected int
	if _, err := parseIntFromString(value, &expected); err != nil {
		return false
	}
	return exitCode == expected
}

// parseIntFromString is a helper to parse int from string.
func parseIntFromString(s string, result *int) (bool, error) {
	n := 0
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false, nil
		}
		n = n*10 + int(ch-'0')
	}
	*result = n
	return true, nil
}

// EvaluateWithStepRef evaluates a condition against a specific step result.
func (c *Condition) EvaluateWithStepRef(results map[string]*StepResult) bool {
	if c == nil {
		return true
	}

	var targetResult *StepResult
	if c.StepRef != "" {
		targetResult = results[c.StepRef]
	} else {

		for _, r := range results {
			if targetResult == nil || r.CompletedAt.After(targetResult.CompletedAt) {
				targetResult = r
			}
		}
	}

	return c.Evaluate(targetResult)
}

// ShouldSkip returns true if the step should be skipped due to condition.
func ShouldSkip(step *Step, results map[string]*StepResult) bool {
	if step.Condition == nil {
		return false
	}
	return !step.Condition.EvaluateWithStepRef(results)
}
