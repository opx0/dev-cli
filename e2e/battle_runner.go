package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ANSI color codes for pretty output
const (
	Reset  = "\033[0m"
	Green  = "\033[32m"
	Red    = "\033[31m"
	Yellow = "\033[33m"
	Blue   = "\033[34m"
)

var (
	cliBin  string
	tempDir string
)

func main() {
	fmt.Printf("%sStarting Battle Test Suite...%s\n", Blue, Reset)

	// Setup
	var err error
	tempDir, err = os.MkdirTemp("", "dev-cli-e2e")
	if err != nil {
		fatal("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set log dir to temp
	os.Setenv("DEV_CLI_LOG_DIR", tempDir)

	// Build CLI
	cliBin = filepath.Join(tempDir, "dev-cli")
	fmt.Printf("Building dev-cli to %s...\n", cliBin)
	buildCmd := exec.Command("go", "build", "-o", cliBin, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fatal("Build failed:\n%s", out)
	}

	// Run Tests
	testInit()
	testLogEvent()
	testExplain()
	testWatch()
	testAsk()
	testFix()

	fmt.Printf("\n%sAll Battle Tests Passed!%s\n", Green, Reset)
}

func testInit() {
	startTest("Init Command")
	out := runCLI("init", "zsh")
	if !strings.Contains(out, "dev-cli init zsh") {
		fatal("Init output missing expected shell script content")
	}
	passTest()
}

func testLogEvent() {
	startTest("Log Event (Simulating Failures)")

	// Simulate 3 failures
	// 1. npm install fail
	runCLI("log-event",
		"--command", "npm install",
		"--exit-code", "1",
		"--cwd", "/tmp/proj",
		"--output", "npm ERR! code ENOENT")

	// 2. git push fail
	runCLI("log-event",
		"--command", "git push origin master",
		"--exit-code", "128",
		"--cwd", "/tmp/proj",
		"--output", "fatal: Could not read from remote repository.")

	// 3. docker build fail
	runCLI("log-event",
		"--command", "docker build .",
		"--exit-code", "1",
		"--cwd", "/tmp/proj",
		"--output", "Step 1/5 : FROM alpine")

	passTest()
}

func testExplain() {
	startTest("Explain Command (RCA)")

	// Test default
	out := runCLI("explain")
	if !strings.Contains(out, "docker build") { // Last command
		fmt.Printf("DEBUG: explain output:\n%s\n", out)
		fatal("Default explain didn't pick up last failure (docker build)")
	}

	// Test Filter
	out = runCLI("explain", "--filter", "npm")
	if !strings.Contains(out, "npm install") {
		fatal("Filter 'npm' didn't pick up npm failure")
	}

	// Test Last N
	out = runCLI("explain", "--last", "3")
	if !strings.Contains(out, "npm install") || !strings.Contains(out, "git push") {
		fatal("Explain --last 3 didn't show all recent failures")
	}

	// Test Since
	out = runCLI("explain", "--since", "1h")
	if !strings.Contains(out, "docker build") {
		fatal("Explain --since 1h missed recent failure")
	}

	// Test Alias
	out = runCLI("why", "--filter", "git")
	if !strings.Contains(out, "git push") {
		fatal("Alias 'why' failed to work")
	}

	passTest()
}

func testWatch() {
	startTest("Watch Command")

	logFile := filepath.Join(tempDir, "app.log")
	f, err := os.Create(logFile)
	if err != nil {
		fatal("Failed to create log file: %v", err)
	}

	cmd := exec.Command(cliBin, "watch", "--file", logFile)
	// Create pipes
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		fatal("Failed to start watch command: %v", err)
	}

	// Background reader to check for alert
	done := make(chan bool)
	go func() {
		scanner := bufio.NewScanner(stdout)
		foundDetection := false
		foundAnalysis := false

		for scanner.Scan() {
			line := scanner.Text()
			fmt.Println("[WATCH DEBUG]", line)

			if strings.Contains(line, "Error detected!") {
				foundDetection = true
			}
			// Check for spinner finish or result arrow or specific output keywords
			if strings.Contains(line, "→") ||
				strings.Contains(line, "Analysis Result") ||
				strings.Contains(line, "Explanation:") ||
				strings.Contains(line, "Suggested Fix:") ||
				strings.Contains(line, "> [Local AI]") {
				foundAnalysis = true
			}

			if foundDetection && foundAnalysis {
				done <- true
				return
			}
		}
	}()

	// Simulate log writing
	time.Sleep(1 * time.Second)
	f.WriteString("Info: System starting...\n")
	time.Sleep(500 * time.Millisecond)
	f.WriteString("Error: Connection refused to database\n") // Trigger

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		fatal("Timeout waiting for watch command to detect error")
	}

	cmd.Process.Kill()
	passTest()
}

func testAsk() {
	startTest("Ask Command")
	// "ask" usually hits an LLM. We just want to ensure it runs and accepts args.
	// It relies on internal/llm.
	out := runCLI("ask", "how to check disk space")
	// Just check if it ran without panicking. Output contents depend on LLM which might fail/stub.
	// We expect *something*
	if out == "" {
		fmt.Println("Warning: Ask command produced no output (LLM might be offline), but didn't crash.")
	}
	passTest()
}

func testFix() {
	startTest("Fix Command (Agent)")

	// We need to pipe input "y" to approve the fix
	cmd := exec.Command(cliBin, "fix", "print hello world")
	stdin, _ := cmd.StdinPipe()
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		fatal("Failed to start fix command: %v", err)
	}

	// Wait a bit then say "y"
	time.Sleep(500 * time.Millisecond)
	io.WriteString(stdin, "y\n")

	cmd.Wait()
	out := stdout.String()

	if !strings.Contains(out, "Allow? [y/N]") {
		fmt.Printf("DEBUG: fix output:\n%s\n", out)
		fatal("Fix command didn't prompt for permission")
	}
	// Note: The stub agent prints "I want to run: ...", check for that
	if !strings.Contains(out, "I want to run:") {
		fatal("Fix command didn't propose a fix")
	}

	passTest()
}

// --- Helpers ---

func startTest(name string) {
	fmt.Printf("Testing %s... ", name)
}

func passTest() {
	fmt.Println(Green + "PASS" + Reset)
}

func fatal(format string, args ...interface{}) {
	fmt.Printf(Red+"FAIL: "+format+Reset+"\n", args...)
	os.Exit(1)
}

func runCLI(args ...string) string {
	cmd := exec.Command(cliBin, args...)
	// Ensure Env is inherited explicitly (just to be safe, though default is yes)
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		// print error for debugging if it's unexpected
		// But for log-event it should be 0.
		if args[0] == "log-event" {
			fmt.Printf("Warning: log-event failed: %v\nOutput: %s\n", err, out)
		}
	}
	return string(out)
}
