package tui

import (
	"database/sql"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"dev-cli/internal/storage"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type historyLoadedMsg struct {
	history []storage.HistoryItem
	db      *sql.DB
	err     error
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 30

		m.viewport.Width = msg.Width - 30
		m.viewport.Height = m.height - 10

	case dockerHealthMsg:
		m.dockerHealth = msg.health
		if msg.health.Available {
			m.state = StateMain
		}

	case gpuStatsMsg:
		m.gpuStats = msg.stats

	case serviceHealthMsg:
		m.serviceHealth = msg.services

	case historyLoadedMsg:
		if msg.err != nil {
			// Handle error, maybe show in status or log
			// For now just ignore or log to viewport if debug
		} else {
			m.db = msg.db
			m.commandHistory = msg.history
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd, checkGPUStats, checkDockerHealth, checkServices)

	case commandOutputMsg:
		m.viewport.SetContent(m.viewport.View() + string(msg))
		m.viewport.GotoBottom()

	case errMsg:
		m.viewport.SetContent(m.viewport.View() + fmt.Sprintf("Error: %v\n", msg))
		m.viewport.GotoBottom()

	case clearViewportMsg:
		m.viewport.SetContent("")

	case tea.KeyMsg:

		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// Global Tab Switching
		switch msg.String() {
		case "1":
			m.activeTab = TabDashboard
			m.viewport.SetContent("")
		case "2":
			m.activeTab = TabMonitor
			m.viewport.SetContent("")
		case "3":
			m.activeTab = TabAssist
			m.viewport.SetContent(strings.Join(m.chatHistory, "\n"))
		case "4":
			m.activeTab = TabHistory
			m.viewport.SetContent("")
		}

		if m.mode == ModeInsert {

			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				m.input.Blur()
				return m, nil

			case "enter":
				val := m.input.Value()
				m.input.SetValue("")

				if m.activeTab == TabAssist {
					m.chatHistory = append(m.chatHistory, "> "+val)
					m.chatHistory = append(m.chatHistory, "AI: Valid point. (Real AI integration coming soon)")
					m.viewport.SetContent(strings.Join(m.chatHistory, "\n"))
					m.viewport.GotoBottom()
					return m, nil
				}

				return m, executeCommand(val)
			}

			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

		} else {
			switch msg.String() {

			case "q":
				m.quitting = true
				return m, tea.Quit

			case "tab":
				m.focus = (m.focus + 1) % 2 // Toggle between 0 (Sidebar) and 1 (Main)

			// Navigation
			case "j", "down":
				if m.focus == FocusSidebar {
					if m.activeTab == TabMonitor {
						if len(m.dockerHealth.Containers) > 0 {
							m.monitorCursor = (m.monitorCursor + 1) % len(m.dockerHealth.Containers)
						}
					} else if m.activeTab == TabHistory {
						if len(m.commandHistory) > 0 {
							m.historyCursor = (m.historyCursor + 1) % len(m.commandHistory)
						}
					}
				} else {
					m.viewport.ScrollDown(1)
				}

			case "k", "up":
				if m.focus == FocusSidebar {
					if m.activeTab == TabMonitor {
						if len(m.dockerHealth.Containers) > 0 {
							m.monitorCursor--
							if m.monitorCursor < 0 {
								m.monitorCursor = len(m.dockerHealth.Containers) - 1
							}
						}
					} else if m.activeTab == TabHistory {
						if len(m.commandHistory) > 0 {
							m.historyCursor--
							if m.historyCursor < 0 {
								m.historyCursor = len(m.commandHistory) - 1
							}
						}
					}
				} else {
					m.viewport.ScrollUp(1)
				}

			case "enter":
				if m.focus == FocusMain {
					m.mode = ModeInsert
					m.input.Focus()
				}

			case "i":
				m.mode = ModeInsert
				m.input.Focus()

			case "ctrl+t":
				if m.activeTab == TabAssist {
					if m.aiMode == "local" {
						m.aiMode = "cloud"
					} else {
						m.aiMode = "local"
					}
				}

			case "?":
				if m.activeTab == TabMonitor {
					// TODO: Helper / RCA trigger
				}
			}
		}
	}

	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

func executeCommand(input string) tea.Cmd {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	forceInteractive := false
	if strings.HasPrefix(input, "!") {
		forceInteractive = true
		input = strings.TrimPrefix(input, "!")
		input = strings.TrimSpace(input)
		if input == "" {
			return nil
		}
	}

	args := strings.Fields(input)
	cmdName := args[0]

	if cmdName == "cd" {
		path := ""
		if len(args) > 1 {
			path = args[1]
		} else {
			path, _ = os.UserHomeDir()
		}
		if err := os.Chdir(path); err != nil {
			return func() tea.Msg { return commandOutputMsg(fmt.Sprintf("> cd %s\nError: %v\n", path, err)) }
		}
		return func() tea.Msg { return commandOutputMsg(fmt.Sprintf("> cd %s\n", path)) }
	}

	if cmdName == "clear" {
		return func() tea.Msg { return clearViewportMsg{} }
	}

	c := exec.Command(cmdName, args[1:]...)

	interactiveCommands := map[string]bool{

		"vim": true, "vi": true, "nvim": true, "nano": true, "emacs": true, "micro": true,

		"htop": true, "top": true, "btop": true, "gtop": true,

		"bash": true, "sh": true, "zsh": true, "fish": true,

		"python": true, "python3": true, "node": true, "irb": true, "ghci": true,

		"less": true, "more": true, "man": true,

		"ssh": true,
	}

	isInteractive := forceInteractive || interactiveCommands[cmdName]

	if cmdName == "docker" || cmdName == "podman" || cmdName == "kubectl" {
		if len(args) > 1 && (args[1] == "exec" || args[1] == "run") {

			for _, arg := range args {
				if arg == "-it" || arg == "-ti" || arg == "-i" || arg == "-t" {
					isInteractive = true
					break
				}
			}
		}
	}

	if isInteractive {
		return tea.ExecProcess(c, func(err error) tea.Msg {
			if err != nil {
				return errMsg(err)
			}
			return commandOutputMsg(fmt.Sprintf("\n> %s (completed)\n", input))
		})
	}

	return func() tea.Msg {
		output, err := c.CombinedOutput()
		if err != nil {
			return commandOutputMsg(fmt.Sprintf("> %s\nError: %s\n", input, err.Error()))
		}
		return commandOutputMsg(fmt.Sprintf("> %s\n%s\n", input, string(output)))
	}
}
