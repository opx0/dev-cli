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
	Peach    = lipgloss.Color("#fab387")
	Teal     = lipgloss.Color("#94e2d5")
	Pink     = lipgloss.Color("#f5c2e7")
	Overlay0 = lipgloss.Color("#6c7086")
	Overlay1 = lipgloss.Color("#7f849c")
	Surface0 = lipgloss.Color("#313244")
	Surface1 = lipgloss.Color("#45475a")
	Surface2 = lipgloss.Color("#585b70")
	Lavender = lipgloss.Color("#b4befe")
	Text     = lipgloss.Color("#cdd6f4")
	Subtext0 = lipgloss.Color("#a6adc8")
)

var (
	LogError = Red
	LogWarn  = Yellow
	LogInfo  = Blue
	LogDebug = Overlay0
)

var (
	StatusRunning = Green
	StatusStopped = Red
	StatusPending = Yellow
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
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Green).
	Padding(0, 1)

var Header = lipgloss.NewStyle().
	Bold(true).
	Foreground(Lavender)

var SubHeader = lipgloss.NewStyle().
	Foreground(Subtext0)

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
	Padding(0, 2).
	Foreground(Overlay0)

var ActiveTab = lipgloss.NewStyle().
	Padding(0, 2).
	Foreground(Mauve).
	Bold(true).
	Background(Surface0)

var ModeIndicator = lipgloss.NewStyle().
	Background(Green).
	Foreground(Crust).
	Padding(0, 1).
	Bold(true)

var NormalModeIndicator = lipgloss.NewStyle().
	Background(Mauve).
	Foreground(Crust).
	Padding(0, 1).
	Bold(true)

var Selection = lipgloss.NewStyle().
	Background(Surface1).
	Foreground(Text).
	Bold(true)

var Prompt = lipgloss.NewStyle().
	Foreground(Green).
	Bold(true)

var Badge = lipgloss.NewStyle().
	Foreground(Text).
	Background(Surface0).
	Padding(0, 1)

var BadgeSuccess = lipgloss.NewStyle().
	Foreground(Crust).
	Background(Green).
	Padding(0, 1)

var BadgeError = lipgloss.NewStyle().
	Foreground(Crust).
	Background(Red).
	Padding(0, 1)

var BadgeWarn = lipgloss.NewStyle().
	Foreground(Crust).
	Background(Yellow).
	Padding(0, 1)

var BadgeInfo = lipgloss.NewStyle().
	Foreground(Crust).
	Background(Blue).
	Padding(0, 1)

var HeaderWidget = lipgloss.NewStyle().
	Foreground(Overlay0).
	Padding(0, 1)

var HeaderWidgetActive = lipgloss.NewStyle().
	Foreground(Text).
	Background(Surface0).
	Padding(0, 1)

var ActionMenu = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(Mauve).
	Background(Base).
	Padding(0, 1)

var ActionMenuItem = lipgloss.NewStyle().
	Foreground(Text).
	Padding(0, 1)

var ActionMenuItemSelected = lipgloss.NewStyle().
	Foreground(Mauve).
	Background(Surface1).
	Padding(0, 1).
	Bold(true)

var ActionMenuKey = lipgloss.NewStyle().
	Foreground(Mauve).
	Bold(true)

var UserBubble = lipgloss.NewStyle().
	Foreground(Text).
	Background(Surface1).
	Padding(0, 1).
	MarginLeft(4)

var AssistantBubble = lipgloss.NewStyle().
	Foreground(Text).
	Background(Surface0).
	Padding(0, 1).
	MarginRight(4)

var CodeBlock = lipgloss.NewStyle().
	Foreground(Text).
	Background(Mantle).
	Padding(0, 1)

var OutputBlock = lipgloss.NewStyle().
	Border(lipgloss.Border{Left: "│"}).
	BorderForeground(Surface2).
	PaddingLeft(1)

var OutputBlockSuccess = lipgloss.NewStyle().
	Border(lipgloss.Border{Left: "│"}).
	BorderForeground(Green).
	PaddingLeft(1)

var OutputBlockError = lipgloss.NewStyle().
	Border(lipgloss.Border{Left: "│"}).
	BorderForeground(Red).
	PaddingLeft(1)

var OutputBlockSelected = lipgloss.NewStyle().
	Border(lipgloss.Border{Left: "▐"}).
	BorderForeground(Mauve).
	PaddingLeft(1).
	Background(Surface0)

var ContextBadge = lipgloss.NewStyle().
	Foreground(Overlay0).
	Italic(true)

var SparklineBar = lipgloss.NewStyle().
	Foreground(Teal)

var SparklineBarHigh = lipgloss.NewStyle().
	Foreground(Yellow)

var SparklineBarCritical = lipgloss.NewStyle().
	Foreground(Red)
