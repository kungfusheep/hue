package tui

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// QueryInput is a component for entering and editing light queries.
// It wraps a text input and emits QueryChangedMsg/QuerySubmittedMsg.
type QueryInput struct {
	input   textinput.Model
	styles  Styles
	focused bool
	width   int
}

// NewQueryInput creates a new query input component.
func NewQueryInput(styles Styles) QueryInput {
	ti := textinput.New()
	ti.Placeholder = `@"room:office on:"`
	ti.CharLimit = 256
	ti.Width = 50

	return QueryInput{
		input:   ti,
		styles:  styles,
		focused: false,
		width:   60,
	}
}

// Init initialises the component.
func (q QueryInput) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns commands.
func (q QueryInput) Update(msg tea.Msg) (QueryInput, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !q.focused {
			return q, nil
		}

		switch msg.String() {
		case "enter":
			// Emit submitted message
			return q, func() tea.Msg {
				return QuerySubmittedMsg{Query: q.input.Value()}
			}
		case "esc":
			// Blur and emit focus change
			q.input.Blur()
			q.focused = false
			return q, func() tea.Msg {
				return FocusChangedMsg{Component: "lights"}
			}
		}
	}

	// Update the text input
	var cmd tea.Cmd
	prevValue := q.input.Value()
	q.input, cmd = q.input.Update(msg)
	cmds = append(cmds, cmd)

	// If value changed, emit QueryChangedMsg
	if q.focused && q.input.Value() != prevValue {
		cmds = append(cmds, func() tea.Msg {
			return QueryChangedMsg{Query: q.input.Value()}
		})
	}

	return q, tea.Batch(cmds...)
}

// View renders the component.
func (q QueryInput) View() string {
	label := q.styles.QueryLabel.Render("Query: ")

	input := q.input.View()
	if q.focused {
		input = q.styles.Section.
			BorderForeground(lipgloss.Color("69")).
			Width(q.width).
			Render(input)
	} else {
		input = q.styles.Section.
			Width(q.width).
			Render(input)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, label, input)
}

// Focus sets focus on the input.
func (q *QueryInput) Focus() tea.Cmd {
	q.focused = true
	return q.input.Focus()
}

// Blur removes focus from the input.
func (q *QueryInput) Blur() {
	q.focused = false
	q.input.Blur()
}

// Focused returns whether the input is focused.
func (q QueryInput) Focused() bool {
	return q.focused
}

// Value returns the current query string.
func (q QueryInput) Value() string {
	return q.input.Value()
}

// SetValue sets the query string.
func (q *QueryInput) SetValue(v string) {
	q.input.SetValue(v)
}

// SetWidth sets the width of the input.
func (q *QueryInput) SetWidth(w int) {
	q.width = w
	q.input.Width = w - 4 // Account for padding/border
}
