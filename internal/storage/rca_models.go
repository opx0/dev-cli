package storage

import (
	"fmt"
	"hash/fnv"
	"time"
)

// RootCause represents a diagnosed failure with remediation steps.
// It captures the causal chain and suggested fixes for an error pattern.
type RootCause struct {
	ID               string    `json:"id"`
	ErrorSignature   string    `json:"error_signature"` // Hash/pattern of error
	Timestamp        time.Time `json:"timestamp"`
	RootCauseNodes   []string  `json:"root_cause_nodes"`  // Causal chain nodes
	RemediationSteps []string  `json:"remediation_steps"` // Ordered fix steps
	Confidence       float64   `json:"confidence"`        // 0.0-1.0 confidence score
	HistoryItemID    int64     `json:"history_item_id"`   // Link to original failure
}

// Runbook represents a reusable remediation workflow.
// Runbooks are learned from successful fixes and can be reapplied.
type Runbook struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Steps       []RunbookStep `json:"steps"`        // Ordered workflow steps
	SuccessRate float64       `json:"success_rate"` // Historical success percentage
	LastUsed    time.Time     `json:"last_used"`
	UsageCount  int           `json:"usage_count"`
	Tags        []string      `json:"tags"` // For categorization
}

// RunbookStep represents a single step in a runbook.
type RunbookStep struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Rollback    string `json:"rollback,omitempty"`  // Optional rollback command
	Condition   string `json:"condition,omitempty"` // Optional execution condition
}

// ProjectFingerprint identifies project characteristics for targeted RCA.
// Used to match errors to relevant runbooks based on project type.
type ProjectFingerprint struct {
	ID                 string    `json:"id"`
	ProjectType        string    `json:"project_type"`        // "nodejs", "go", "python", etc.
	PackageManager     string    `json:"package_manager"`     // "npm", "go mod", "pip", etc.
	CommonIssues       []string  `json:"common_issues"`       // Frequent error patterns
	AssociatedRunbooks []string  `json:"associated_runbooks"` // Runbook IDs
	DetectedAt         string    `json:"detected_at"`         // Directory path
	DetectedTime       time.Time `json:"detected_time"`
}

// GenerateErrorSignature generates a normalized signature for an error.
// Used for cache lookups and pattern matching.
func GenerateErrorSignature(command string, exitCode int, output string) string {

	firstLine := output
	if idx := indexOf(output, '\n'); idx > 0 {
		firstLine = output[:idx]
	}
	if len(firstLine) > 100 {
		firstLine = firstLine[:100]
	}

	combined := fmt.Sprintf("%s|%d|%s", command, exitCode, firstLine)
	return hashString(combined)
}

// indexOf finds the first occurrence of a rune in a string
func indexOf(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

// hashString creates a hex hash string using FNV-1a algorithm
func hashString(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}
