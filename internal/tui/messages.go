package tui

import (
	"dev-cli/internal/infra"
	"dev-cli/internal/storage"
)

type dockerHealthMsg struct{ health infra.DockerHealth }

type containerLogsMsg struct {
	containerID string
	lines       []string
	err         error
}

type historyLoadedMsg struct {
	history []storage.HistoryItem
	err     error
}
