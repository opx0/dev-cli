package storage

import (
	"database/sql"
	"os"
	"testing"
	"time"
)

func setupTestDB(t *testing.T) *sql.DB {

	tmpFile, err := os.CreateTemp("", "rca_test_*.db")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	tmpFile.Close()

	db, err := OpenDB(tmpFile.Name())
	if err != nil {
		t.Fatalf("failed to open DB: %v", err)
	}

	t.Cleanup(func() {
		db.Close()
		os.Remove(tmpFile.Name())
	})

	return db
}

func TestRootCause_CRUD(t *testing.T) {
	db := setupTestDB(t)

	rc := RootCause{
		ID:               "rc-001",
		ErrorSignature:   "npm-enoent-001",
		Timestamp:        time.Now(),
		RootCauseNodes:   []string{"missing package.json", "npm install required"},
		RemediationSteps: []string{"npm install", "retry command"},
		Confidence:       0.85,
		HistoryItemID:    1,
	}

	err := SaveRootCause(db, rc)
	if err != nil {
		t.Fatalf("SaveRootCause failed: %v", err)
	}

	retrieved, err := GetRootCauseBySignature(db, "npm-enoent-001")
	if err != nil {
		t.Fatalf("GetRootCauseBySignature failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected to find root cause, got nil")
	}
	if retrieved.ID != "rc-001" {
		t.Errorf("expected ID 'rc-001', got '%s'", retrieved.ID)
	}
	if len(retrieved.RootCauseNodes) != 2 {
		t.Errorf("expected 2 root cause nodes, got %d", len(retrieved.RootCauseNodes))
	}
	if retrieved.Confidence != 0.85 {
		t.Errorf("expected confidence 0.85, got %f", retrieved.Confidence)
	}

	byID, err := GetRootCauseByID(db, "rc-001")
	if err != nil {
		t.Fatalf("GetRootCauseByID failed: %v", err)
	}
	if byID == nil || byID.ID != "rc-001" {
		t.Error("GetRootCauseByID should return the correct root cause")
	}

	rcList, err := GetRecentRootCauses(db, 10)
	if err != nil {
		t.Fatalf("GetRecentRootCauses failed: %v", err)
	}
	if len(rcList) != 1 {
		t.Errorf("expected 1 root cause, got %d", len(rcList))
	}
}

func TestRunbook_CRUD(t *testing.T) {
	db := setupTestDB(t)

	rb := Runbook{
		ID:          "rb-001",
		ProjectID:   "proj-nodejs",
		Name:        "NPM Install Fix",
		Description: "Fixes missing dependency issues",
		Steps: []RunbookStep{
			{ID: "s1", Name: "Check package.json", Command: "cat package.json", Description: "Verify package.json exists"},
			{ID: "s2", Name: "Install dependencies", Command: "npm install", Rollback: "rm -rf node_modules"},
		},
		SuccessRate: 0.9,
		LastUsed:    time.Now(),
		UsageCount:  10,
		Tags:        []string{"npm", "dependencies"},
	}

	err := SaveRunbook(db, rb)
	if err != nil {
		t.Fatalf("SaveRunbook failed: %v", err)
	}

	retrieved, err := GetRunbookByID(db, "rb-001")
	if err != nil {
		t.Fatalf("GetRunbookByID failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected to find runbook, got nil")
	}
	if retrieved.Name != "NPM Install Fix" {
		t.Errorf("expected name 'NPM Install Fix', got '%s'", retrieved.Name)
	}
	if len(retrieved.Steps) != 2 {
		t.Errorf("expected 2 steps, got %d", len(retrieved.Steps))
	}
	if retrieved.Steps[0].Command != "cat package.json" {
		t.Errorf("expected first step command 'cat package.json', got '%s'", retrieved.Steps[0].Command)
	}

	projectRunbooks, err := GetRunbooksForProject(db, "proj-nodejs")
	if err != nil {
		t.Fatalf("GetRunbooksForProject failed: %v", err)
	}
	if len(projectRunbooks) != 1 {
		t.Errorf("expected 1 runbook for project, got %d", len(projectRunbooks))
	}

	err = UpdateRunbookStats(db, "rb-001", true)
	if err != nil {
		t.Fatalf("UpdateRunbookStats failed: %v", err)
	}

	updated, _ := GetRunbookByID(db, "rb-001")
	if updated.UsageCount != 11 {
		t.Errorf("expected usage count 11, got %d", updated.UsageCount)
	}
}

func TestProjectFingerprint_CRUD(t *testing.T) {
	db := setupTestDB(t)

	fp := ProjectFingerprint{
		ID:                 "fp-001",
		ProjectType:        "nodejs",
		PackageManager:     "npm",
		CommonIssues:       []string{"ENOENT", "MODULE_NOT_FOUND"},
		AssociatedRunbooks: []string{"rb-001", "rb-002"},
		DetectedAt:         "/home/user/project",
		DetectedTime:       time.Now(),
	}

	err := SaveProjectFingerprint(db, fp)
	if err != nil {
		t.Fatalf("SaveProjectFingerprint failed: %v", err)
	}

	retrieved, err := GetProjectFingerprint(db, "/home/user/project")
	if err != nil {
		t.Fatalf("GetProjectFingerprint failed: %v", err)
	}
	if retrieved == nil {
		t.Fatal("expected to find fingerprint, got nil")
	}
	if retrieved.ProjectType != "nodejs" {
		t.Errorf("expected project type 'nodejs', got '%s'", retrieved.ProjectType)
	}
	if len(retrieved.CommonIssues) != 2 {
		t.Errorf("expected 2 common issues, got %d", len(retrieved.CommonIssues))
	}

	byType, err := GetProjectFingerprintByType(db, "nodejs")
	if err != nil {
		t.Fatalf("GetProjectFingerprintByType failed: %v", err)
	}
	if len(byType) != 1 {
		t.Errorf("expected 1 fingerprint, got %d", len(byType))
	}
}

func TestErrorSignature(t *testing.T) {
	sig1 := GenerateErrorSignature("npm install", 1, "ENOENT: no such file")
	sig2 := GenerateErrorSignature("npm install", 1, "ENOENT: no such file")
	sig3 := GenerateErrorSignature("npm install", 1, "EACCES: permission denied")

	if sig1 != sig2 {
		t.Error("same input should produce same signature")
	}
	if sig1 == sig3 {
		t.Error("different errors should produce different signatures")
	}
}
