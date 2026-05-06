package memory

import (
	"os"
	"path/filepath"
	"strings"

	"dev-cli/internal/storage"
)

// deriveWing returns a project-scoped wing string of the form
// "dev-cli/<projectType>/<dirName>" (e.g. "dev-cli/go/dev-cli"). The "dev-cli"
// prefix lets users distinguish dev-cli-managed memories from their curated
// vault, and the project-type segment lets recall queries be scoped to "all
// my Go projects" without hard-coding paths.
//
// When cwd is "", uses os.Getwd(). When fingerprint detection fails, falls
// back to "dev-cli/generic". Never panics.
func deriveWing(cwd string) string {
	if cwd == "" {
		if wd, err := os.Getwd(); err == nil {
			cwd = wd
		}
	}
	if cwd == "" {
		return "dev-cli/generic"
	}

	_, projectType, _ := storage.DetectProjectFingerprintID(cwd)
	if projectType == "" {
		projectType = "generic"
	}

	dirName := filepath.Base(cwd)
	dirName = sanitizeWingSegment(dirName)
	if dirName == "" {
		return "dev-cli/" + projectType
	}
	return "dev-cli/" + projectType + "/" + dirName
}

// sanitizeWingSegment drops shell-unsafe characters from a wing segment.
// MemPalace itself accepts arbitrary strings, but keeping segments URL-safe
// avoids surprises when the same memories show up in the markdown export
// path tree.
func sanitizeWingSegment(s string) string {
	s = strings.TrimSpace(s)
	if s == "" || s == "." || s == "/" {
		return ""
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '-', r == '_', r == '.':
			b.WriteRune(r)
		default:
			b.WriteRune('-')
		}
	}
	return b.String()
}
