package memory

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"dev-cli/internal/config"
)

// stubBin writes a fake `mempalace` shell script into a tempdir, prepends it
// to PATH, and returns a cleanup. The script:
//   - logs every invocation's args (one line per call) to <tmp>/calls.log
//   - emits the contents of <tmp>/stdout.json on `search` calls
//   - exits with code from <tmp>/exit if present, else 0.
func stubBin(t *testing.T, stdout string) (logPath string, cleanup func()) {
	t.Helper()
	tmp := t.TempDir()

	stdoutPath := filepath.Join(tmp, "stdout.json")
	if err := os.WriteFile(stdoutPath, []byte(stdout), 0o644); err != nil {
		t.Fatalf("write stub stdout: %v", err)
	}

	script := `#!/bin/sh
echo "$@" >> "` + filepath.Join(tmp, "calls.log") + `"
case "$1" in
  search) cat "` + stdoutPath + `" ;;
  store)  echo "stored" ;;
  *)      echo "stub" ;;
esac
exit 0
`
	binPath := filepath.Join(tmp, "mempalace")
	if err := os.WriteFile(binPath, []byte(script), 0o755); err != nil {
		t.Fatalf("write stub script: %v", err)
	}

	prevPath := os.Getenv("PATH")
	t.Setenv("PATH", tmp+":"+prevPath)

	return filepath.Join(tmp, "calls.log"), func() {}
}

func readLog(t *testing.T, path string) []string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read log: %v", err)
	}
	out := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	if len(out) == 1 && out[0] == "" {
		return nil
	}
	return out
}

func TestSearch_Disabled_NoCall(t *testing.T) {
	logPath, cleanup := stubBin(t, `[]`)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: false}
	hits, err := Search(context.Background(), cfg, "anything", SearchOpts{})
	if err != nil {
		t.Fatalf("Search returned error when disabled: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected 0 hits when disabled, got %d", len(hits))
	}
	if logged := readLog(t, logPath); len(logged) != 0 {
		t.Fatalf("expected stub never invoked when disabled, got %v", logged)
	}
}

func TestSearch_EmptyQuery_NoCall(t *testing.T) {
	logPath, cleanup := stubBin(t, `[]`)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: true, MemPalaceLimit: 5}
	hits, err := Search(context.Background(), cfg, "   ", SearchOpts{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected 0 hits for blank query, got %d", len(hits))
	}
	if logged := readLog(t, logPath); len(logged) != 0 {
		t.Fatalf("expected stub never invoked for blank query, got %v", logged)
	}
}

func TestSearch_ParsesJSON(t *testing.T) {
	stdout := `[
		{"id":"m1","score":0.91,"wing":"dev-cli/go/proj","hall":"advice","text":"hit one","source":"dev-cli/runbook"},
		{"id":"m2","score":0.42,"wing":"obsidian","hall":"fact","text":"hit two"}
	]`
	logPath, cleanup := stubBin(t, stdout)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: true, MemPalaceLimit: 10, MemPalaceWing: "obsidian"}
	hits, err := Search(context.Background(), cfg, "kubectl", SearchOpts{Limit: 3, Hall: "advice"})
	if err != nil {
		t.Fatalf("Search returned error: %v", err)
	}
	if len(hits) != 2 {
		t.Fatalf("expected 2 hits, got %d", len(hits))
	}
	if hits[0].ID != "m1" || hits[0].Score != 0.91 {
		t.Errorf("first hit unexpected: %+v", hits[0])
	}
	if hits[1].Hall != "fact" {
		t.Errorf("second hit hall: %q", hits[1].Hall)
	}

	logged := readLog(t, logPath)
	if len(logged) != 1 {
		t.Fatalf("expected 1 invocation, got %d: %v", len(logged), logged)
	}
	args := logged[0]
	for _, want := range []string{"search", "kubectl", "--limit", "3", "--json", "--wing", "obsidian", "--hall", "advice"} {
		if !strings.Contains(args, want) {
			t.Errorf("expected %q in args, got %q", want, args)
		}
	}
}

func TestSearch_MissingBinary_SoftFail(t *testing.T) {
	t.Setenv("PATH", t.TempDir()) // empty PATH guarantees missing
	cfg := &config.Config{MemPalaceEnabled: true, MemPalaceLimit: 5}
	hits, err := Search(context.Background(), cfg, "x", SearchOpts{})
	if err == nil || !IsBinaryMissing(err) {
		t.Fatalf("expected binary-missing error, got: %v", err)
	}
	if len(hits) != 0 {
		t.Fatalf("expected 0 hits on missing binary, got %d", len(hits))
	}
}

func TestStore_Disabled_NoCall(t *testing.T) {
	logPath, cleanup := stubBin(t, `[]`)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: false}
	if err := Store(context.Background(), cfg, StoreReq{Text: "x", Hall: "advice"}); err != nil {
		t.Fatalf("Store returned error when disabled: %v", err)
	}
	if logged := readLog(t, logPath); len(logged) != 0 {
		t.Fatalf("expected no calls when disabled, got %v", logged)
	}
}

func TestStore_PassesArgs(t *testing.T) {
	logPath, cleanup := stubBin(t, `[]`)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: true, MemPalaceWing: "fixed-wing"}
	err := Store(context.Background(), cfg, StoreReq{
		Hall:   "advice",
		Text:   "always pin sqlite-vec on Arch",
		Source: "dev-cli/runbook:abc",
	})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	logged := readLog(t, logPath)
	if len(logged) != 1 {
		t.Fatalf("expected 1 call, got %d: %v", len(logged), logged)
	}
	for _, want := range []string{"store", "always pin sqlite-vec on Arch", "--wing", "fixed-wing", "--hall", "advice", "--source", "dev-cli/runbook:abc"} {
		if !strings.Contains(logged[0], want) {
			t.Errorf("missing %q in args: %q", want, logged[0])
		}
	}
}

func TestStore_EmptyText_NoCall(t *testing.T) {
	logPath, cleanup := stubBin(t, `[]`)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: true}
	if err := Store(context.Background(), cfg, StoreReq{Text: "  \n  "}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if logged := readLog(t, logPath); len(logged) != 0 {
		t.Fatalf("expected no call for empty text, got %v", logged)
	}
}

func TestBuildPromptContext(t *testing.T) {
	stdout := `[{"id":"m1","score":0.9,"wing":"dev-cli/go","hall":"advice","text":"prior fix"}]`
	_, cleanup := stubBin(t, stdout)
	defer cleanup()

	cfg := &config.Config{MemPalaceEnabled: true, MemPalaceLimit: 5}
	ctx, ok := BuildPromptContext(cfg, "anything")
	if !ok {
		t.Fatalf("expected hit")
	}
	if !strings.Contains(ctx, "Relevant prior memory") {
		t.Errorf("missing header: %q", ctx)
	}
	if !strings.Contains(ctx, "prior fix") {
		t.Errorf("missing hit text: %q", ctx)
	}
}

func TestDeriveWing_GoProject(t *testing.T) {
	tmp := t.TempDir()
	if err := os.WriteFile(filepath.Join(tmp, "go.mod"), []byte("module x\n"), 0o644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	wing := deriveWing(tmp)
	if !strings.HasPrefix(wing, "dev-cli/go/") {
		t.Errorf("expected dev-cli/go/ prefix, got %q", wing)
	}
	if !strings.Contains(wing, filepath.Base(tmp)) {
		t.Errorf("expected dir name in wing, got %q", wing)
	}
}

func TestDeriveWing_Generic(t *testing.T) {
	tmp := t.TempDir() // no manifest files
	wing := deriveWing(tmp)
	if !strings.HasPrefix(wing, "dev-cli/generic/") {
		t.Errorf("expected dev-cli/generic/ prefix, got %q", wing)
	}
}

func TestSanitizeWingSegment(t *testing.T) {
	cases := map[string]string{
		"normal":       "normal",
		"with space":   "with-space",
		"weird@chars!": "weird-chars-",
		"":             "",
		".":            "",
	}
	for in, want := range cases {
		got := sanitizeWingSegment(in)
		if got != want {
			t.Errorf("sanitizeWingSegment(%q) = %q, want %q", in, got, want)
		}
	}
}
