package components

import (
	"fmt"
	"strings"

	"dev-cli/internal/tui/theme"

	"github.com/charmbracelet/lipgloss"
)

// HeaderWidget renders a compact status widget for the header bar
type HeaderWidget struct {
	Icon    string
	Label   string
	Value   string
	Active  bool
	Success bool
	Error   bool
}

func NewHeaderWidget(icon, label, value string) HeaderWidget {
	return HeaderWidget{
		Icon:  icon,
		Label: label,
		Value: value,
	}
}

func (w HeaderWidget) SetActive(active bool) HeaderWidget {
	w.Active = active
	return w
}

func (w HeaderWidget) SetSuccess(success bool) HeaderWidget {
	w.Success = success
	return w
}

func (w HeaderWidget) SetError(err bool) HeaderWidget {
	w.Error = err
	return w
}

func (w HeaderWidget) Render() string {
	var style lipgloss.Style
	if w.Active {
		style = theme.HeaderWidgetActive
	} else {
		style = theme.HeaderWidget
	}

	icon := w.Icon
	if w.Success {
		icon = lipgloss.NewStyle().Foreground(theme.Green).Render(icon)
	} else if w.Error {
		icon = lipgloss.NewStyle().Foreground(theme.Red).Render(icon)
	}

	if w.Value != "" {
		return style.Render(fmt.Sprintf("%s %s", icon, w.Value))
	}
	return style.Render(fmt.Sprintf("%s %s", icon, w.Label))
}

// HeaderWidgetBar renders multiple widgets in a horizontal bar
type HeaderWidgetBar struct {
	Widgets []HeaderWidget
	Width   int
}

func NewHeaderWidgetBar(widgets ...HeaderWidget) HeaderWidgetBar {
	return HeaderWidgetBar{Widgets: widgets}
}

func (b HeaderWidgetBar) SetWidth(w int) HeaderWidgetBar {
	b.Width = w
	return b
}

func (b HeaderWidgetBar) Render() string {
	var rendered []string
	for _, w := range b.Widgets {
		rendered = append(rendered, w.Render())
	}

	separator := lipgloss.NewStyle().Foreground(theme.Surface2).Render(" │ ")
	return strings.Join(rendered, separator)
}

// Badge renders a notification or status badge
type Badge struct {
	Label string
	Count int
	Type  string // "success", "error", "warn", "info", ""
}

func NewBadge(label string) Badge {
	return Badge{Label: label}
}

func (b Badge) SetCount(count int) Badge {
	b.Count = count
	return b
}

func (b Badge) SetType(t string) Badge {
	b.Type = t
	return b
}

func (b Badge) Render() string {
	var style lipgloss.Style
	switch b.Type {
	case "success":
		style = theme.BadgeSuccess
	case "error":
		style = theme.BadgeError
	case "warn":
		style = theme.BadgeWarn
	case "info":
		style = theme.BadgeInfo
	default:
		style = theme.Badge
	}

	if b.Count > 0 {
		return style.Render(fmt.Sprintf("%s %d", b.Label, b.Count))
	}
	return style.Render(b.Label)
}

// ActionMenu renders a popup action menu
type ActionMenu struct {
	Title    string
	Items    []ActionMenuItem
	Selected int
	Width    int
}

type ActionMenuItem struct {
	Key   string
	Label string
}

func NewActionMenu(title string, items ...ActionMenuItem) ActionMenu {
	return ActionMenu{
		Title: title,
		Items: items,
		Width: 24,
	}
}

func (m ActionMenu) SetWidth(w int) ActionMenu {
	m.Width = w
	return m
}

func (m ActionMenu) SetSelected(s int) ActionMenu {
	if s >= 0 && s < len(m.Items) {
		m.Selected = s
	}
	return m
}

func (m ActionMenu) Render() string {
	var content strings.Builder

	titleStyle := lipgloss.NewStyle().
		Foreground(theme.Lavender).
		Bold(true)
	content.WriteString(titleStyle.Render(m.Title) + "\n")

	for i, item := range m.Items {
		var itemStyle lipgloss.Style
		if i == m.Selected {
			itemStyle = theme.ActionMenuItemSelected
		} else {
			itemStyle = theme.ActionMenuItem
		}

		key := theme.ActionMenuKey.Render("[" + item.Key + "]")
		label := itemStyle.Render(" " + item.Label)

		content.WriteString(key + label + "\n")
	}

	return theme.ActionMenu.Width(m.Width).Render(content.String())
}

// Sparkline renders a mini bar chart
type Sparkline struct {
	Values    []int
	Max       int
	Width     int
	Height    int
	ShowValue bool
}

func NewSparkline(values []int, max int) Sparkline {
	return Sparkline{
		Values: values,
		Max:    max,
		Width:  20,
		Height: 1,
	}
}

func (s Sparkline) SetWidth(w int) Sparkline {
	s.Width = w
	return s
}

func (s Sparkline) SetShowValue(show bool) Sparkline {
	s.ShowValue = show
	return s
}

func (s Sparkline) Render() string {
	if len(s.Values) == 0 {
		return strings.Repeat("░", s.Width)
	}

	// Take last N values that fit in width
	values := s.Values
	if len(values) > s.Width {
		values = values[len(values)-s.Width:]
	}

	bars := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}

	var result strings.Builder
	for _, v := range values {
		if s.Max == 0 {
			result.WriteRune(bars[0])
			continue
		}

		// Normalize to 0-7 range
		normalized := (v * 7) / s.Max
		if normalized > 7 {
			normalized = 7
		}
		if normalized < 0 {
			normalized = 0
		}

		// Color based on value
		var style lipgloss.Style
		ratio := float64(v) / float64(s.Max)
		if ratio > 0.8 {
			style = theme.SparklineBarCritical
		} else if ratio > 0.5 {
			style = theme.SparklineBarHigh
		} else {
			style = theme.SparklineBar
		}

		result.WriteString(style.Render(string(bars[normalized])))
	}

	// Pad with empty if needed
	for i := len(values); i < s.Width; i++ {
		result.WriteString(lipgloss.NewStyle().Foreground(theme.Surface1).Render("░"))
	}

	sparkline := result.String()

	if s.ShowValue && len(s.Values) > 0 {
		lastVal := s.Values[len(s.Values)-1]
		pct := (lastVal * 100) / s.Max
		pctStr := fmt.Sprintf(" %d%%", pct)

		var pctStyle lipgloss.Style
		if pct > 80 {
			pctStyle = lipgloss.NewStyle().Foreground(theme.Red).Bold(true)
		} else {
			pctStyle = lipgloss.NewStyle().Foreground(theme.Text)
		}

		return sparkline + pctStyle.Render(pctStr)
	}

	return sparkline
}

// ProgressBar renders a horizontal progress bar
type ProgressBar struct {
	Value   int
	Max     int
	Width   int
	Label   string
	ShowPct bool
}

func NewProgressBar(value, max int) ProgressBar {
	return ProgressBar{
		Value:   value,
		Max:     max,
		Width:   20,
		ShowPct: true,
	}
}

func (p ProgressBar) SetWidth(w int) ProgressBar {
	p.Width = w
	return p
}

func (p ProgressBar) SetLabel(l string) ProgressBar {
	p.Label = l
	return p
}

func (p ProgressBar) Render() string {
	if p.Max == 0 {
		return strings.Repeat("░", p.Width)
	}

	filled := (p.Value * p.Width) / p.Max
	if filled > p.Width {
		filled = p.Width
	}

	var bar strings.Builder

	for i := 0; i < p.Width; i++ {
		var style lipgloss.Style
		ratio := float64(i) / float64(p.Width)

		if ratio < 0.5 {
			style = lipgloss.NewStyle().Foreground(theme.Green)
		} else if ratio < 0.75 {
			style = lipgloss.NewStyle().Foreground(theme.Yellow)
		} else {
			style = lipgloss.NewStyle().Foreground(theme.Red)
		}

		if i < filled {
			bar.WriteString(style.Render("█"))
		} else {
			bar.WriteString(lipgloss.NewStyle().Foreground(theme.Surface1).Render("░"))
		}
	}

	result := bar.String()

	if p.ShowPct {
		pct := (p.Value * 100) / p.Max
		pctStr := fmt.Sprintf(" %d%%", pct)

		var pctStyle lipgloss.Style
		if pct > 80 {
			pctStyle = lipgloss.NewStyle().Foreground(theme.Red).Bold(true)
		} else {
			pctStyle = lipgloss.NewStyle().Foreground(theme.Overlay0)
		}

		result += pctStyle.Render(pctStr)
	}

	if p.Label != "" {
		labelStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		result = labelStyle.Render(p.Label+" ") + result
	}

	return result
}

// OutputBlock represents a single command output block (warp-style)
type OutputBlock struct {
	Command   string
	Output    string
	ExitCode  int
	Timestamp string
	Selected  bool
	Folded    bool
}

func NewOutputBlock(command string) OutputBlock {
	return OutputBlock{
		Command: command,
	}
}

func (b OutputBlock) SetOutput(output string) OutputBlock {
	b.Output = output
	return b
}

func (b OutputBlock) SetExitCode(code int) OutputBlock {
	b.ExitCode = code
	return b
}

func (b OutputBlock) SetTimestamp(ts string) OutputBlock {
	b.Timestamp = ts
	return b
}

func (b OutputBlock) SetSelected(sel bool) OutputBlock {
	b.Selected = sel
	return b
}

func (b OutputBlock) SetFolded(folded bool) OutputBlock {
	b.Folded = folded
	return b
}

func (b OutputBlock) Render(width int) string {
	var style lipgloss.Style
	if b.Selected {
		style = theme.OutputBlockSelected
	} else if b.ExitCode != 0 {
		style = theme.OutputBlockError
	} else if b.Output != "" {
		style = theme.OutputBlockSuccess
	} else {
		style = theme.OutputBlock
	}

	// Command header
	cmdStyle := theme.Prompt
	tsStyle := theme.Dim

	var header strings.Builder
	header.WriteString(cmdStyle.Render("❯ "))
	header.WriteString(lipgloss.NewStyle().Foreground(theme.Text).Bold(true).Render(b.Command))

	if b.Timestamp != "" {
		header.WriteString("  ")
		header.WriteString(tsStyle.Render(b.Timestamp))
	}

	if b.ExitCode != 0 {
		exitStyle := lipgloss.NewStyle().Foreground(theme.Red).Bold(true)
		header.WriteString("  ")
		header.WriteString(exitStyle.Render(fmt.Sprintf("✗ %d", b.ExitCode)))
	}

	// Fold indicator
	if b.Folded {
		foldStyle := lipgloss.NewStyle().Foreground(theme.Overlay0)
		header.WriteString("  ")
		header.WriteString(foldStyle.Render("▸ (folded)"))
	}

	var content strings.Builder
	content.WriteString(header.String())

	if !b.Folded && b.Output != "" {
		content.WriteString("\n")
		// Limit output lines
		lines := strings.Split(b.Output, "\n")
		maxLines := 15
		if len(lines) > maxLines {
			for _, line := range lines[:maxLines] {
				content.WriteString(line + "\n")
			}
			more := len(lines) - maxLines
			content.WriteString(theme.Dim.Render(fmt.Sprintf("... %d more lines", more)))
		} else {
			content.WriteString(b.Output)
		}
	}

	return style.Width(width - 2).Render(content.String())
}

// LogLine renders a log line with level highlighting
type LogLine struct {
	Content string
	Level   string // "ERROR", "WARN", "INFO", "DEBUG"
}

func NewLogLine(content string) LogLine {
	line := LogLine{Content: content}

	// Auto-detect level
	upperContent := strings.ToUpper(content)
	if strings.Contains(upperContent, "ERROR") || strings.Contains(upperContent, "ERR") {
		line.Level = "ERROR"
	} else if strings.Contains(upperContent, "WARN") {
		line.Level = "WARN"
	} else if strings.Contains(upperContent, "INFO") {
		line.Level = "INFO"
	} else if strings.Contains(upperContent, "DEBUG") || strings.Contains(upperContent, "TRACE") {
		line.Level = "DEBUG"
	}

	return line
}

func (l LogLine) Render() string {
	var style lipgloss.Style
	switch l.Level {
	case "ERROR":
		style = lipgloss.NewStyle().Foreground(theme.LogError)
	case "WARN":
		style = lipgloss.NewStyle().Foreground(theme.LogWarn)
	case "INFO":
		style = lipgloss.NewStyle().Foreground(theme.LogInfo)
	case "DEBUG":
		style = lipgloss.NewStyle().Foreground(theme.LogDebug)
	default:
		style = lipgloss.NewStyle().Foreground(theme.Text)
	}

	return style.Render(l.Content)
}

// ContextBadge shows AI context awareness
type ContextBadge struct {
	Commands   int
	Containers int
	Errors     int
}

func NewContextBadge() ContextBadge {
	return ContextBadge{}
}

func (c ContextBadge) SetCommands(n int) ContextBadge {
	c.Commands = n
	return c
}

func (c ContextBadge) SetContainers(n int) ContextBadge {
	c.Containers = n
	return c
}

func (c ContextBadge) SetErrors(n int) ContextBadge {
	c.Errors = n
	return c
}

func (c ContextBadge) Render() string {
	var parts []string

	if c.Commands > 0 {
		parts = append(parts, fmt.Sprintf("📋 %d commands", c.Commands))
	}
	if c.Containers > 0 {
		parts = append(parts, fmt.Sprintf("🐳 %d containers", c.Containers))
	}
	if c.Errors > 0 {
		errStyle := lipgloss.NewStyle().Foreground(theme.Red)
		parts = append(parts, errStyle.Render(fmt.Sprintf("🔴 %d errors", c.Errors)))
	}

	if len(parts) == 0 {
		return ""
	}

	content := "Context: " + strings.Join(parts, " • ")
	return theme.ContextBadge.Render(content)
}
