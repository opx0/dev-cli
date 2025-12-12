package tui

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

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

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

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

		if m.mode == ModeInsert {

			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				m.input.Blur()
				return m, nil

			case "enter":
				val := m.input.Value()
				m.input.SetValue("")

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

			case "j", "tab":
				m.focus = (m.focus + 1) % 3
			case "k", "shift+tab":
				if m.focus == 0 {
					m.focus = 2
				} else {
					m.focus--
				}

			case "1":
				m.focus = FocusStatus
			case "2":
				m.focus = FocusKeybinds
			case "3":
				m.focus = FocusTerminal

			case "up":
				if m.focus == FocusTerminal {
					m.viewport.ScrollUp(1)
				}
			case "down":
				if m.focus == FocusTerminal {
					m.viewport.ScrollDown(1)
				}

			case "i", "enter":
				if m.focus == FocusTerminal {
					m.mode = ModeInsert
					m.input.Focus()
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
		os.Chdir(path)
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
