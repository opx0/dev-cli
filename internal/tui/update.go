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

	// Handle Window Resize
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.input.Width = msg.Width - 30
		// Resize viewport for terminal history
		m.viewport.Width = msg.Width - 30
		m.viewport.Height = m.height - 10

	// Handle Infrastructure Messages
	case dockerHealthMsg:
		m.dockerHealth = msg.health
		if msg.health.Available {
			m.state = StateMain
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		cmds = append(cmds, cmd)

	// Handle Command Output
	case commandOutputMsg:
		m.viewport.SetContent(m.viewport.View() + string(msg))
		m.viewport.GotoBottom()

	case errMsg:
		m.viewport.SetContent(m.viewport.View() + fmt.Sprintf("Error: %v\n", msg))
		m.viewport.GotoBottom()

	case clearViewportMsg:
		m.viewport.SetContent("")

	// --- KEYBOARD INPUT ---
	case tea.KeyMsg:

		// ALWAYS allow Ctrl+C to quit
		if msg.String() == "ctrl+c" {
			m.quitting = true
			return m, tea.Quit
		}

		// LOGIC BRANCH: NORMAL MODE vs INSERT MODE
		if m.mode == ModeInsert {
			// --- INSERT MODE ---
			switch msg.String() {
			case "esc":
				m.mode = ModeNormal
				m.input.Blur()
				return m, nil

			case "enter":
				val := m.input.Value()
				m.input.SetValue("")

				// Execute the command
				return m, executeCommand(val)
			}

			// Pass all other keys to the text input
			var cmd tea.Cmd
			m.input, cmd = m.input.Update(msg)
			cmds = append(cmds, cmd)

		} else {
			// --- NORMAL MODE ---
			switch msg.String() {
			// Quit
			case "q":
				m.quitting = true
				return m, tea.Quit

			// Vim Navigation
			case "j", "tab":
				m.focus = (m.focus + 1) % 4
			case "k", "shift+tab":
				if m.focus == 0 {
					m.focus = 3
				} else {
					m.focus--
				}

			// Direct Panel Access
			case "1":
				m.focus = FocusContainers
			case "2":
				m.focus = FocusObservability
			case "3":
				m.focus = FocusKeybinds
			case "4":
				m.focus = FocusTerminal

			// Viewport Scrolling (when terminal focused)
			case "up":
				if m.focus == FocusTerminal {
					m.viewport.LineUp(1)
				}
			case "down":
				if m.focus == FocusTerminal {
					m.viewport.LineDown(1)
				}

			// Enter Insert Mode (only if terminal is focused)
			case "i", "enter":
				if m.focus == FocusTerminal {
					m.mode = ModeInsert
					m.input.Focus()
				}
			}
		}
	}

	// Always update viewport for scrolling
	var vpCmd tea.Cmd
	m.viewport, vpCmd = m.viewport.Update(msg)
	cmds = append(cmds, vpCmd)

	return m, tea.Batch(cmds...)
}

// executeCommand runs a command.
// If it's interactive (like 'vim', 'top'), it suspends the TUI.
// If it's standard, it captures output.
// Prefix command with '!' to force interactive mode.
func executeCommand(input string) tea.Cmd {
	if strings.TrimSpace(input) == "" {
		return nil
	}

	// Check for forced interactive mode
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

	// Handle special internal commands like 'cd'
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

	// Handle 'clear' - just clear the viewport content
	if cmdName == "clear" {
		return func() tea.Msg { return clearViewportMsg{} }
	}

	// Create the command
	c := exec.Command(cmdName, args[1:]...)

	// Heuristic: Check if the command is likely interactive
	// Note: docker, podman, kubectl only interactive with exec/run -it
	interactiveCommands := map[string]bool{
		// Editors
		"vim": true, "vi": true, "nvim": true, "nano": true, "emacs": true, "micro": true,
		// System monitors
		"htop": true, "top": true, "btop": true, "gtop": true,
		// Shells
		"bash": true, "sh": true, "zsh": true, "fish": true,
		// REPLs
		"python": true, "python3": true, "node": true, "irb": true, "ghci": true,
		// TUI apps
		"less": true, "more": true, "man": true,
		// Remote
		"ssh": true,
	}

	isInteractive := forceInteractive || interactiveCommands[cmdName]

	// Special case: docker/podman/kubectl exec or run with -it flags
	if cmdName == "docker" || cmdName == "podman" || cmdName == "kubectl" {
		if len(args) > 1 && (args[1] == "exec" || args[1] == "run") {
			// Check for -it or -i or -t flags
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

	// Standard execution (Capture output)
	return func() tea.Msg {
		output, err := c.CombinedOutput()
		if err != nil {
			return commandOutputMsg(fmt.Sprintf("> %s\nError: %s\n", input, err.Error()))
		}
		return commandOutputMsg(fmt.Sprintf("> %s\n%s\n", input, string(output)))
	}
}
