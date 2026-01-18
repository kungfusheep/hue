package tui

import (
	tea "github.com/charmbracelet/bubbletea"
)

// Run starts the TUI with the given action handler.
func Run(handler ActionHandler) error {
	model := NewModel(handler)
	p := tea.NewProgram(model, tea.WithAltScreen())
	_, err := p.Run()
	return err
}
