package cmd

import (
	"fmt"
	"os"

	"dev-cli/internal/hook"
)

func handleHook() {
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "Usage: dev-cli hook <shell>")
		fmt.Fprintln(os.Stderr, "Supported shells: zsh")
		os.Exit(1)
	}

	switch os.Args[2] {
	case "zsh":
		os.Stdout.WriteString(hook.ZshHook)
	default:
		fmt.Fprintf(os.Stderr, "Unsupported shell: %s\n", os.Args[2])
		fmt.Fprintln(os.Stderr, "Supported shells: zsh")
		os.Exit(1)
	}
}
