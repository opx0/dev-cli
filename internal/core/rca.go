package core

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"time"
)

type RootCause struct {
	ID               string    `json:"id"`
	ErrorSignature   string    `json:"error_signature"`
	Timestamp        time.Time `json:"timestamp"`
	RootCauseNodes   []string  `json:"root_cause_nodes"`
	RemediationSteps []string  `json:"remediation_steps"`
	Confidence       float64   `json:"confidence"`
	HistoryItemID    int64     `json:"history_item_id"`
}

type Runbook struct {
	ID          string        `json:"id"`
	ProjectID   string        `json:"project_id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Steps       []RunbookStep `json:"steps"`
	SuccessRate float64       `json:"success_rate"`
	LastUsed    time.Time     `json:"last_used"`
	UsageCount  int           `json:"usage_count"`
	Tags        []string      `json:"tags"`
}

type RunbookStep struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Command     string `json:"command"`
	Description string `json:"description"`
	Rollback    string `json:"rollback,omitempty"`
	Condition   string `json:"condition,omitempty"`
}

type ProjectFingerprint struct {
	ID                 string    `json:"id"`
	ProjectType        string    `json:"project_type"`
	PackageManager     string    `json:"package_manager"`
	CommonIssues       []string  `json:"common_issues"`
	AssociatedRunbooks []string  `json:"associated_runbooks"`
	DetectedAt         string    `json:"detected_at"`
	DetectedTime       time.Time `json:"detected_time"`
}

func GenerateErrorSignature(command string, exitCode int, output string) string {
	firstLine := output
	if idx := indexOfRune(output, '\n'); idx > 0 {
		firstLine = output[:idx]
	}
	if len(firstLine) > 100 {
		firstLine = firstLine[:100]
	}

	combined := fmt.Sprintf("%s|%d|%s", command, exitCode, firstLine)
	return hashString(combined)
}

func indexOfRune(s string, r rune) int {
	for i, c := range s {
		if c == r {
			return i
		}
	}
	return -1
}

func hashString(s string) string {
	h := fnv.New64a()
	h.Write([]byte(s))
	return fmt.Sprintf("%016x", h.Sum64())
}

func SaveRootCause(db *sql.DB, rc RootCause) error {
	nodesJSON, err := json.Marshal(rc.RootCauseNodes)
	if err != nil {
		return fmt.Errorf("marshal root_cause_nodes: %w", err)
	}
	stepsJSON, err := json.Marshal(rc.RemediationSteps)
	if err != nil {
		return fmt.Errorf("marshal remediation_steps: %w", err)
	}

	query := `INSERT OR REPLACE INTO root_causes 
		(id, error_signature, timestamp, root_cause_nodes, remediation_steps, confidence, history_item_id)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query,
		rc.ID,
		rc.ErrorSignature,
		rc.Timestamp.Unix(),
		string(nodesJSON),
		string(stepsJSON),
		rc.Confidence,
		rc.HistoryItemID,
	)
	return err
}

func GetRootCauseBySignature(db *sql.DB, signature string) (*RootCause, error) {
	query := `SELECT id, error_signature, timestamp, root_cause_nodes, remediation_steps, confidence, history_item_id
		FROM root_causes WHERE error_signature = ? ORDER BY timestamp DESC LIMIT 1`

	row := db.QueryRow(query, signature)
	return scanRootCause(row)
}

func GetRootCauseByID(db *sql.DB, id string) (*RootCause, error) {
	query := `SELECT id, error_signature, timestamp, root_cause_nodes, remediation_steps, confidence, history_item_id
		FROM root_causes WHERE id = ?`

	row := db.QueryRow(query, id)
	return scanRootCause(row)
}

func GetRecentRootCauses(db *sql.DB, limit int) ([]RootCause, error) {
	query := `SELECT id, error_signature, timestamp, root_cause_nodes, remediation_steps, confidence, history_item_id
		FROM root_causes ORDER BY timestamp DESC LIMIT ?`

	rows, err := db.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []RootCause
	for rows.Next() {
		rc, err := scanRootCauseRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *rc)
	}
	return results, nil
}

func scanRootCause(row *sql.Row) (*RootCause, error) {
	var rc RootCause
	var ts int64
	var nodesJSON, stepsJSON string

	err := row.Scan(&rc.ID, &rc.ErrorSignature, &ts, &nodesJSON, &stepsJSON, &rc.Confidence, &rc.HistoryItemID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rc.Timestamp = time.Unix(ts, 0)
	if err := json.Unmarshal([]byte(nodesJSON), &rc.RootCauseNodes); err != nil {
		rc.RootCauseNodes = []string{}
	}
	if err := json.Unmarshal([]byte(stepsJSON), &rc.RemediationSteps); err != nil {
		rc.RemediationSteps = []string{}
	}

	return &rc, nil
}

func scanRootCauseRow(rows *sql.Rows) (*RootCause, error) {
	var rc RootCause
	var ts int64
	var nodesJSON, stepsJSON string

	err := rows.Scan(&rc.ID, &rc.ErrorSignature, &ts, &nodesJSON, &stepsJSON, &rc.Confidence, &rc.HistoryItemID)
	if err != nil {
		return nil, err
	}

	rc.Timestamp = time.Unix(ts, 0)
	if err := json.Unmarshal([]byte(nodesJSON), &rc.RootCauseNodes); err != nil {
		rc.RootCauseNodes = []string{}
	}
	if err := json.Unmarshal([]byte(stepsJSON), &rc.RemediationSteps); err != nil {
		rc.RemediationSteps = []string{}
	}

	return &rc, nil
}

func SaveRunbook(db *sql.DB, rb Runbook) error {
	stepsJSON, err := json.Marshal(rb.Steps)
	if err != nil {
		return fmt.Errorf("marshal steps: %w", err)
	}
	tagsJSON, err := json.Marshal(rb.Tags)
	if err != nil {
		return fmt.Errorf("marshal tags: %w", err)
	}

	query := `INSERT OR REPLACE INTO runbooks
		(id, project_id, name, description, steps, success_rate, last_used, usage_count, tags)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query,
		rb.ID,
		rb.ProjectID,
		rb.Name,
		rb.Description,
		string(stepsJSON),
		rb.SuccessRate,
		rb.LastUsed.Unix(),
		rb.UsageCount,
		string(tagsJSON),
	)
	return err
}

func GetRunbookByID(db *sql.DB, id string) (*Runbook, error) {
	query := `SELECT id, project_id, name, description, steps, success_rate, last_used, usage_count, tags
		FROM runbooks WHERE id = ?`

	row := db.QueryRow(query, id)
	return scanRunbook(row)
}

func GetRunbooksForProject(db *sql.DB, projectID string) ([]Runbook, error) {
	query := `SELECT id, project_id, name, description, steps, success_rate, last_used, usage_count, tags
		FROM runbooks WHERE project_id = ? ORDER BY success_rate DESC`

	rows, err := db.Query(query, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []Runbook
	for rows.Next() {
		rb, err := scanRunbookRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *rb)
	}
	return results, nil
}

func UpdateRunbookStats(db *sql.DB, id string, success bool) error {
	rb, err := GetRunbookByID(db, id)
	if err != nil {
		return err
	}
	if rb == nil {
		return fmt.Errorf("runbook not found: %s", id)
	}

	rb.UsageCount++
	if success {
		rb.SuccessRate = ((rb.SuccessRate * float64(rb.UsageCount-1)) + 1.0) / float64(rb.UsageCount)
	} else {
		rb.SuccessRate = (rb.SuccessRate * float64(rb.UsageCount-1)) / float64(rb.UsageCount)
	}
	rb.LastUsed = time.Now()

	query := `UPDATE runbooks SET success_rate = ?, last_used = ?, usage_count = ? WHERE id = ?`
	_, err = db.Exec(query, rb.SuccessRate, rb.LastUsed.Unix(), rb.UsageCount, id)
	return err
}

func scanRunbook(row *sql.Row) (*Runbook, error) {
	var rb Runbook
	var lastUsed int64
	var stepsJSON, tagsJSON string

	err := row.Scan(&rb.ID, &rb.ProjectID, &rb.Name, &rb.Description, &stepsJSON, &rb.SuccessRate, &lastUsed, &rb.UsageCount, &tagsJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rb.LastUsed = time.Unix(lastUsed, 0)
	if err := json.Unmarshal([]byte(stepsJSON), &rb.Steps); err != nil {
		rb.Steps = []RunbookStep{}
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rb.Tags); err != nil {
		rb.Tags = []string{}
	}

	return &rb, nil
}

func scanRunbookRow(rows *sql.Rows) (*Runbook, error) {
	var rb Runbook
	var lastUsed int64
	var stepsJSON, tagsJSON string

	err := rows.Scan(&rb.ID, &rb.ProjectID, &rb.Name, &rb.Description, &stepsJSON, &rb.SuccessRate, &lastUsed, &rb.UsageCount, &tagsJSON)
	if err != nil {
		return nil, err
	}

	rb.LastUsed = time.Unix(lastUsed, 0)
	if err := json.Unmarshal([]byte(stepsJSON), &rb.Steps); err != nil {
		rb.Steps = []RunbookStep{}
	}
	if err := json.Unmarshal([]byte(tagsJSON), &rb.Tags); err != nil {
		rb.Tags = []string{}
	}

	return &rb, nil
}

func SaveProjectFingerprint(db *sql.DB, fp ProjectFingerprint) error {
	issuesJSON, err := json.Marshal(fp.CommonIssues)
	if err != nil {
		return fmt.Errorf("marshal common_issues: %w", err)
	}
	runbooksJSON, err := json.Marshal(fp.AssociatedRunbooks)
	if err != nil {
		return fmt.Errorf("marshal associated_runbooks: %w", err)
	}

	query := `INSERT OR REPLACE INTO project_fingerprints
		(id, project_type, package_manager, common_issues, associated_runbooks, detected_at, detected_time)
		VALUES (?, ?, ?, ?, ?, ?, ?)`

	_, err = db.Exec(query,
		fp.ID,
		fp.ProjectType,
		fp.PackageManager,
		string(issuesJSON),
		string(runbooksJSON),
		fp.DetectedAt,
		fp.DetectedTime.Unix(),
	)
	return err
}

func GetProjectFingerprint(db *sql.DB, path string) (*ProjectFingerprint, error) {
	query := `SELECT id, project_type, package_manager, common_issues, associated_runbooks, detected_at, detected_time
		FROM project_fingerprints WHERE detected_at = ?`

	row := db.QueryRow(query, path)
	return scanProjectFingerprint(row)
}

func GetProjectFingerprintByType(db *sql.DB, projectType string) ([]ProjectFingerprint, error) {
	query := `SELECT id, project_type, package_manager, common_issues, associated_runbooks, detected_at, detected_time
		FROM project_fingerprints WHERE project_type = ?`

	rows, err := db.Query(query, projectType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []ProjectFingerprint
	for rows.Next() {
		fp, err := scanProjectFingerprintRow(rows)
		if err != nil {
			return nil, err
		}
		results = append(results, *fp)
	}
	return results, nil
}

func scanProjectFingerprint(row *sql.Row) (*ProjectFingerprint, error) {
	var fp ProjectFingerprint
	var detectedTime int64
	var issuesJSON, runbooksJSON string

	err := row.Scan(&fp.ID, &fp.ProjectType, &fp.PackageManager, &issuesJSON, &runbooksJSON, &fp.DetectedAt, &detectedTime)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	fp.DetectedTime = time.Unix(detectedTime, 0)
	if err := json.Unmarshal([]byte(issuesJSON), &fp.CommonIssues); err != nil {
		fp.CommonIssues = []string{}
	}
	if err := json.Unmarshal([]byte(runbooksJSON), &fp.AssociatedRunbooks); err != nil {
		fp.AssociatedRunbooks = []string{}
	}

	return &fp, nil
}

func scanProjectFingerprintRow(rows *sql.Rows) (*ProjectFingerprint, error) {
	var fp ProjectFingerprint
	var detectedTime int64
	var issuesJSON, runbooksJSON string

	err := rows.Scan(&fp.ID, &fp.ProjectType, &fp.PackageManager, &issuesJSON, &runbooksJSON, &fp.DetectedAt, &detectedTime)
	if err != nil {
		return nil, err
	}

	fp.DetectedTime = time.Unix(detectedTime, 0)
	if err := json.Unmarshal([]byte(issuesJSON), &fp.CommonIssues); err != nil {
		fp.CommonIssues = []string{}
	}
	if err := json.Unmarshal([]byte(runbooksJSON), &fp.AssociatedRunbooks); err != nil {
		fp.AssociatedRunbooks = []string{}
	}

	return &fp, nil
}
