package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var (
	testHandoff bool
	testVerbose bool
)

var testCmd = &cobra.Command{
	Use:   "test [--] [extra args]",
	Short: "Run the project's test suite; on failure, summarise and offer to fix it",
	Long: `Auto-detect the project's test runner and execute it:

  go.mod              -> go test ./...
  package.json        -> npm test
  pyproject.toml /
  requirements.txt    -> pytest
  Cargo.toml          -> cargo test

Override detection with $DEV_CLI_TEST_CMD. Extra arguments after -- are
forwarded to the runner.

On non-zero exit, dev-cli summarises the first failing block and lets you
hand it over to the fix agent with a single keystroke.`,
	Example: `  dev-cli test
  dev-cli test -- ./internal/llm/...
  DEV_CLI_TEST_CMD="just test-fast" dev-cli test`,
	DisableFlagParsing: false,
	Run:                runTest,
}

func init() {
	testCmd.Flags().BoolVar(&testHandoff, "auto-fix", false, "On failure, run `dev-cli fix` without prompting")
	testCmd.Flags().BoolVarP(&testVerbose, "verbose", "v", false, "Stream full output instead of only failures")
	rootCmd.AddCommand(testCmd)
}

func runTest(cmd *cobra.Command, args []string) {
	bin, runnerArgs, label := detectTestRunner()
	runnerArgs = append(runnerArgs, args...)

	if bin == "" {
		printError("could not detect a test runner; set $DEV_CLI_TEST_CMD")
		os.Exit(2)
	}

	printInfo(fmt.Sprintf("Running %s: %s %s", label, bin, strings.Join(runnerArgs, " ")))
	start := time.Now()

	exitCode, output, err := streamCommand(bin, runnerArgs, testVerbose)
	duration := time.Since(start)

	if err != nil && exitCode == 0 {
		// Something non-test-related blew up (exec failure, for example).
		printError(fmt.Sprintf("runner error: %v", err))
		os.Exit(1)
	}

	if exitCode == 0 {
		printSuccess(fmt.Sprintf("Tests passed in %s", duration.Round(time.Millisecond)))
		return
	}

	summary := summariseFailure(output)
	fmt.Println()
	printError(fmt.Sprintf("Tests failed (exit %d, %s)", exitCode, duration.Round(time.Millisecond)))
	if summary != "" {
		fmt.Printf("\n%s First failure:%s\n%s\n", colorBold, colorReset, summary)
	}

	if testHandoff {
		handoffToFix(label, summary, output)
		return
	}

	fmt.Print("\n[F]ix via agent  [E]xplain  [Q]uit: ")
	var resp string
	fmt.Scanln(&resp)
	switch strings.ToLower(strings.TrimSpace(resp)) {
	case "f", "fix":
		handoffToFix(label, summary, output)
	case "e", "explain":
		handoffToExplain(output)
	}
	os.Exit(exitCode)
}

// detectTestRunner picks the right test command for the current working dir.
// Returns (binary, args, human-readable label). $DEV_CLI_TEST_CMD short-circuits
// detection and is parsed as a `sh -c` string.
func detectTestRunner() (bin string, args []string, label string) {
	if custom := strings.TrimSpace(os.Getenv("DEV_CLI_TEST_CMD")); custom != "" {
		return "sh", []string{"-c", custom}, "custom ($DEV_CLI_TEST_CMD)"
	}
	cwd, _ := os.Getwd()
	check := func(name string) bool {
		_, err := os.Stat(filepath.Join(cwd, name))
		return err == nil
	}

	switch {
	case check("go.mod"):
		return "go", []string{"test", "./..."}, "go"
	case check("Cargo.toml"):
		return "cargo", []string{"test"}, "cargo"
	case check("package.json"):
		return "npm", []string{"test", "--silent"}, "npm"
	case check("pyproject.toml"), check("pytest.ini"), check("setup.cfg"):
		if _, err := exec.LookPath("pytest"); err == nil {
			return "pytest", nil, "pytest"
		}
		return "python", []string{"-m", "unittest"}, "unittest"
	case check("requirements.txt"):
		if _, err := exec.LookPath("pytest"); err == nil {
			return "pytest", nil, "pytest"
		}
	case check("pom.xml"):
		return "mvn", []string{"test"}, "maven"
	case check("build.gradle"), check("build.gradle.kts"):
		return "gradle", []string{"test"}, "gradle"
	}
	return "", nil, ""
}

// streamCommand runs a subprocess, tees stdout+stderr to the terminal (unless
// `verbose=false` in which case we buffer and only print on failure), and
// returns the combined captured output for downstream summarisation.
func streamCommand(bin string, args []string, verbose bool) (int, string, error) {
	cmd := exec.Command(bin, args...)
	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return 0, "", err
	}

	var buf strings.Builder
	drain := func(r io.Reader) {
		scanner := bufio.NewScanner(r)
		scanner.Buffer(make([]byte, 1024*1024), 16*1024*1024)
		for scanner.Scan() {
			line := scanner.Text()
			buf.WriteString(line)
			buf.WriteByte('\n')
			if verbose {
				fmt.Println(line)
			}
		}
	}

	done := make(chan struct{}, 2)
	go func() { drain(stdout); done <- struct{}{} }()
	go func() { drain(stderr); done <- struct{}{} }()
	<-done
	<-done

	err := cmd.Wait()
	exit := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exit = exitErr.ExitCode()
			err = nil
		}
	}
	return exit, buf.String(), err
}

// summariseFailure returns a compact snippet around the first failing block.
// We don't parse runner-specific output; a rolling window around the first
// "FAIL"/"FAILED"/"error:" hit is enough signal for the agent and the user.
func summariseFailure(output string) string {
	lines := strings.Split(output, "\n")
	markers := []string{"FAIL", "FAILED", "AssertionError", "panic:", "error:", "Error:", "× ", "✗ "}

	for i, line := range lines {
		for _, m := range markers {
			if strings.Contains(line, m) {
				start := i - 2
				if start < 0 {
					start = 0
				}
				end := i + 20
				if end > len(lines) {
					end = len(lines)
				}
				return strings.Join(lines[start:end], "\n")
			}
		}
	}

	// No marker found — return the tail of the output.
	const tailLines = 40
	if len(lines) > tailLines {
		return strings.Join(lines[len(lines)-tailLines:], "\n")
	}
	return output
}

func handoffToFix(label, summary, full string) {
	issue := fmt.Sprintf("%s tests failing. First failure:\n%s", label, summary)
	if summary == "" {
		const tail = 2000
		if len(full) > tail {
			full = full[len(full)-tail:]
		}
		issue = fmt.Sprintf("%s tests failing. Output tail:\n%s", label, full)
	}
	fmt.Println()
	printInfo("Handing off to fix agent…")
	runFix(fixCmd, []string{issue})
}

func handoffToExplain(output string) {
	fmt.Println()
	printInfo("Use `dev-cli explain --last 1` or paste output into `dev-cli ask`.")
	// Keep the last chunk for the user's clipboard convenience.
	const tail = 4000
	if len(output) > tail {
		output = output[len(output)-tail:]
	}
	fmt.Println(output)
	_ = context.Background() // keep import slot if we later add an agent call
}
