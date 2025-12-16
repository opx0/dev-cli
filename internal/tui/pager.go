package tui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	pagerTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#11111b")).
			Background(lipgloss.Color("#cba6f7")).
			Padding(0, 1)

	pagerHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#6c7086"))

	pagerBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("#585b70"))
)

type PagerModel struct {
	viewport viewport.Model
	title    string
	content  string
	ready    bool
	width    int
	height   int
}

func NewPager(title, content string) PagerModel {
	return PagerModel{
		title:   title,
		content: content,
	}
}

func (m PagerModel) Init() tea.Cmd {
	return nil
}

func (m PagerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		case "g":
			m.viewport.GotoTop()
		case "G":
			m.viewport.GotoBottom()
		}

	case tea.WindowSizeMsg:
		headerHeight := 3
		footerHeight := 2

		if !m.ready {
			m.width = msg.Width
			m.height = msg.Height

			m.viewport = viewport.New(msg.Width-2, msg.Height-headerHeight-footerHeight)
			m.viewport.SetContent(m.content)
			m.ready = true
		} else {
			m.width = msg.Width
			m.height = msg.Height
			m.viewport.Width = msg.Width - 2
			m.viewport.Height = msg.Height - headerHeight - footerHeight
		}
	}

	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m PagerModel) View() string {
	if !m.ready {
		return "Initializing..."
	}

	title := pagerTitleStyle.Render(m.title)

	scrollPercent := int(m.viewport.ScrollPercent() * 100)
	scrollInfo := pagerHelpStyle.Render(" ↑/↓ j/k scroll • g/G top/bottom • q quit ")
	percent := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#b4befe")).
		Render(fmt.Sprintf(" %d%% ", scrollPercent))

	header := lipgloss.JoinHorizontal(lipgloss.Center, title, "  ", percent)
	footer := scrollInfo

	content := pagerBorderStyle.Width(m.width - 2).Render(m.viewport.View())

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		"",
		content,
		footer,
	)
}

// RunPager starts a full-screen pager with the given content
func RunPager(title, content string) error {
	p := tea.NewProgram(
		NewPager(title, content),
		tea.WithAltScreen(),
	)
	_, err := p.Run()
	return err
}
