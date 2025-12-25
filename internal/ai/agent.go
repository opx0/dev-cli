package ai

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/briandowns/spinner"
)

const maxRetries = 3

type Solver interface {
	Solve(goal string) (string, error)
}

type Executor interface {
	Execute(command string) (success bool, errOutput string)
}

type Agent struct {
	solver   Solver
	executor Executor
}

func NewAgent() *Agent {
	return &Agent{
		solver:   NewHybridClient(),
		executor: &shellExecutor{},
	}
}

func NewAgentWithDeps(solver Solver, executor Executor) *Agent {
	return &Agent{
		solver:   solver,
		executor: executor,
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

		proposal, err := a.solver.Solve(prompt)
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
		success, errOutput := a.executor.Execute(proposal)

		if success {
			fmt.Println("  + Done")
			return nil
		}

		lastError = truncateAgent(errOutput, 500)
		fmt.Printf("  x Failed. Retrying...\n")
	}

	fmt.Println("  x Max retries reached")
	return fmt.Errorf("max retries exceeded")
}

type shellExecutor struct{}

func (e *shellExecutor) Execute(command string) (bool, string) {
	cmd := exec.Command("sh", "-c", command)

	var stderrBuf bytes.Buffer
	cmd.Stdout = os.Stdout
	cmd.Stderr = io.MultiWriter(os.Stderr, &stderrBuf)
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	return err == nil, stderrBuf.String()
}

func truncateAgent(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
