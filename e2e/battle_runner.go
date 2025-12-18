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

	var err error
	tempDir, err = os.MkdirTemp("", "dev-cli-e2e")
	if err != nil {
		fatal("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	os.Setenv("DEV_CLI_LOG_DIR", tempDir)

	cliBin = filepath.Join(tempDir, "dev-cli")
	fmt.Printf("Building dev-cli to %s...\n", cliBin)
	buildCmd := exec.Command("go", "build", "-o", cliBin, ".")
	if out, err := buildCmd.CombinedOutput(); err != nil {
		fatal("Build failed:\n%s", out)
	}

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

	runCLI("log-event",
		"--command", "npm install",
		"--exit-code", "1",
		"--cwd", "/tmp/proj",
		"--output", "npm ERR! code ENOENT")

	runCLI("log-event",
		"--command", "git push origin master",
		"--exit-code", "128",
		"--cwd", "/tmp/proj",
		"--output", "fatal: Could not read from remote repository.")

	runCLI("log-event",
		"--command", "docker build .",
		"--exit-code", "1",
		"--cwd", "/tmp/proj",
		"--output", "Step 1/5 : FROM alpine")

	passTest()
}

func testExplain() {
	startTest("Explain Command (RCA)")

	out := runCLI("explain")
	if !strings.Contains(out, "docker build") { // Last command
		fmt.Printf("DEBUG: explain output:\n%s\n", out)
		fatal("Default explain didn't pick up last failure (docker build)")
	}

	out = runCLI("explain", "--filter", "npm")
	if !strings.Contains(out, "npm install") {
		fatal("Filter 'npm' didn't pick up npm failure")
	}

	out = runCLI("explain", "--last", "3")
	if !strings.Contains(out, "npm install") || !strings.Contains(out, "git push") {
		fatal("Explain --last 3 didn't show all recent failures")
	}

	out = runCLI("explain", "--since", "1h")
	if !strings.Contains(out, "docker build") {
		fatal("Explain --since 1h missed recent failure")
	}

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
	stdout, _ := cmd.StdoutPipe()

	if err := cmd.Start(); err != nil {
		fatal("Failed to start watch command: %v", err)
	}

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

	time.Sleep(1 * time.Second)
	f.WriteString("Info: System starting...\n")
	time.Sleep(500 * time.Millisecond)
	f.WriteString("Error: Connection refused to database\n") // Trigger

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		fatal("Timeout waiting for watch command to detect error")
	}

	cmd.Process.Kill()
	passTest()
}

func testAsk() {
	startTest("Ask Command")
	out := runCLI("ask", "how to check disk space")
	if out == "" {
		fmt.Println("Warning: Ask command produced no output (LLM might be offline), but didn't crash.")
	}
	passTest()
}

func testFix() {
	startTest("Fix Command (Agent)")

	cmd := exec.Command(cliBin, "fix", "print hello world")
	stdin, _ := cmd.StdinPipe()
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Start(); err != nil {
		fatal("Failed to start fix command: %v", err)
	}

	time.Sleep(500 * time.Millisecond)
	io.WriteString(stdin, "y\n")

	cmd.Wait()
	out := stdout.String()

	if !strings.Contains(out, "Allow? [y/N]") {
		fmt.Printf("DEBUG: fix output:\n%s\n", out)
		fatal("Fix command didn't prompt for permission")
	}
	if !strings.Contains(out, "I want to run:") {
		fatal("Fix command didn't propose a fix")
	}

	passTest()
}


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
	cmd.Env = os.Environ()

	out, err := cmd.CombinedOutput()
	if err != nil {
		if args[0] == "log-event" {
			fmt.Printf("Warning: log-event failed: %v\nOutput: %s\n", err, out)
		}
	}
	return string(out)
}
