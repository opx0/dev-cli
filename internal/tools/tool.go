// Package tools provides a unified tool abstraction for the RCA agent.
// Tools enable structured execution of diagnostic operations like file inspection,
// command execution, Docker queries, and Git analysis.
package tools

import (
	"context"
	"time"
)

// Tool defines the interface for all agent tools.
type Tool interface {
	// Name returns the unique identifier for this tool.
	Name() string

	// Description returns a human-readable description of what this tool does.
	Description() string

	// Parameters returns the parameter definitions for this tool.
	Parameters() []ToolParam

	// Execute runs the tool with the given parameters.
	Execute(ctx context.Context, params map[string]any) ToolResult
}

// ToolParam defines a parameter for a tool.
type ToolParam struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // string, int, bool, []string, []int
	Description string `json:"description"`
	Required    bool   `json:"required"`
	Default     any    `json:"default,omitempty"`
}

// ToolResult represents the outcome of a tool execution.
type ToolResult struct {
	Success  bool          `json:"success"`
	Data     any           `json:"data,omitempty"`
	Error    string        `json:"error,omitempty"`
	Duration time.Duration `json:"duration"`
}

// ToolInfo provides metadata about a registered tool.
type ToolInfo struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  []ToolParam `json:"parameters"`
}

// NewResult creates a successful result with data.
func NewResult(data any, duration time.Duration) ToolResult {
	return ToolResult{
		Success:  true,
		Data:     data,
		Duration: duration,
	}
}

// NewErrorResult creates a failed result with an error message.
func NewErrorResult(err string, duration time.Duration) ToolResult {
	return ToolResult{
		Success:  false,
		Error:    err,
		Duration: duration,
	}
}

// GetString extracts a string parameter with a default value.
func GetString(params map[string]any, key string, defaultVal string) string {
	if v, ok := params[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return defaultVal
}

// GetInt extracts an int parameter with a default value.
func GetInt(params map[string]any, key string, defaultVal int) int {
	if v, ok := params[key]; ok {
		switch n := v.(type) {
		case int:
			return n
		case int64:
			return int(n)
		case float64:
			return int(n)
		}
	}
	return defaultVal
}

// GetBool extracts a bool parameter with a default value.
func GetBool(params map[string]any, key string, defaultVal bool) bool {
	if v, ok := params[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return defaultVal
}

// GetStringSlice extracts a string slice parameter.
func GetStringSlice(params map[string]any, key string) []string {
	if v, ok := params[key]; ok {
		switch s := v.(type) {
		case []string:
			return s
		case []any:
			result := make([]string, 0, len(s))
			for _, item := range s {
				if str, ok := item.(string); ok {
					result = append(result, str)
				}
			}
			return result
		}
	}
	return nil
}

// GetDuration extracts a duration parameter (accepts string like "30s" or seconds as int).
func GetDuration(params map[string]any, key string, defaultVal time.Duration) time.Duration {
	if v, ok := params[key]; ok {
		switch d := v.(type) {
		case time.Duration:
			return d
		case string:
			if parsed, err := time.ParseDuration(d); err == nil {
				return parsed
			}
		case int:
			return time.Duration(d) * time.Second
		case int64:
			return time.Duration(d) * time.Second
		case float64:
			return time.Duration(d) * time.Second
		}
	}
	return defaultVal
}
