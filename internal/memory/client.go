// Package memory wraps the external MemPalace CLI for hybrid recall and
// optional write-back of dev-cli's own outcomes (runbooks, resolutions, PR
// drafts). All operations are best-effort: when the binary is missing or a
// call fails, callers get an empty result rather than a user-visible error.
package memory

import (
	"context"
	"os/exec"
	"time"
)

const (
	defaultTimeout    = 3 * time.Second
	storeTimeout      = 5 * time.Second
	defaultSearchSize = 5
)

// findBinary returns the absolute path to the MemPalace CLI. Returns "" when
// neither `mempalace` nor `mp` is on PATH.
func findBinary() string {
	if path, err := exec.LookPath("mempalace"); err == nil {
		return path
	}
	if path, err := exec.LookPath("mp"); err == nil {
		return path
	}
	return ""
}

// Available reports whether the MemPalace CLI is on PATH. Cheap; does not
// invoke the binary.
func Available() bool { return findBinary() != "" }

// run shells the CLI with combined output capture. Combined (stdout+stderr)
// is intentional — search errors land on stderr and we want to see them when
// JSON parsing fails.
func run(ctx context.Context, args []string) ([]byte, error) {
	bin := findBinary()
	if bin == "" {
		return nil, errBinaryMissing
	}
	return exec.CommandContext(ctx, bin, args...).CombinedOutput()
}

type binaryMissingErr struct{}

func (binaryMissingErr) Error() string { return "mempalace CLI not on PATH" }

var errBinaryMissing = binaryMissingErr{}

// IsBinaryMissing reports whether err originates from a missing CLI.
func IsBinaryMissing(err error) bool {
	_, ok := err.(binaryMissingErr)
	return ok
}

// withDefaultTimeout returns a context honouring the parent. When the parent
// has no deadline, applies d as a fallback.
func withDefaultTimeout(parent context.Context, d time.Duration) (context.Context, context.CancelFunc) {
	if _, hasDeadline := parent.Deadline(); hasDeadline {
		return context.WithCancel(parent)
	}
	return context.WithTimeout(parent, d)
}
