package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ColorItem represents a color option in the selector.
type ColorItem struct {
	Name   string
	Hex    string
	Swatch string
}

// ColorSelector is a modal component for selecting light colors.
type ColorSelector struct {
	items    []ColorItem
	cursor   int
	styles   Styles
	keys     KeyMap
	visible  bool
	width    int
	height   int
	lightIDs []string // IDs of lights to apply color to
}

// GetColorPresets returns the list of color presets.
func GetColorPresets() []ColorItem {
	presets := []struct {
		name string
		hex  string
	}{
		{"Warm White", "#FFE4B5"},
		{"Cool White", "#F0F8FF"},
		{"Daylight", "#FFFACD"},
		{"Red", "#FF0000"},
		{"Orange", "#FF8C00"},
		{"Yellow", "#FFD700"},
		{"Lime", "#32CD32"},
		{"Green", "#00FF00"},
		{"Teal", "#008080"},
		{"Cyan", "#00FFFF"},
		{"Sky Blue", "#87CEEB"},
		{"Blue", "#0000FF"},
		{"Purple", "#8B00FF"},
		{"Magenta", "#FF00FF"},
		{"Pink", "#FF69B4"},
		{"Coral", "#FF7F50"},
	}

	items := make([]ColorItem, len(presets))
	for i, p := range presets {
		items[i] = ColorItem{
			Name:   p.name,
			Hex:    p.hex,
			Swatch: ColorSwatch(p.hex),
		}
	}
	return items
}

// NewColorSelector creates a new color selector component.
func NewColorSelector(styles Styles, keys KeyMap) ColorSelector {
	return ColorSelector{
		items:  GetColorPresets(),
		cursor: 0,
		styles: styles,
		keys:   keys,
		width:  50,
		height: 20,
	}
}

// Init initialises the component.
func (c ColorSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns commands.
func (c ColorSelector) Update(msg tea.Msg) (ColorSelector, tea.Cmd) {
	if !c.visible {
		return c, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, c.keys.Up):
			c.moveUp()
		case key.Matches(msg, c.keys.Down):
			c.moveDown()
		case key.Matches(msg, c.keys.Top):
			c.cursor = 0
		case key.Matches(msg, c.keys.Bottom):
			c.cursor = len(c.items) - 1
		case key.Matches(msg, c.keys.Enter):
			return c, c.selectColor()
		case key.Matches(msg, c.keys.Escape):
			c.visible = false
			return c, func() tea.Msg {
				return ColorSelectorClosedMsg{}
			}
		}
	}

	return c, nil
}

// View renders the component.
func (c ColorSelector) View() string {
	if !c.visible {
		return ""
	}

	var lines []string

	// Title
	title := c.styles.Title.Render("Select Color")
	lines = append(lines, title)
	lines = append(lines, "")

	// Colors list - display in two columns
	itemsPerColumn := (len(c.items) + 1) / 2
	for i := 0; i < itemsPerColumn; i++ {
		var row string

		// Left column
		leftItem := c.items[i]
		row = c.renderItem(leftItem, i)

		// Right column (if exists)
		rightIdx := i + itemsPerColumn
		if rightIdx < len(c.items) {
			rightItem := c.items[rightIdx]
			row += "  " + c.renderItem(rightItem, rightIdx)
		}

		lines = append(lines, row)
	}

	lines = append(lines, "")

	// Help
	help := c.styles.Subtle.Render("j/k navigate • enter select • esc cancel")
	lines = append(lines, help)

	content := strings.Join(lines, "\n")

	// Modal style
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(c.width)

	return modal.Render(content)
}

func (c ColorSelector) renderItem(item ColorItem, idx int) string {
	cursor := "  "
	if idx == c.cursor {
		cursor = c.styles.Accent.Render("▸ ")
	}

	// Style based on selection
	var nameStyle lipgloss.Style
	if idx == c.cursor {
		nameStyle = c.styles.LightSelected
	} else {
		nameStyle = c.styles.LightUnselected
	}

	nameStr := nameStyle.Width(12).Render(item.Name)

	return fmt.Sprintf("%s%s %s", cursor, item.Swatch, nameStr)
}

// Show displays the selector with the given context.
func (c *ColorSelector) Show(lightIDs []string) {
	c.visible = true
	c.lightIDs = lightIDs
	c.cursor = 0
}

// Hide hides the selector.
func (c *ColorSelector) Hide() {
	c.visible = false
}

// Visible returns whether the selector is visible.
func (c ColorSelector) Visible() bool {
	return c.visible
}

// SetSize sets the dimensions.
func (c *ColorSelector) SetSize(width, height int) {
	c.width = width
	if c.width > 60 {
		c.width = 60
	}
	c.height = height
}

func (c *ColorSelector) moveUp() {
	if c.cursor > 0 {
		c.cursor--
	}
}

func (c *ColorSelector) moveDown() {
	if c.cursor < len(c.items)-1 {
		c.cursor++
	}
}

func (c *ColorSelector) selectColor() tea.Cmd {
	if c.cursor < 0 || c.cursor >= len(c.items) {
		return nil
	}

	item := c.items[c.cursor]

	// Don't close - let user try multiple colors
	return func() tea.Msg {
		return ColorSelectedMsg{
			Color:    item.Hex,
			LightIDs: c.lightIDs,
		}
	}
}

// ColorSelectedMsg is sent when a color is selected.
type ColorSelectedMsg struct {
	Color    string
	LightIDs []string
}

// ColorSelectorClosedMsg is sent when the selector is closed without selection.
type ColorSelectorClosedMsg struct{}
