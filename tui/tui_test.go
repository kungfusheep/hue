package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/kungfusheep/hue/client"
)

// Test fixtures
func makeTestLight(id, name, archetype string, on bool, brightness float64) client.Light {
	light := client.Light{
		ID: id,
		Metadata: client.Metadata{
			Name:      name,
			Archetype: archetype,
		},
		On:      client.OnState{On: on},
		Dimming: client.Dimming{Brightness: brightness},
	}
	return light
}

func makeTestLightWithColor(id, name, archetype string, on bool, brightness float64) client.Light {
	light := makeTestLight(id, name, archetype, on, brightness)
	light.Color = &client.Color{
		XY: client.XY{X: 0.5, Y: 0.4},
	}
	return light
}

func getTestLights() []client.Light {
	return []client.Light{
		makeTestLightWithColor("light-1", "Office Desk", "desk_lamp", true, 80),
		makeTestLightWithColor("light-2", "Office Ceiling", "ceiling_round", true, 100),
		makeTestLight("light-3", "Bedroom Lamp", "table_lamp", false, 0),
		makeTestLightWithColor("light-4", "Kitchen Strip", "hue_lightstrip", true, 50),
		makeTestLight("light-5", "Hallway", "ceiling_round", false, 0),
	}
}

// Mock action handler for testing
type mockActionHandler struct {
	lights            []client.Light
	rooms             []client.Room
	toggledIDs        []string
	fetchError        error
	toggleError       error
	lastBrightnessIDs []string
	lastBrightness    float64
}

func (m *mockActionHandler) FetchLights() ([]client.Light, error) {
	if m.fetchError != nil {
		return nil, m.fetchError
	}
	return m.lights, nil
}

func (m *mockActionHandler) FetchRooms() ([]client.Room, error) {
	return m.rooms, nil
}

func (m *mockActionHandler) ToggleLights(ids []string) error {
	m.toggledIDs = append(m.toggledIDs, ids...)
	return m.toggleError
}

func (m *mockActionHandler) SetBrightness(ids []string, brightness float64) error {
	m.lastBrightnessIDs = ids
	m.lastBrightness = brightness
	return nil
}

func (m *mockActionHandler) SetColor(ids []string, color string) error {
	return nil
}

func (m *mockActionHandler) SetEffect(ids []string, effect string) error {
	return nil
}

func (m *mockActionHandler) FetchScenes() ([]client.Scene, error) {
	return nil, nil
}

func (m *mockActionHandler) ActivateScene(id string) error {
	return nil
}

// ============= QueryInput Tests =============

func TestQueryInput_Init(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)

	if q.Focused() {
		t.Error("QueryInput should not be focused on init")
	}
	if q.Value() != "" {
		t.Error("QueryInput should be empty on init")
	}
}

func TestQueryInput_Focus(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)

	q.Focus()

	if !q.Focused() {
		t.Error("QueryInput should be focused after Focus()")
	}
}

func TestQueryInput_Blur(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)

	q.Focus()
	q.Blur()

	if q.Focused() {
		t.Error("QueryInput should not be focused after Blur()")
	}
}

func TestQueryInput_SetValue(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)

	q.SetValue(`@"office"`)

	if q.Value() != `@"office"` {
		t.Errorf("Expected @\"office\", got %s", q.Value())
	}
}

func TestQueryInput_TypeEmitsChange(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)
	q.Focus()

	// Simulate typing 'o'
	var cmd tea.Cmd
	q, cmd = q.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if cmd == nil {
		t.Fatal("Expected command after typing")
	}

	// The cmd is a batch, so we need to check value changed
	if q.Value() != "o" {
		t.Errorf("Expected value 'o', got %s", q.Value())
	}

	// Execute cmd and look for QueryChangedMsg in batch
	msg := cmd()
	// BatchMsg contains multiple messages - just verify we got something
	// The important thing is the value changed
	if msg == nil {
		t.Error("Expected non-nil message")
	}
}

func TestQueryInput_EnterEmitsSubmit(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)
	q.Focus()
	q.SetValue(`@"office"`)

	var cmd tea.Cmd
	q, cmd = q.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Expected command after enter")
	}

	msg := cmd()
	submitMsg, ok := msg.(QuerySubmittedMsg)
	if !ok {
		t.Fatalf("Expected QuerySubmittedMsg, got %T", msg)
	}
	if submitMsg.Query != `@"office"` {
		t.Errorf("Expected query @\"office\", got %s", submitMsg.Query)
	}
}

func TestQueryInput_EscapeBlurs(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)
	q.Focus()

	q, _ = q.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if q.Focused() {
		t.Error("QueryInput should blur on escape")
	}
}

func TestQueryInput_View(t *testing.T) {
	styles := DefaultStyles()
	q := NewQueryInput(styles)

	view := q.View()

	if !strings.Contains(view, "Query") {
		t.Error("QueryInput view should contain label")
	}
}

// ============= LightList Tests =============

func TestLightList_Init(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)

	if l.Focused() {
		t.Error("LightList should not be focused on init")
	}
	if len(l.SelectedIDs()) != 0 {
		t.Error("LightList should have no selections on init")
	}
}

func TestLightList_SetLights(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)

	lights := getTestLights()
	l.SetLights(lights)

	if len(l.filtered) != 5 {
		t.Errorf("Expected 5 lights, got %d", len(l.filtered))
	}
}

func TestLightList_Navigation(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Initial cursor at 0
	if l.cursor != 0 {
		t.Errorf("Initial cursor should be 0, got %d", l.cursor)
	}

	// Move down
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if l.cursor != 1 {
		t.Errorf("Cursor should be 1 after j, got %d", l.cursor)
	}

	// Move down again
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if l.cursor != 2 {
		t.Errorf("Cursor should be 2 after second j, got %d", l.cursor)
	}

	// Move up
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if l.cursor != 1 {
		t.Errorf("Cursor should be 1 after k, got %d", l.cursor)
	}
}

func TestLightList_NavigationBounds(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Try to go above 0
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if l.cursor != 0 {
		t.Error("Cursor should stay at 0 when moving up at top")
	}

	// Go to bottom
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if l.cursor != 4 {
		t.Errorf("Cursor should be 4 at bottom, got %d", l.cursor)
	}

	// Try to go past bottom
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if l.cursor != 4 {
		t.Error("Cursor should stay at bottom")
	}

	// Go to top
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if l.cursor != 0 {
		t.Errorf("Cursor should be 0 at top, got %d", l.cursor)
	}
}

func TestLightList_Selection(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Select first light
	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeySpace})

	if len(l.SelectedIDs()) != 1 {
		t.Errorf("Expected 1 selected, got %d", len(l.SelectedIDs()))
	}

	// Verify message
	if cmd != nil {
		msg := cmd()
		selectMsg, ok := msg.(LightSelectedMsg)
		if !ok {
			t.Fatalf("Expected LightSelectedMsg, got %T", msg)
		}
		if selectMsg.LightID != "light-1" {
			t.Errorf("Expected light-1, got %s", selectMsg.LightID)
		}
		if !selectMsg.Selected {
			t.Error("Expected Selected=true")
		}
	}

	// Deselect
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeySpace})
	if len(l.SelectedIDs()) != 0 {
		t.Error("Should have no selections after toggle")
	}
}

func TestLightList_SelectAll(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Select all
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(l.SelectedIDs()) != 5 {
		t.Errorf("Expected 5 selected, got %d", len(l.SelectedIDs()))
	}

	// Deselect all
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if len(l.SelectedIDs()) != 0 {
		t.Error("Expected 0 selected after second toggle all")
	}
}

func TestLightList_Toggle(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Toggle cursor light (no selection)
	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	if cmd == nil {
		t.Fatal("Expected toggle command")
	}

	msg := cmd()
	toggleMsg, ok := msg.(LightToggledMsg)
	if !ok {
		t.Fatalf("Expected LightToggledMsg, got %T", msg)
	}
	if len(toggleMsg.LightIDs) != 1 || toggleMsg.LightIDs[0] != "light-1" {
		t.Errorf("Expected [light-1], got %v", toggleMsg.LightIDs)
	}
}

func TestLightList_ToggleSelection(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Select lights 1 and 3
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeySpace}) // Select light-1
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeySpace}) // Select light-3

	// Toggle selected
	_, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})

	msg := cmd()
	toggleMsg := msg.(LightToggledMsg)
	if len(toggleMsg.LightIDs) != 2 {
		t.Errorf("Expected 2 lights to toggle, got %d", len(toggleMsg.LightIDs))
	}
}

func TestLightList_BrightnessUp(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Increase brightness
	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{']'}})

	if cmd == nil {
		t.Fatal("Expected brightness command")
	}

	msg := cmd()
	briMsg, ok := msg.(BrightnessAdjustMsg)
	if !ok {
		t.Fatalf("Expected BrightnessAdjustMsg, got %T", msg)
	}
	if briMsg.Delta != 10 {
		t.Errorf("Expected delta 10, got %v", briMsg.Delta)
	}
	if len(briMsg.LightIDs) != 1 || briMsg.LightIDs[0] != "light-1" {
		t.Errorf("Expected [light-1], got %v", briMsg.LightIDs)
	}
}

func TestLightList_BrightnessDown(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Decrease brightness
	l, cmd := l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'['}})

	if cmd == nil {
		t.Fatal("Expected brightness command")
	}

	msg := cmd()
	briMsg, ok := msg.(BrightnessAdjustMsg)
	if !ok {
		t.Fatalf("Expected BrightnessAdjustMsg, got %T", msg)
	}
	if briMsg.Delta != -10 {
		t.Errorf("Expected delta -10, got %v", briMsg.Delta)
	}
}

func TestLightList_CursorLight(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())

	light := l.CursorLight()
	if light == nil {
		t.Fatal("Expected cursor light")
	}
	if light.ID != "light-1" {
		t.Errorf("Expected light-1, got %s", light.ID)
	}
}

func TestLightList_SetFiltered(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Move cursor to position 3
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Filter to only 2 lights
	filtered := []client.Light{
		getTestLights()[0],
		getTestLights()[1],
	}
	l.SetFiltered(filtered)

	// Cursor should be clamped
	if l.cursor >= len(filtered) {
		t.Errorf("Cursor should be clamped, got %d", l.cursor)
	}
}

func TestLightList_View(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	view := l.View()

	// Should contain light names
	if !strings.Contains(view, "Office Desk") {
		t.Error("View should contain light name")
	}

	// Should show state
	if !strings.Contains(view, "On") {
		t.Error("View should show on/off state")
	}
}

func TestLightList_ViewEmpty(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights([]client.Light{})

	view := l.View()

	if !strings.Contains(view, "No lights") {
		t.Error("Empty view should show 'No lights' message")
	}
}

func TestLightList_UpdateLightsPreservesCursor(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// Move cursor to position 3
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	if l.cursor != 3 {
		t.Fatalf("Expected cursor at 3, got %d", l.cursor)
	}

	// Update lights (simulating refresh after action)
	l.UpdateLights(getTestLights())

	// Cursor should be preserved
	if l.cursor != 3 {
		t.Errorf("Expected cursor to stay at 3 after UpdateLights, got %d", l.cursor)
	}
}

func TestLightList_UpdateLightsPreservesFilter(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())

	// Filter to office lights only
	officeOnly := []client.Light{
		getTestLights()[0], // Office Desk
		getTestLights()[1], // Office Ceiling
	}
	l.SetFiltered(officeOnly)

	if len(l.filtered) != 2 {
		t.Fatalf("Expected 2 filtered lights, got %d", len(l.filtered))
	}

	// Update lights (simulating refresh after action)
	l.UpdateLights(getTestLights())

	// Filter should be preserved (still only office lights)
	if len(l.filtered) != 2 {
		t.Errorf("Expected filter preserved with 2 lights, got %d", len(l.filtered))
	}
}

func TestLightList_EffectiveSelection(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	l := NewLightList(styles, keys)
	l.SetLights(getTestLights())
	l.Focus()

	// No selection - should return cursor light
	ids := l.EffectiveSelection()
	if len(ids) != 1 || ids[0] != "light-1" {
		t.Errorf("Expected cursor light, got %v", ids)
	}

	// With selection - should return selection
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	l, _ = l.Update(tea.KeyMsg{Type: tea.KeySpace})

	ids = l.EffectiveSelection()
	if len(ids) != 1 || ids[0] != "light-2" {
		t.Errorf("Expected selected light, got %v", ids)
	}
}

// ============= Model Tests =============

func TestModel_Init(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)

	cmds := m.Init()
	if cmds == nil {
		t.Error("Init should return commands to load data")
	}
}

func TestModel_LightsLoaded(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true

	newM, _ := m.Update(LightsLoadedMsg{Lights: getTestLights()})
	m = newM.(Model)

	if len(m.allLights) != 5 {
		t.Errorf("Expected 5 lights, got %d", len(m.allLights))
	}
	if !strings.Contains(m.status, "5 lights") {
		t.Errorf("Status should mention loaded lights, got: %s", m.status)
	}
}

func TestModel_QueryChanged(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())

	newM, _ := m.Update(QueryChangedMsg{Query: `@"office"`})
	m = newM.(Model)

	// Should filter to office lights
	if len(m.lights.filtered) != 2 {
		t.Errorf("Expected 2 office lights, got %d", len(m.lights.filtered))
	}
}

func TestModel_QueryFiltering(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.allLights = getTestLights()

	tests := []struct {
		query    string
		expected int
	}{
		{"", 5},                // All lights
		{`@"office"`, 2},      // Office lights
		{`@"bedroom"`, 1},     // Bedroom light
		{`@"on:"`, 3},         // On lights
		{`@"off:"`, 2},        // Off lights
		{`@"nonexistent"`, 0}, // No matches
	}

	for _, tt := range tests {
		filtered := m.filterLights(tt.query)
		if len(filtered) != tt.expected {
			t.Errorf("Query %q: expected %d, got %d", tt.query, tt.expected, len(filtered))
		}
	}
}

func TestModel_FocusSwitch(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())
	// Simulate state where lights are focused (e.g., after submitting a query)
	m.query.Blur()
	m.lights.Focus()

	// Switch to query with /
	newM, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newM.(Model)

	if !m.query.Focused() {
		t.Error("Query should be focused after /")
	}
	if m.lights.Focused() {
		t.Error("Lights should not be focused after /")
	}
}

func TestModel_FocusChangedMsg(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true

	// Focus query via message
	newM, _ := m.Update(FocusChangedMsg{Component: "query"})
	m = newM.(Model)
	// Note: Focus() returns a command, so we can't directly test focus state here
	// but we can verify the lights are blurred
	if m.lights.Focused() {
		t.Error("Lights should be blurred when query focused")
	}
}

func TestModel_ToggleAction(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())
	m.lights.Focus()

	// Should emit toggle command
	_, cmd := m.Update(LightToggledMsg{LightIDs: []string{"light-1", "light-2"}})

	if cmd == nil {
		t.Fatal("Expected toggle command")
	}

	// Execute the command
	msg := cmd()

	// Should be ActionCompletedMsg
	if _, ok := msg.(ActionCompletedMsg); !ok {
		t.Errorf("Expected ActionCompletedMsg, got %T", msg)
	}

	// Handler should have been called
	if len(handler.toggledIDs) != 2 {
		t.Errorf("Expected 2 toggled IDs, got %d", len(handler.toggledIDs))
	}
}

func TestModel_BrightnessAdjust(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())

	// Test brightness increase - light-1 has 80% brightness, light-4 has 50%
	// Average should be 65%, +10 = 75%
	_, cmd := m.Update(BrightnessAdjustMsg{LightIDs: []string{"light-1", "light-4"}, Delta: 10})

	if cmd == nil {
		t.Fatal("Expected brightness command")
	}

	// Execute the command
	msg := cmd()

	// Should be ActionCompletedMsg
	if completed, ok := msg.(ActionCompletedMsg); !ok {
		t.Errorf("Expected ActionCompletedMsg, got %T", msg)
	} else {
		if !strings.Contains(completed.Action, "Brightness") {
			t.Errorf("Expected Brightness action, got: %s", completed.Action)
		}
	}

	// Handler should have been called with correct brightness
	if len(handler.lastBrightnessIDs) != 2 {
		t.Errorf("Expected 2 brightness IDs, got %d", len(handler.lastBrightnessIDs))
	}
	// Average of 80 and 50 is 65, +10 = 75
	if handler.lastBrightness != 75 {
		t.Errorf("Expected brightness 75, got %v", handler.lastBrightness)
	}
}

func TestModel_BrightnessClamp(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())

	// Test brightness increase beyond 100% - light-2 is at 100%
	_, cmd := m.Update(BrightnessAdjustMsg{LightIDs: []string{"light-2"}, Delta: 10})

	if cmd == nil {
		t.Fatal("Expected brightness command")
	}

	// Execute the command
	cmd()

	// Should be clamped to 100
	if handler.lastBrightness != 100 {
		t.Errorf("Expected brightness clamped to 100, got %v", handler.lastBrightness)
	}
}

func TestModel_ActionCompleted(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true

	newM, _ := m.Update(ActionCompletedMsg{Action: "Toggle", Err: nil})
	m = newM.(Model)

	if !strings.Contains(m.status, "Toggle") || !strings.Contains(m.status, "completed") {
		t.Errorf("Status should mention action completed, got: %s", m.status)
	}
}

func TestModel_Quit(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true

	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})

	if cmd == nil {
		t.Fatal("Expected quit command")
	}

	// Execute and check for quit
	msg := cmd()
	if msg != tea.Quit() {
		t.Error("Expected tea.Quit message")
	}
}

func TestModel_WindowSize(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)

	newM, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	m = newM.(Model)

	if !m.ready {
		t.Error("Model should be ready after window size")
	}
	if m.width != 120 || m.height != 40 {
		t.Errorf("Expected 120x40, got %dx%d", m.width, m.height)
	}
}

func TestModel_View(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)
	m.ready = true
	m.allLights = getTestLights()
	m.lights.SetLights(getTestLights())
	m.width = 80
	m.height = 24

	view := m.View()

	// Should contain title
	if !strings.Contains(view, "HUE") {
		t.Error("View should contain title")
	}

	// Should contain help
	if !strings.Contains(view, "quit") {
		t.Error("View should contain help")
	}
}

func TestModel_ViewNotReady(t *testing.T) {
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)

	view := m.View()

	if !strings.Contains(view, "Loading") {
		t.Error("Not ready view should show loading")
	}
}

// ============= Styles Tests =============

func TestDefaultStyles(t *testing.T) {
	styles := DefaultStyles()

	// Just verify styles are created without panic
	_ = styles.Header.Render("test")
	_ = styles.LightOn.Render("On")
	_ = styles.LightOff.Render("Off")
}

func TestColorSwatch(t *testing.T) {
	swatch := ColorSwatch("#FF0000")
	if len(swatch) == 0 {
		t.Error("Color swatch should not be empty")
	}

	emptySwatch := ColorSwatch("")
	if len(emptySwatch) != 4 {
		t.Error("Empty color should return 4 spaces")
	}
}

// ============= Keys Tests =============

func TestDefaultKeyMap(t *testing.T) {
	keys := DefaultKeyMap()

	// Verify key bindings exist
	if len(keys.Up.Keys()) == 0 {
		t.Error("Up key should be bound")
	}
	if len(keys.Down.Keys()) == 0 {
		t.Error("Down key should be bound")
	}
	if len(keys.Quit.Keys()) == 0 {
		t.Error("Quit key should be bound")
	}
}

func TestKeyMap_ShortHelp(t *testing.T) {
	keys := DefaultKeyMap()
	help := keys.ShortHelp()

	if len(help) == 0 {
		t.Error("ShortHelp should return bindings")
	}
}

func TestKeyMap_FullHelp(t *testing.T) {
	keys := DefaultKeyMap()
	help := keys.FullHelp()

	if len(help) == 0 {
		t.Error("FullHelp should return binding groups")
	}
}

// ============= EffectSelector Tests =============

func TestEffectSelector_Init(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)

	if e.Visible() {
		t.Error("EffectSelector should not be visible on init")
	}
}

func TestEffectSelector_Show(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)

	e.Show([]string{"light-1"}, []string{"candle", "fire"})

	if !e.Visible() {
		t.Error("EffectSelector should be visible after Show")
	}
	if len(e.lightIDs) != 1 {
		t.Errorf("Expected 1 light ID, got %d", len(e.lightIDs))
	}
}

func TestEffectSelector_Hide(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)

	e.Show([]string{"light-1"}, nil)
	e.Hide()

	if e.Visible() {
		t.Error("EffectSelector should not be visible after Hide")
	}
}

func TestEffectSelector_Navigation(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)
	e.Show([]string{"light-1"}, nil)

	// Initial cursor at 0
	if e.cursor != 0 {
		t.Errorf("Initial cursor should be 0, got %d", e.cursor)
	}

	// Move down
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if e.cursor != 1 {
		t.Errorf("Cursor should be 1 after j, got %d", e.cursor)
	}

	// Move up
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if e.cursor != 0 {
		t.Errorf("Cursor should be 0 after k, got %d", e.cursor)
	}
}

func TestEffectSelector_SelectEffect(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)
	e.Show([]string{"light-1"}, nil) // nil = all effects available

	// Select with enter
	e, cmd := e.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should stay visible so user can try multiple effects
	if !e.Visible() {
		t.Error("EffectSelector should stay visible after selection (to allow trying multiple effects)")
	}

	if cmd == nil {
		t.Fatal("Expected command after selection")
	}

	msg := cmd()
	selectMsg, ok := msg.(EffectSelectedMsg)
	if !ok {
		t.Fatalf("Expected EffectSelectedMsg, got %T", msg)
	}

	if selectMsg.Effect != "no_effect" { // First item is "no_effect"
		t.Errorf("Expected 'no_effect', got %s", selectMsg.Effect)
	}
	if len(selectMsg.LightIDs) != 1 {
		t.Errorf("Expected 1 light ID, got %d", len(selectMsg.LightIDs))
	}
}

func TestEffectSelector_EscapeCloses(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)
	e.Show([]string{"light-1"}, nil)

	e, cmd := e.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if e.Visible() {
		t.Error("EffectSelector should hide after escape")
	}

	if cmd == nil {
		t.Fatal("Expected command after escape")
	}

	msg := cmd()
	if _, ok := msg.(EffectSelectorClosedMsg); !ok {
		t.Errorf("Expected EffectSelectorClosedMsg, got %T", msg)
	}
}

func TestEffectSelector_View(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)

	// Not visible - should return empty
	view := e.View()
	if view != "" {
		t.Error("Hidden selector should return empty string")
	}

	// Visible - should have content
	e.Show([]string{"light-1"}, nil)
	view = e.View()

	if !strings.Contains(view, "Select Effect") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "candle") {
		t.Error("View should contain effect names")
	}
}

func TestEffectSelector_UnavailableEffect(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	e := NewEffectSelector(styles, keys)

	// Only "fire" is available
	e.Show([]string{"light-1"}, []string{"fire"})

	// Move to "candle" (second item, not in available list)
	e, _ = e.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})

	// Try to select unavailable effect
	e, cmd := e.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should still be visible (selection failed)
	if !e.Visible() {
		t.Error("EffectSelector should remain visible when selecting unavailable effect")
	}

	// Should emit status message, not selection
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(EffectSelectedMsg); ok {
			t.Error("Should not emit EffectSelectedMsg for unavailable effect")
		}
	}
}

// ============= Integration Tests =============

func TestFullWorkflow(t *testing.T) {
	// Simulate a full workflow: load lights, filter, select, toggle
	handler := &mockActionHandler{lights: getTestLights()}
	m := NewModel(handler)

	// Window size (makes model ready)
	newM, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m = newM.(Model)

	// Lights loaded
	newM, _ = m.Update(LightsLoadedMsg{Lights: getTestLights()})
	m = newM.(Model)

	// Focus query
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m = newM.(Model)

	// Simulate typing a query by sending QueryChangedMsg directly
	// (typing individual keys is complex due to textinput internals)
	newM, _ = m.Update(QueryChangedMsg{Query: `@"office"`})
	m = newM.(Model)

	// Switch back to lights
	newM, _ = m.Update(FocusChangedMsg{Component: "lights"})
	m = newM.(Model)

	// Select first light
	newM, _ = m.Update(tea.KeyMsg{Type: tea.KeySpace})
	m = newM.(Model)

	// Toggle
	newM, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	m = newM.(Model)

	// Get the toggle message
	if cmd != nil {
		msg := cmd()
		if toggleMsg, ok := msg.(LightToggledMsg); ok {
			// Execute toggle
			newM, cmd = m.Update(toggleMsg)
			m = newM.(Model)
			if cmd != nil {
				// Execute toggle action
				msg = cmd()
				newM, _ = m.Update(msg)
				m = newM.(Model)
			}
		}
	}

	// Verify handler was called
	if len(handler.toggledIDs) == 0 {
		t.Error("Expected lights to be toggled")
	}
}

// ============= ColorSelector Tests =============

func TestColorSelector_Init(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)

	if c.Visible() {
		t.Error("ColorSelector should not be visible on init")
	}
}

func TestColorSelector_Show(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)

	c.Show([]string{"light-1", "light-2"})

	if !c.Visible() {
		t.Error("ColorSelector should be visible after Show")
	}
}

func TestColorSelector_Hide(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)

	c.Show([]string{"light-1"})
	c.Hide()

	if c.Visible() {
		t.Error("ColorSelector should not be visible after Hide")
	}
}

func TestColorSelector_Navigation(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)
	c.Show([]string{"light-1"})

	// Initial cursor is 0
	if c.cursor != 0 {
		t.Errorf("Initial cursor should be 0, got %d", c.cursor)
	}

	// Move down
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if c.cursor != 1 {
		t.Errorf("Cursor should be 1 after j, got %d", c.cursor)
	}

	// Move up
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if c.cursor != 0 {
		t.Errorf("Cursor should be 0 after k, got %d", c.cursor)
	}

	// Go to bottom
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'G'}})
	if c.cursor != len(c.items)-1 {
		t.Errorf("Cursor should be at end after G, got %d", c.cursor)
	}

	// Go to top
	c, _ = c.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'g'}})
	if c.cursor != 0 {
		t.Errorf("Cursor should be 0 after g, got %d", c.cursor)
	}
}

func TestColorSelector_SelectColor(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)
	c.Show([]string{"light-1"})

	// Select first color (Warm White)
	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEnter})

	// Should still be visible (stays open for experimentation)
	if !c.Visible() {
		t.Error("ColorSelector should remain visible after selection")
	}

	if cmd == nil {
		t.Fatal("Expected selection command")
	}

	msg := cmd()
	selectMsg, ok := msg.(ColorSelectedMsg)
	if !ok {
		t.Fatalf("Expected ColorSelectedMsg, got %T", msg)
	}

	if selectMsg.Color != "#FFE4B5" { // Warm White
		t.Errorf("Expected '#FFE4B5', got %s", selectMsg.Color)
	}
	if len(selectMsg.LightIDs) != 1 {
		t.Errorf("Expected 1 light ID, got %d", len(selectMsg.LightIDs))
	}
}

func TestColorSelector_EscapeCloses(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)
	c.Show([]string{"light-1"})

	c, cmd := c.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if c.Visible() {
		t.Error("ColorSelector should hide after escape")
	}

	if cmd == nil {
		t.Fatal("Expected command after escape")
	}

	msg := cmd()
	if _, ok := msg.(ColorSelectorClosedMsg); !ok {
		t.Errorf("Expected ColorSelectorClosedMsg, got %T", msg)
	}
}

func TestColorSelector_View(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	c := NewColorSelector(styles, keys)

	// Not visible - should return empty
	view := c.View()
	if view != "" {
		t.Error("Hidden selector should return empty string")
	}

	// Visible - should have content
	c.Show([]string{"light-1"})
	view = c.View()

	if !strings.Contains(view, "Select Color") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "Red") {
		t.Error("View should contain color names")
	}
}

func TestGetColorPresets(t *testing.T) {
	presets := GetColorPresets()

	if len(presets) == 0 {
		t.Error("Should have color presets")
	}

	// Verify first preset
	if presets[0].Name != "Warm White" {
		t.Errorf("First preset should be 'Warm White', got %s", presets[0].Name)
	}

	// All presets should have hex values
	for _, p := range presets {
		if p.Hex == "" {
			t.Errorf("Preset %s should have a hex value", p.Name)
		}
		if len(p.Hex) != 7 || p.Hex[0] != '#' {
			t.Errorf("Preset %s should have valid hex format, got %s", p.Name, p.Hex)
		}
	}
}

// ============= SceneSelector Tests =============

func TestSceneSelector_Init(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	if s.Visible() {
		t.Error("SceneSelector should not be visible on init")
	}
}

func TestSceneSelector_Show(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	scenes := []client.Scene{
		{ID: "scene-1", Metadata: client.Metadata{Name: "Relax"}, Group: client.ResourceIdentifier{RID: "room-1"}},
		{ID: "scene-2", Metadata: client.Metadata{Name: "Energize"}, Group: client.ResourceIdentifier{RID: "room-1"}},
	}
	rooms := []client.Room{
		{ID: "room-1", Metadata: client.Metadata{Name: "Living Room"}},
	}

	s.Show(scenes, rooms)

	if !s.Visible() {
		t.Error("SceneSelector should be visible after Show")
	}
	if len(s.items) != 2 {
		t.Errorf("Expected 2 items, got %d", len(s.items))
	}
}

func TestSceneSelector_Hide(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	s.Show(nil, nil)
	s.Hide()

	if s.Visible() {
		t.Error("SceneSelector should not be visible after Hide")
	}
}

func TestSceneSelector_Navigation(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	scenes := []client.Scene{
		{ID: "scene-1", Metadata: client.Metadata{Name: "A"}, Group: client.ResourceIdentifier{RID: "room-1"}},
		{ID: "scene-2", Metadata: client.Metadata{Name: "B"}, Group: client.ResourceIdentifier{RID: "room-1"}},
		{ID: "scene-3", Metadata: client.Metadata{Name: "C"}, Group: client.ResourceIdentifier{RID: "room-1"}},
	}
	rooms := []client.Room{{ID: "room-1", Metadata: client.Metadata{Name: "Room"}}}
	s.Show(scenes, rooms)

	// Initial cursor is 0
	if s.cursor != 0 {
		t.Errorf("Initial cursor should be 0, got %d", s.cursor)
	}

	// Move down
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if s.cursor != 1 {
		t.Errorf("Cursor should be 1 after j, got %d", s.cursor)
	}

	// Move up
	s, _ = s.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	if s.cursor != 0 {
		t.Errorf("Cursor should be 0 after k, got %d", s.cursor)
	}
}

func TestSceneSelector_SelectScene(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	scenes := []client.Scene{
		{ID: "scene-1", Metadata: client.Metadata{Name: "Relax"}, Group: client.ResourceIdentifier{RID: "room-1"}},
	}
	rooms := []client.Room{{ID: "room-1", Metadata: client.Metadata{Name: "Room"}}}
	s.Show(scenes, rooms)

	s, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEnter})

	if cmd == nil {
		t.Fatal("Expected selection command")
	}

	msg := cmd()
	selectMsg, ok := msg.(SceneSelectedMsg)
	if !ok {
		t.Fatalf("Expected SceneSelectedMsg, got %T", msg)
	}

	if selectMsg.SceneID != "scene-1" {
		t.Errorf("Expected scene-1, got %s", selectMsg.SceneID)
	}
	if selectMsg.SceneName != "Relax" {
		t.Errorf("Expected 'Relax', got %s", selectMsg.SceneName)
	}
}

func TestSceneSelector_EscapeCloses(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)
	s.Show(nil, nil)

	s, cmd := s.Update(tea.KeyMsg{Type: tea.KeyEscape})

	if s.Visible() {
		t.Error("SceneSelector should hide after escape")
	}

	if cmd == nil {
		t.Fatal("Expected command after escape")
	}

	msg := cmd()
	if _, ok := msg.(SceneSelectorClosedMsg); !ok {
		t.Errorf("Expected SceneSelectorClosedMsg, got %T", msg)
	}
}

func TestSceneSelector_View(t *testing.T) {
	styles := DefaultStyles()
	keys := DefaultKeyMap()
	s := NewSceneSelector(styles, keys)

	// Not visible - should return empty
	view := s.View()
	if view != "" {
		t.Error("Hidden selector should return empty string")
	}

	// Visible - should have content
	scenes := []client.Scene{
		{ID: "scene-1", Metadata: client.Metadata{Name: "Relax"}, Group: client.ResourceIdentifier{RID: "room-1"}},
	}
	rooms := []client.Room{{ID: "room-1", Metadata: client.Metadata{Name: "Living Room"}}}
	s.Show(scenes, rooms)
	view = s.View()

	if !strings.Contains(view, "Select Scene") {
		t.Error("View should contain title")
	}
	if !strings.Contains(view, "Relax") {
		t.Error("View should contain scene names")
	}
	if !strings.Contains(view, "Living Room") {
		t.Error("View should contain room names")
	}
}
