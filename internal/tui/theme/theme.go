package theme

import "github.com/charmbracelet/lipgloss"

var (
	Crust    = lipgloss.Color("#11111b")
	Base     = lipgloss.Color("#1e1e2e")
	Mantle   = lipgloss.Color("#181825")
	Mauve    = lipgloss.Color("#cba6f7")
	Red      = lipgloss.Color("#f38ba8")
	Green    = lipgloss.Color("#a6e3a1")
	Yellow   = lipgloss.Color("#f9e2af")
	Blue     = lipgloss.Color("#89b4fa")
	Overlay0 = lipgloss.Color("#6c7086")
	Surface0 = lipgloss.Color("#313244")
	Surface1 = lipgloss.Color("#45475a")
	Surface2 = lipgloss.Color("#585b70")
	Lavender = lipgloss.Color("#b4befe")
	Text     = lipgloss.Color("#cdd6f4")
)

var Title = lipgloss.NewStyle().
	Bold(true).
	Foreground(Crust).
	Background(Mauve).
	Padding(0, 1)

var Panel = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Surface2).
	Padding(0, 1)

var FocusedPanel = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Mauve).
	Padding(0, 1)

var InsertModePanel = lipgloss.NewStyle().
	Border(lipgloss.ThickBorder()).
	BorderForeground(Green).
	Padding(0, 1)

var Header = lipgloss.NewStyle().
	Bold(true).
	Foreground(Lavender)

var Running = lipgloss.NewStyle().
	Foreground(Green)

var Stopped = lipgloss.NewStyle().
	Foreground(Red)

var Dim = lipgloss.NewStyle().
	Foreground(Overlay0)

var Key = lipgloss.NewStyle().
	Foreground(Mauve).
	Bold(true)

var Desc = lipgloss.NewStyle().
	Foreground(Overlay0)

var StatusBar = lipgloss.NewStyle().
	Background(Surface0).
	Foreground(Text).
	Padding(0, 1)

var StatusKey = lipgloss.NewStyle().
	Background(Surface0).
	Foreground(Mauve).
	Bold(true)

var StatusDesc = lipgloss.NewStyle().
	Background(Surface0).
	Foreground(Overlay0)

var Tab = lipgloss.NewStyle().
	Padding(0, 1).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Surface2)

var ActiveTab = lipgloss.NewStyle().
	Padding(0, 1).
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Mauve).
	Foreground(Mauve).
	Bold(true)

var ModeIndicator = lipgloss.NewStyle().
	Background(Green).
	Foreground(Crust).
	Padding(0, 1).
	Bold(true)

var Selection = lipgloss.NewStyle().
	Background(Surface0).
	Foreground(Text).
	Bold(true)

var Prompt = lipgloss.NewStyle().
	Foreground(Green)
