package tui

import "github.com/charmbracelet/lipgloss"

// Styles holds all the lipgloss styles for the TUI.
// Centralised here so we can easily theme and adjust.
type Styles struct {
	// Layout
	App       lipgloss.Style
	Header    lipgloss.Style
	Section   lipgloss.Style
	StatusBar lipgloss.Style

	// Query input
	QueryLabel  lipgloss.Style
	QueryInput  lipgloss.Style
	QueryCursor lipgloss.Style

	// Light list
	LightSelected   lipgloss.Style
	LightUnselected lipgloss.Style
	LightOn         lipgloss.Style
	LightOff        lipgloss.Style
	LightName       lipgloss.Style
	LightType       lipgloss.Style
	LightBrightness lipgloss.Style

	// General
	Title    lipgloss.Style
	Subtle   lipgloss.Style
	Accent   lipgloss.Style
	Error    lipgloss.Style
	Success  lipgloss.Style
	Help     lipgloss.Style
	HelpKey  lipgloss.Style
	HelpDesc lipgloss.Style
}

// DefaultStyles returns the default style configuration.
func DefaultStyles() Styles {
	accent := lipgloss.Color("69")  // Nice blue
	subtle := lipgloss.Color("241") // Gray
	success := lipgloss.Color("42") // Green
	errColor := lipgloss.Color("196") // Red

	return Styles{
		App: lipgloss.NewStyle().
			Padding(1, 2),

		Header: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent).
			MarginBottom(1),

		Section: lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(subtle).
			Padding(0, 1),

		StatusBar: lipgloss.NewStyle().
			Foreground(subtle).
			MarginTop(1),

		QueryLabel: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		QueryInput: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),

		QueryCursor: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		LightSelected: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		LightUnselected: lipgloss.NewStyle().
			Foreground(lipgloss.Color("255")),

		LightOn: lipgloss.NewStyle().
			Foreground(success).
			Bold(true),

		LightOff: lipgloss.NewStyle().
			Foreground(subtle),

		LightName: lipgloss.NewStyle().
			Width(20),

		LightType: lipgloss.NewStyle().
			Width(12).
			Foreground(subtle),

		LightBrightness: lipgloss.NewStyle().
			Width(6).
			Align(lipgloss.Right),

		Title: lipgloss.NewStyle().
			Bold(true).
			Foreground(accent),

		Subtle: lipgloss.NewStyle().
			Foreground(subtle),

		Accent: lipgloss.NewStyle().
			Foreground(accent),

		Error: lipgloss.NewStyle().
			Foreground(errColor),

		Success: lipgloss.NewStyle().
			Foreground(success),

		Help: lipgloss.NewStyle().
			Foreground(subtle),

		HelpKey: lipgloss.NewStyle().
			Foreground(accent).
			Bold(true),

		HelpDesc: lipgloss.NewStyle().
			Foreground(subtle),
	}
}

// ColorSwatch returns a colored block representing a light's color.
func ColorSwatch(hex string) string {
	if hex == "" {
		return "    "
	}
	return lipgloss.NewStyle().
		Background(lipgloss.Color(hex)).
		Render("    ")
}
