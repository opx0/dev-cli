package agent

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/briandowns/spinner"

	"dev-cli/internal/llm"
)

type Agent struct {
	client *llm.HybridClient
}

func New() *Agent {
	return &Agent{
		client: llm.NewHybridClient(),
	}
}

func (a *Agent) Resolve(issue string, approval func(string) bool) error {
	s := spinner.New(spinner.CharSets[14], 100*time.Millisecond)
	s.Suffix = " 🧠 Designing solution..."
	s.Start()
	proposal, err := a.client.Solve(issue)
	s.Stop()

	if err != nil {
		return fmt.Errorf("agent analyze: %w", err)
	}

	if proposal == "" {
		return fmt.Errorf("agent could not find a solution")
	}

	if approval(proposal) {
		cmd := exec.Command("sh", "-c", proposal)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		fmt.Printf("\nExecuting: %s\n", proposal)

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("execution failed: %w", err)
		}
		return nil
	}

	return fmt.Errorf("user denied fix")
}
