package agent

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"

	"dev-cli/internal/llm"
)

const maxRetries = 3

type Agent struct {
	client *llm.HybridClient
}

func New() *Agent {
	return &Agent{
		client: llm.NewHybridClient(),
	}
}

func (a *Agent) Resolve(issue string, approval func(string) bool) error {
	context := issue
	var lastError string

	for attempt := 1; attempt <= maxRetries; attempt++ {
		s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
		if attempt == 1 {
			s.Suffix = " > Analyzing..."
		} else {
			s.Suffix = fmt.Sprintf(" > Retry %d/%d...", attempt, maxRetries)
		}
		s.Start()

		prompt := context
		if lastError != "" {
			prompt = fmt.Sprintf("Previous command failed with:\n%s\n\nOriginal task: %s\n\nPlease provide a corrected command.", lastError, issue)
		}

		proposal, err := a.client.Solve(prompt)
		s.Stop()

		if err != nil {
			fmt.Printf("  x LLM error: %v\n", err)
			return err
		}

		if proposal == "" {
			fmt.Println("  ! No solution found")
			return fmt.Errorf("no solution")
		}

		if !approval(proposal) {
			return fmt.Errorf("denied by user")
		}

		fmt.Printf("\n  > Running: %s\n", proposal)
		success, errOutput := a.executeWithCapture(proposal)

		if success {
			fmt.Println("  + Done")
			return nil
		}

		lastError = truncate(errOutput, 500)
		fmt.Printf("  x Failed. Retrying...\n")
	}

	fmt.Println("  x Max retries reached")
	return fmt.Errorf("max retries exceeded")
}

func (a *Agent) executeWithCapture(command string) (bool, string) {
	cmd := exec.Command("sh", "-c", command)

	var stderrBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	return err == nil, stderrBuf.String()
}

func truncate(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
