package cmd

import (
	"time"

	"github.com/briandowns/spinner"
)

// ── Spinner helper ───────────────────────────────────────────────────────────

// newSpinner creates a standard spinner with the given suffix text.
func newSpinner(suffix string) *spinner.Spinner {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " " + suffix
	s.Start()
	return s
}
