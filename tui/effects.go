package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kungfusheep/hue/effects"
)

// EffectItem represents an effect option in the selector.
type EffectItem struct {
	Name        string
	Description string
	Available   bool // Whether selected lights support this effect
}

// EffectSelector is a modal component for selecting light effects.
type EffectSelector struct {
	items    []EffectItem
	cursor   int
	styles   Styles
	keys     KeyMap
	visible  bool
	width    int
	height   int
	lightIDs []string // IDs of lights to apply effect to
}

// NewEffectSelector creates a new effect selector component.
func NewEffectSelector(styles Styles, keys KeyMap) EffectSelector {
	return EffectSelector{
		items:  buildEffectItems(nil),
		cursor: 0,
		styles: styles,
		keys:   keys,
		width:  60,
		height: 15,
	}
}

// buildEffectItems creates the list of effect items.
// If supportedEffects is nil, all effects are marked as available.
func buildEffectItems(supportedEffects []string) []EffectItem {
	allEffects := effects.GetAllEffects()
	items := make([]EffectItem, len(allEffects))

	supportedMap := make(map[string]bool)
	if supportedEffects != nil {
		for _, e := range supportedEffects {
			supportedMap[e] = true
		}
	}

	for i, effect := range allEffects {
		available := true
		if supportedEffects != nil {
			available = supportedMap[effect]
		}

		items[i] = EffectItem{
			Name:        effect,
			Description: effects.GetDescription(effect),
			Available:   available,
		}
	}

	return items
}

// Init initialises the component.
func (e EffectSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns commands.
func (e EffectSelector) Update(msg tea.Msg) (EffectSelector, tea.Cmd) {
	if !e.visible {
		return e, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, e.keys.Up):
			e.moveUp()
		case key.Matches(msg, e.keys.Down):
			e.moveDown()
		case key.Matches(msg, e.keys.Top):
			e.cursor = 0
		case key.Matches(msg, e.keys.Bottom):
			e.cursor = len(e.items) - 1
		case key.Matches(msg, e.keys.Enter):
			return e, e.selectEffect()
		case key.Matches(msg, e.keys.Escape):
			e.visible = false
			return e, func() tea.Msg {
				return EffectSelectorClosedMsg{}
			}
		}
	}

	return e, nil
}

// View renders the component.
func (e EffectSelector) View() string {
	if !e.visible {
		return ""
	}

	var lines []string

	// Title
	title := e.styles.Title.Render("Select Effect")
	lines = append(lines, title)
	lines = append(lines, "")

	// Effects list
	for i, item := range e.items {
		line := e.renderItem(item, i)
		lines = append(lines, line)
	}

	lines = append(lines, "")

	// Help
	help := e.styles.Subtle.Render("j/k navigate • enter select • esc cancel")
	lines = append(lines, help)

	content := strings.Join(lines, "\n")

	// Modal style
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(e.width)

	return modal.Render(content)
}

func (e EffectSelector) renderItem(item EffectItem, idx int) string {
	cursor := "  "
	if idx == e.cursor {
		cursor = e.styles.Accent.Render("▸ ")
	}

	// Effect name
	name := item.Name
	if item.Name == "no_effect" {
		name = "None"
	}

	// Style based on availability and selection
	var nameStyle, descStyle lipgloss.Style
	if !item.Available {
		nameStyle = e.styles.Subtle
		descStyle = e.styles.Subtle
	} else if idx == e.cursor {
		nameStyle = e.styles.LightSelected
		descStyle = e.styles.Subtle
	} else {
		nameStyle = e.styles.LightUnselected
		descStyle = e.styles.Subtle
	}

	nameStr := nameStyle.Width(15).Render(name)
	descStr := descStyle.Render(item.Description)

	return fmt.Sprintf("%s%s %s", cursor, nameStr, descStr)
}

// Show displays the selector with the given context.
func (e *EffectSelector) Show(lightIDs []string, supportedEffects []string) {
	e.visible = true
	e.lightIDs = lightIDs
	e.items = buildEffectItems(supportedEffects)
	e.cursor = 0
}

// Hide hides the selector.
func (e *EffectSelector) Hide() {
	e.visible = false
}

// Visible returns whether the selector is visible.
func (e EffectSelector) Visible() bool {
	return e.visible
}

// SetSize sets the dimensions.
func (e *EffectSelector) SetSize(width, height int) {
	e.width = width
	if e.width > 70 {
		e.width = 70
	}
	e.height = height
}

func (e *EffectSelector) moveUp() {
	if e.cursor > 0 {
		e.cursor--
	}
}

func (e *EffectSelector) moveDown() {
	if e.cursor < len(e.items)-1 {
		e.cursor++
	}
}

func (e *EffectSelector) selectEffect() tea.Cmd {
	if e.cursor < 0 || e.cursor >= len(e.items) {
		return nil
	}

	item := e.items[e.cursor]
	if !item.Available {
		// Can't select unavailable effect
		return func() tea.Msg {
			return StatusMsg{Message: fmt.Sprintf("Effect '%s' not supported by selected lights", item.Name)}
		}
	}

	// Don't close - let user try multiple effects
	return func() tea.Msg {
		return EffectSelectedMsg{
			Effect:   item.Name,
			LightIDs: e.lightIDs,
		}
	}
}

// EffectSelectedMsg is sent when an effect is selected.
type EffectSelectedMsg struct {
	Effect   string
	LightIDs []string
}

// EffectSelectorClosedMsg is sent when the selector is closed without selection.
type EffectSelectorClosedMsg struct{}
