package cmd

import (
	"fmt"
	"os"
)

// ── ANSI color codes ─────────────────────────────────────────────────────────

const (
	colorReset  = "\033[0m"
	colorBold   = "\033[1m"
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorGray   = "\033[90m"
	colorWhite  = "\033[37m"

	// Bold color combinations
	boldGreen  = "\033[1;32m"
	boldCyan   = "\033[1;36m"
	boldYellow = "\033[1;33m"
	boldRed    = "\033[1;31m"
)

// ── Icons ────────────────────────────────────────────────────────────────────

func iconOK() string   { return colorGreen + "✓" + colorReset }
func iconFail() string { return colorRed + "✗" + colorReset }
func iconWarn() string { return colorYellow + "⚠" + colorReset }
func iconInfo() string { return colorCyan + "→" + colorReset }

// statusIcon returns a colored icon for ok/warn/fail status strings.
func statusIcon(status string) string {
	switch status {
	case "ok":
		return iconOK()
	case "warn":
		return iconWarn()
	case "fail":
		return iconFail()
	default:
		return "?"
	}
}

// ── Output helpers ───────────────────────────────────────────────────────────

// printSuccess prints a green success message to stdout.
func printSuccess(msg string) {
	fmt.Printf("%s %s\n", iconOK(), msg)
}

// printError prints a red error message to stderr.
func printError(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", iconFail(), msg)
}

// printWarning prints a yellow warning message to stderr.
func printWarning(msg string) {
	fmt.Fprintf(os.Stderr, "%s %s\n", iconWarn(), msg)
}

// printInfo prints a gray informational message to stdout.
func printInfo(msg string) {
	fmt.Printf("%s%s%s\n", colorGray, msg, colorReset)
}

// fmtBold wraps text in bold ANSI codes.
func fmtBold(s string) string {
	return colorBold + s + colorReset
}

// fmtColor wraps text in the given color and reset.
func fmtColor(color, s string) string {
	return color + s + colorReset
}

// separator prints a gray horizontal rule.
func separator() {
	fmt.Println(colorGray + "────────────────────────────────" + colorReset)
}
