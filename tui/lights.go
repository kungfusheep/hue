package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kungfusheep/hue/client"
)

// LightList is a component for displaying and selecting lights.
// It handles navigation and selection, emitting messages for actions.
type LightList struct {
	lights   []client.Light
	filtered []client.Light // After query filter
	selected map[string]bool
	cursor   int
	styles   Styles
	keys     KeyMap
	focused  bool
	height   int
	width    int
	offset   int // For scrolling
}

// NewLightList creates a new light list component.
func NewLightList(styles Styles, keys KeyMap) LightList {
	return LightList{
		lights:   nil,
		filtered: nil,
		selected: make(map[string]bool),
		cursor:   0,
		styles:   styles,
		keys:     keys,
		focused:  false,
		height:   10,
		width:    70,
		offset:   0,
	}
}

// Init initialises the component.
func (l LightList) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns commands.
func (l LightList) Update(msg tea.Msg) (LightList, tea.Cmd) {
	if !l.focused {
		return l, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, l.keys.Up):
			l.moveUp()
		case key.Matches(msg, l.keys.Down):
			l.moveDown()
		case key.Matches(msg, l.keys.Top):
			l.cursor = 0
			l.offset = 0
		case key.Matches(msg, l.keys.Bottom):
			l.cursor = len(l.filtered) - 1
			l.ensureVisible()
		case key.Matches(msg, l.keys.Select):
			return l, l.toggleSelection()
		case key.Matches(msg, l.keys.SelectAll):
			return l, l.selectAll()
		case key.Matches(msg, l.keys.Toggle):
			return l, l.emitToggle()
		case key.Matches(msg, l.keys.BrightnessUp):
			return l, l.emitBrightnessChange(10)
		case key.Matches(msg, l.keys.BrightnessDown):
			return l, l.emitBrightnessChange(-10)
		case key.Matches(msg, l.keys.FocusQuery):
			return l, func() tea.Msg {
				return FocusChangedMsg{Component: "query"}
			}
		}
	}

	return l, nil
}

// View renders the component.
func (l LightList) View() string {
	if len(l.filtered) == 0 {
		empty := l.styles.Subtle.Render("No lights match query")
		return l.styles.Section.Width(l.width).Render(empty)
	}

	var lines []string
	visibleCount := l.height

	// Header with dynamic widths
	nameWidth, typeWidth := l.columnWidths()
	header := l.styles.Subtle.Render(
		fmt.Sprintf("    %-*s %-*s %-5s %s", nameWidth, "Name", typeWidth, "Type", "State", "Bri"),
	)
	lines = append(lines, header)

	// Lights
	end := l.offset + visibleCount
	if end > len(l.filtered) {
		end = len(l.filtered)
	}

	for i := l.offset; i < end; i++ {
		light := l.filtered[i]
		lines = append(lines, l.renderLight(light, i))
	}

	// Scroll indicator
	if len(l.filtered) > visibleCount {
		scrollInfo := l.styles.Subtle.Render(
			fmt.Sprintf(" [%d-%d of %d]", l.offset+1, end, len(l.filtered)),
		)
		lines = append(lines, scrollInfo)
	}

	content := strings.Join(lines, "\n")

	style := l.styles.Section.Width(l.width)
	if l.focused {
		style = style.BorderForeground(lipgloss.Color("69"))
	}

	return style.Render(content)
}

// columnWidths calculates dynamic column widths based on available space.
func (l LightList) columnWidths() (nameWidth, typeWidth int) {
	// Fixed widths: cursor(2) + sel(1) + spaces(4) + state(3) + bri(5) + swatch(4) + border(4)
	fixedWidth := 23
	availableWidth := l.width - fixedWidth
	if availableWidth < 20 {
		availableWidth = 20
	}

	// Give 65% to name, 35% to type
	nameWidth = (availableWidth * 65) / 100
	typeWidth = availableWidth - nameWidth

	// Minimums
	if nameWidth < 15 {
		nameWidth = 15
	}
	if typeWidth < 10 {
		typeWidth = 10
	}

	return nameWidth, typeWidth
}

func (l LightList) renderLight(light client.Light, idx int) string {
	nameWidth, typeWidth := l.columnWidths()

	// Selection indicator
	sel := " "
	if l.selected[light.ID] {
		sel = l.styles.Accent.Render("●")
	}

	// Cursor indicator
	cursor := "  "
	if idx == l.cursor && l.focused {
		cursor = l.styles.Accent.Render("▸ ")
	}

	// Name - dynamic width
	name := truncate(light.Metadata.Name, nameWidth-2)
	nameStyle := l.styles.LightUnselected
	if idx == l.cursor && l.focused {
		nameStyle = l.styles.LightSelected
	}
	name = nameStyle.Width(nameWidth).Render(name)

	// Type (archetype) - dynamic width
	archetype := truncate(light.Metadata.Archetype, typeWidth-2)
	archetype = l.styles.LightType.Width(typeWidth).Render(archetype)

	// State
	var state string
	if light.On.On {
		state = l.styles.LightOn.Render("On ")
	} else {
		state = l.styles.LightOff.Render("Off")
	}

	// Brightness
	bri := ""
	if light.On.On {
		bri = fmt.Sprintf("%3.0f%%", light.Dimming.Brightness)
	} else {
		bri = "  - "
	}
	bri = l.styles.LightBrightness.Render(bri)

	// Color swatch (if we have color info)
	swatch := "    "
	if light.Color != nil && light.On.On {
		hex := xyToHex(light.Color.XY.X, light.Color.XY.Y, light.Dimming.Brightness/100.0)
		swatch = ColorSwatch(hex)
	}

	return fmt.Sprintf("%s%s%s %s %s %s %s", cursor, sel, name, archetype, state, bri, swatch)
}

// SetLights sets the full light list (resets cursor and filter).
func (l *LightList) SetLights(lights []client.Light) {
	l.lights = lights
	l.filtered = lights
	l.cursor = 0
	l.offset = 0
}

// UpdateLights updates the light data without resetting cursor or selection.
// Use this when refreshing state after an action.
func (l *LightList) UpdateLights(lights []client.Light) {
	// Build a map of new lights by ID for quick lookup
	newLightMap := make(map[string]client.Light)
	for _, light := range lights {
		newLightMap[light.ID] = light
	}

	l.lights = lights

	// Update filtered list with new data, preserving order
	newFiltered := make([]client.Light, 0, len(l.filtered))
	for _, oldLight := range l.filtered {
		if newLight, ok := newLightMap[oldLight.ID]; ok {
			newFiltered = append(newFiltered, newLight)
		}
	}
	l.filtered = newFiltered

	// Clamp cursor if filtered list shrunk
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
	l.ensureVisible()
}

// SetFiltered sets the filtered light list (after query).
func (l *LightList) SetFiltered(lights []client.Light) {
	l.filtered = lights
	if l.cursor >= len(l.filtered) {
		l.cursor = max(0, len(l.filtered)-1)
	}
	l.ensureVisible()
}

// Focus sets focus on the list.
func (l *LightList) Focus() {
	l.focused = true
}

// Blur removes focus from the list.
func (l *LightList) Blur() {
	l.focused = false
}

// Focused returns whether the list is focused.
func (l LightList) Focused() bool {
	return l.focused
}

// SetSize sets the dimensions of the list.
func (l *LightList) SetSize(width, height int) {
	l.width = width
	l.height = height - 2 // Account for header and scroll indicator
}

// SelectedIDs returns the IDs of selected lights.
func (l LightList) SelectedIDs() []string {
	var ids []string
	for id, selected := range l.selected {
		if selected {
			ids = append(ids, id)
		}
	}
	return ids
}

// CursorLight returns the light under the cursor.
func (l LightList) CursorLight() *client.Light {
	if l.cursor < 0 || l.cursor >= len(l.filtered) {
		return nil
	}
	return &l.filtered[l.cursor]
}

// EffectiveSelection returns selected IDs, or cursor light if none selected.
func (l LightList) EffectiveSelection() []string {
	ids := l.SelectedIDs()
	if len(ids) == 0 {
		if light := l.CursorLight(); light != nil {
			return []string{light.ID}
		}
	}
	return ids
}

func (l *LightList) moveUp() {
	if l.cursor > 0 {
		l.cursor--
		l.ensureVisible()
	}
}

func (l *LightList) moveDown() {
	if l.cursor < len(l.filtered)-1 {
		l.cursor++
		l.ensureVisible()
	}
}

func (l *LightList) ensureVisible() {
	if l.cursor < l.offset {
		l.offset = l.cursor
	}
	if l.cursor >= l.offset+l.height {
		l.offset = l.cursor - l.height + 1
	}
}

func (l *LightList) toggleSelection() tea.Cmd {
	if light := l.CursorLight(); light != nil {
		l.selected[light.ID] = !l.selected[light.ID]
		return func() tea.Msg {
			return LightSelectedMsg{
				LightID:  light.ID,
				Selected: l.selected[light.ID],
			}
		}
	}
	return nil
}

func (l *LightList) selectAll() tea.Cmd {
	// If all filtered are selected, deselect all. Otherwise select all.
	allSelected := true
	for _, light := range l.filtered {
		if !l.selected[light.ID] {
			allSelected = false
			break
		}
	}

	for _, light := range l.filtered {
		l.selected[light.ID] = !allSelected
	}
	return nil
}

func (l LightList) emitToggle() tea.Cmd {
	ids := l.EffectiveSelection()
	if len(ids) == 0 {
		return nil
	}
	return func() tea.Msg {
		return LightToggledMsg{LightIDs: ids}
	}
}

func (l LightList) emitBrightnessChange(delta float64) tea.Cmd {
	ids := l.EffectiveSelection()
	if len(ids) == 0 {
		return nil
	}
	return func() tea.Msg {
		return BrightnessAdjustMsg{LightIDs: ids, Delta: delta}
	}
}

// Helpers

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-1] + "…"
}

// xyToHex converts CIE xy coordinates to a hex color.
// This is a simplified conversion - not perfectly accurate but good for display.
func xyToHex(x, y, brightness float64) string {
	// Convert xy to XYZ
	z := 1.0 - x - y
	Y := brightness
	X := (Y / y) * x
	Z := (Y / y) * z

	// Convert XYZ to RGB
	r := X*3.2406 - Y*1.5372 - Z*0.4986
	g := -X*0.9689 + Y*1.8758 + Z*0.0415
	b := X*0.0557 - Y*0.2040 + Z*1.0570

	// Gamma correction
	correct := func(c float64) float64 {
		if c <= 0.0031308 {
			return 12.92 * c
		}
		return 1.055*pow(c, 1.0/2.4) - 0.055
	}

	r = correct(r)
	g = correct(g)
	b = correct(b)

	// Clamp and convert to 0-255
	clamp := func(c float64) int {
		if c < 0 {
			return 0
		}
		if c > 1 {
			return 255
		}
		return int(c * 255)
	}

	return fmt.Sprintf("#%02X%02X%02X", clamp(r), clamp(g), clamp(b))
}

// pow wraps math.Pow
func pow(base, exp float64) float64 {
	// Using iterative approximation to avoid math import bloat
	// For gamma correction (exp ~= 0.4167), this is good enough
	if exp == 0 {
		return 1
	}
	if base <= 0 {
		return 0
	}
	// Newton-Raphson approximation for x^(1/2.4)
	// exp = 1/2.4 ≈ 0.4167
	result := base
	for i := 0; i < 10; i++ {
		result = result - (result*result*result/base/base - 1) / (3 * result * result / base / base)
	}
	return result
}
