package infra

import (
	"fmt"
	"os/exec"
	"strings"
)

type GitStatus struct {
	IsRepo    bool
	Branch    string
	Ahead     int
	Behind    int
	Added     int
	Modified  int
	Deleted   int
	Untracked int
}

func GetGitStatus() GitStatus {
	status := GitStatus{}

	cmd := exec.Command("git", "rev-parse", "--is-inside-work-tree")
	if err := cmd.Run(); err != nil {
		return status
	}
	status.IsRepo = true

	cmd = exec.Command("git", "branch", "--show-current")
	if out, err := cmd.Output(); err == nil {
		status.Branch = strings.TrimSpace(string(out))
	}

	cmd = exec.Command("git", "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if out, err := cmd.Output(); err == nil {
		parts := strings.Fields(string(out))
		if len(parts) >= 2 {
			fmt.Sscanf(parts[0], "%d", &status.Ahead)
			fmt.Sscanf(parts[1], "%d", &status.Behind)
		}
	}

	cmd = exec.Command("git", "status", "--porcelain")
	if out, err := cmd.Output(); err == nil {
		lines := strings.Split(string(out), "\n")
		for _, line := range lines {
			if len(line) < 2 {
				continue
			}
			xy := line[:2]
			switch {
			case xy[0] == 'A' || xy[1] == 'A':
				status.Added++
			case xy[0] == 'M' || xy[1] == 'M':
				status.Modified++
			case xy[0] == 'D' || xy[1] == 'D':
				status.Deleted++
			case xy[0] == '?' && xy[1] == '?':
				status.Untracked++
			}
		}
	}

	return status
}

func (g GitStatus) Summary() string {
	if !g.IsRepo {
		return ""
	}

	parts := []string{g.Branch}

	if g.Ahead > 0 || g.Behind > 0 {
		parts = append(parts, fmt.Sprintf("↑%d↓%d", g.Ahead, g.Behind))
	}

	changes := g.Added + g.Modified + g.Deleted + g.Untracked
	if changes > 0 {
		parts = append(parts, fmt.Sprintf("•%d", changes))
	}

	return strings.Join(parts, " ")
}
