package storage

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// DetectProjectFingerprintID returns a stable project ID for the given working
// directory based on the detected package manager. Falls back to an absolute
// path hash when no known manifest is found.
func DetectProjectFingerprintID(cwd string) (projectID, projectType, packageManager string) {
	abs, err := filepath.Abs(cwd)
	if err != nil {
		abs = cwd
	}

	type detect struct {
		file    string
		pType   string
		pkgMgr  string
	}
	manifests := []detect{
		{"go.mod", "go", "go"},
		{"package.json", "nodejs", "npm"},
		{"pyproject.toml", "python", "pip"},
		{"requirements.txt", "python", "pip"},
		{"Cargo.toml", "rust", "cargo"},
		{"pom.xml", "java", "maven"},
		{"build.gradle", "java", "gradle"},
		{"composer.json", "php", "composer"},
		{"Gemfile", "ruby", "bundler"},
	}

	for _, m := range manifests {
		if _, err := os.Stat(filepath.Join(abs, m.file)); err == nil {
			id := hashString("proj|" + abs + "|" + m.pType)
			return id, m.pType, m.pkgMgr
		}
	}

	return hashString("proj|" + abs + "|generic"), "generic", "unknown"
}

// RecordedRunbookStep is a storage-side representation of a single successful
// operation performed by the agent. The llm package converts its ToolCallLog
// entries into this shape before handing them over.
type RecordedRunbookStep struct {
	ToolName    string
	Parameters  string // JSON-encoded parameters
	Description string
}

// OnRunbookRecorded is an optional callback invoked after a runbook has been
// successfully persisted via RecordRunbookFromAgent. cmd/ wires this to the
// memory package for MemPalace write-back without creating a storage→memory
// import cycle. Best-effort: runs on the goroutine that called the parent;
// implementations should spawn their own goroutine if they need to be async.
var OnRunbookRecorded func(cwd, issue, runbookID string, steps []RecordedRunbookStep)

// RecordRunbookFromAgent persists a learned runbook from a successful agent run.
// It:
//   - Derives a project fingerprint from cwd,
//   - Writes a Runbook with the supplied steps,
//   - Writes a RootCause with the issue's error signature pointing to the runbook
//     so future runs of cmd/fix can replay it.
// Returns the generated runbook ID (empty on error).
func RecordRunbookFromAgent(db *sql.DB, cwd, issue string, steps []RecordedRunbookStep) (string, error) {
	if db == nil || len(steps) == 0 {
		return "", nil
	}

	projectID, projectType, _ := DetectProjectFingerprintID(cwd)

	name := issue
	if len(name) > 80 {
		name = name[:77] + "…"
	}

	rbSteps := make([]RunbookStep, 0, len(steps))
	remediationStrs := make([]string, 0, len(steps))
	for _, s := range steps {
		rbSteps = append(rbSteps, RunbookStep{
			ID:          uuid.NewString(),
			Name:        s.ToolName,
			Command:     s.Parameters,
			Description: s.Description,
		})
		remediationStrs = append(remediationStrs, s.ToolName+": "+s.Parameters)
	}

	rb := Runbook{
		ID:          uuid.NewString(),
		ProjectID:   projectID,
		Name:        name,
		Description: issue,
		Steps:       rbSteps,
		SuccessRate: 1.0,
		LastUsed:    time.Now(),
		UsageCount:  1,
		Tags:        []string{projectType},
	}
	if err := SaveRunbook(db, rb); err != nil {
		return "", err
	}

	sig := GenerateErrorSignature(strings.TrimSpace(issue), 1, "")
	rc := RootCause{
		ID:               uuid.NewString(),
		ErrorSignature:   sig,
		Timestamp:        time.Now(),
		RootCauseNodes:   []string{"agent-resolved"},
		RemediationSteps: remediationStrs,
		Confidence:       0.75,
	}
	if err := SaveRootCause(db, rc); err != nil {
		return rb.ID, err
	}

	if OnRunbookRecorded != nil {
		OnRunbookRecorded(cwd, issue, rb.ID, steps)
	}
	return rb.ID, nil
}
