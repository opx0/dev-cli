package cmd

import (
	"bufio"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"dev-cli/internal/llm"

	"golang.org/x/term"
)

func handleRCA() {
	fs := flag.NewFlagSet("rca", flag.ExitOnError)
	command := fs.String("command", "", "The failed command")
	exitCode := fs.Int("exit-code", 0, "Exit code of the command")
	output := fs.String("output", "", "Command output (stdout/stderr)")
	interactive := fs.Bool("interactive", false, "Interactive mode with fix prompts")

	if err := fs.Parse(os.Args[2:]); err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing flags: %v\n", err)
		os.Exit(1)
	}

	// Skip if no command
	if *command == "" {
		return
	}

	// Skip Ctrl-C (exit code 130)
	if *exitCode == 130 {
		return
	}

	// In interactive mode, check if stdin is a TTY
	if *interactive && !term.IsTerminal(int(os.Stdin.Fd())) {
		return
	}

	// Print analyzing indicator
	fmt.Println("\n🔍 Analyzing failure...")

	// Call LLM
	client := llm.NewClient()
	result, err := client.Explain(*command, *exitCode, *output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "⚠️  Analysis failed: %v\n", err)
		return
	}

	// Print explanation
	fmt.Printf("\n💡 %s\n", result.Explanation)

	// If fix available and interactive, prompt user
	if result.Fix != "" && *interactive {
		fmt.Printf("\n📝 Suggested fix: %s\n", result.Fix)
		fmt.Print("   [Run Fix?] (y/n): ")

		reader := bufio.NewReader(os.Stdin)
		response, _ := reader.ReadString('\n')
		response = strings.TrimSpace(strings.ToLower(response))

		if response == "y" || response == "yes" {
			fmt.Printf("   Running: %s\n", result.Fix)
			cmd := exec.Command("sh", "-c", result.Fix)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Run(); err != nil {
				fmt.Fprintf(os.Stderr, "   ⚠️  Fix failed: %v\n", err)
			} else {
				fmt.Println("   ✅ Fix applied")
			}
		}
	}
}
