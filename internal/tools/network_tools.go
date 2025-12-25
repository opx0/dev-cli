package tools

import (
	"context"
	"time"

	"dev-cli/internal/infra"
)

// CheckPortsTool checks port availability and conflicts.
type CheckPortsTool struct{}

func (t *CheckPortsTool) Name() string { return "check_ports" }
func (t *CheckPortsTool) Description() string {
	return "Check port availability and find processes using ports"
}

func (t *CheckPortsTool) Parameters() []ToolParam {
	return []ToolParam{
		{Name: "ports", Type: "[]int", Description: "Ports to check", Required: true},
		{Name: "action", Type: "string", Description: "Action: check, suggest", Required: false, Default: "check"},
	}
}

// PortStatus represents the status of a single port.
type PortStatus struct {
	Port      int    `json:"port"`
	Available bool   `json:"available"`
	Process   string `json:"process,omitempty"`
	PID       int    `json:"pid,omitempty"`
	Suggested int    `json:"suggested,omitempty"`
}

// PortCheckResult contains port check results.
type PortCheckResult struct {
	Ports     []PortStatus `json:"ports"`
	Conflicts int          `json:"conflicts"`
	AllFree   bool         `json:"all_free"`
}

func (t *CheckPortsTool) Execute(ctx context.Context, params map[string]any) ToolResult {
	start := time.Now()

	ports := getIntSlice(params, "ports")
	if len(ports) == 0 {
		return NewErrorResult("ports is required", time.Since(start))
	}

	action := GetString(params, "action", "check")

	results := make([]PortStatus, 0, len(ports))
	conflicts := 0

	for _, port := range ports {
		status := PortStatus{Port: port}

		conflict := infra.CheckPortAvailable(port)
		if conflict == nil {
			status.Available = true
		} else {
			status.Available = false
			status.Process = conflict.Process
			status.PID = conflict.PID
			conflicts++

			if action == "suggest" {
				status.Suggested = infra.FindAvailablePort(port)
			}
		}

		results = append(results, status)
	}

	return NewResult(PortCheckResult{
		Ports:     results,
		Conflicts: conflicts,
		AllFree:   conflicts == 0,
	}, time.Since(start))
}

// getIntSlice extracts an int slice from params.
func getIntSlice(params map[string]any, key string) []int {
	if v, ok := params[key]; ok {
		switch s := v.(type) {
		case []int:
			return s
		case []any:
			result := make([]int, 0, len(s))
			for _, item := range s {
				switch n := item.(type) {
				case int:
					result = append(result, n)
				case int64:
					result = append(result, int(n))
				case float64:
					result = append(result, int(n))
				}
			}
			return result
		}
	}
	return nil
}
