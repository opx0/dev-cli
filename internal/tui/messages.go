package tui

import (
	"database/sql"

	"dev-cli/internal/infra"
	"dev-cli/internal/storage"
)

type dockerHealthMsg struct {
	health infra.DockerHealth
}

type gpuStatsMsg struct {
	stats infra.GPUStats
}

type serviceHealthMsg struct {
	services []infra.ServiceStatus
}

type historyLoadedMsg struct {
	history []storage.HistoryItem
	db      *sql.DB
	err     error
}
