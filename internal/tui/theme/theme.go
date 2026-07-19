package theme

import "github.com/charmbracelet/lipgloss"

var (
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

var (
	LogError = Red
	LogWarn  = Yellow
	LogInfo  = Blue
	LogDebug = Overlay0

	StatusBar = lipgloss.NewStyle().
			Background(Surface0).
			Foreground(Text).
			Padding(0, 1)

	Tab = lipgloss.NewStyle().
		Padding(0, 2).
		Foreground(Overlay0)

	ActiveTab = lipgloss.NewStyle().
			Padding(0, 2).
			Foreground(Mauve).
			Bold(true).
			Background(Surface0)
)
