package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileIsPrivateAndDoesNotFollowSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation may require elevated privileges")
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	targetPath := filepath.Join(dir, "target")
	if err := os.WriteFile(targetPath, []byte("unchanged"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(targetPath, configPath); err != nil {
		t.Fatal(err)
	}
	t.Setenv("DEV_CLI_CONFIG", configPath)

	var schema FileSchema
	schema.Ollama.Model = "test-model"
	if err := WriteFile(schema); err != nil {
		t.Fatal(err)
	}

	target, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(target) != "unchanged" {
		t.Fatal("config write followed the symlink")
	}
	info, err := os.Stat(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("config mode is %o, want 600", info.Mode().Perm())
	}
}
