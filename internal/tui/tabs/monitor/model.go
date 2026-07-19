package monitor

import "dev-cli/internal/infra"

type Model struct {
	width             int
	height            int
	services          []infra.ContainerInfo
	selected          int
	logLines          []string
	logLevelFilter    string
	dockerUnavailable string
}

func New() Model { return Model{} }

func (m Model) SetSize(width, height int) Model {
	m.width, m.height = width, height
	return m
}

func (m Model) SetDockerHealth(health infra.DockerHealth) Model {
	if health.Available {
		m.dockerUnavailable = ""
	} else if health.Error != nil {
		m.dockerUnavailable = health.Error.Error()
	} else {
		m.dockerUnavailable = "Docker is unavailable"
	}
	return m
}

func (m Model) SetServices(services []infra.ContainerInfo) Model {
	m.services = services
	if len(services) == 0 {
		m.selected = 0
	} else if m.selected >= len(services) {
		m.selected = len(services) - 1
	}
	return m
}

func (m Model) SetLogLines(lines []string) Model {
	m.logLines = append([]string(nil), lines...)
	return m
}

func (m Model) SelectedService() *infra.ContainerInfo {
	if m.selected < 0 || m.selected >= len(m.services) {
		return nil
	}
	service := m.services[m.selected]
	return &service
}

func (m Model) SelectedIndex() int { return m.selected }
