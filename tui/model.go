package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/query"
)

// ActionHandler is called when the TUI needs to perform an action.
// This keeps the TUI decoupled from the actual Hue client.
type ActionHandler interface {
	// FetchLights retrieves all lights
	FetchLights() ([]client.Light, error)
	// FetchRooms retrieves all rooms (for query context)
	FetchRooms() ([]client.Room, error)
	// FetchScenes retrieves all scenes
	FetchScenes() ([]client.Scene, error)
	// ToggleLights toggles the specified lights on/off
	ToggleLights(ids []string) error
	// SetBrightness sets brightness for the specified lights
	SetBrightness(ids []string, brightness float64) error
	// SetColor sets color for the specified lights
	SetColor(ids []string, color string) error
	// SetEffect sets effect for the specified lights
	SetEffect(ids []string, effect string) error
	// ActivateScene activates a scene
	ActivateScene(id string) error
}

// EventProvider is an optional interface for handlers that support event streaming.
type EventProvider interface {
	// SubscribeEvents starts streaming events and returns a channel.
	// The returned cancel function stops the stream.
	SubscribeEvents() (events <-chan LightUpdateEvent, cancel func(), err error)
}

// LightUpdateEvent represents a light state update from SSE.
type LightUpdateEvent struct {
	LightID    string
	On         *bool
	Brightness *float64
	ColorXY    *struct{ X, Y float64 }
	Effect     *string
}

// Model is the main TUI model that composes all components.
type Model struct {
	// Components
	query   QueryInput
	lights  LightList
	effects EffectSelector
	colors  ColorSelector
	scenes  SceneSelector

	// State
	allLights    []client.Light
	allScenes    []client.Scene
	rooms        []client.Room
	status       string
	err          error
	liveTracking bool // Whether SSE is active

	// Config
	styles  Styles
	keys    KeyMap
	handler ActionHandler

	// Dimensions
	width  int
	height int
	ready  bool

	// Event streaming
	cancelEvents func()
}

// NewModel creates a new TUI model with the given action handler.
func NewModel(handler ActionHandler) Model {
	styles := DefaultStyles()
	keys := DefaultKeyMap()

	q := NewQueryInput(styles)

	m := Model{
		query:   q,
		lights:  NewLightList(styles, keys),
		effects: NewEffectSelector(styles, keys),
		colors:  NewColorSelector(styles, keys),
		scenes:  NewSceneSelector(styles, keys),
		styles:  styles,
		keys:    keys,
		handler: handler,
		status:  "Loading...",
	}
	m.query.Focus() // Start with query focused
	return m
}

// Init initialises the model and starts loading data.
func (m Model) Init() tea.Cmd {
	cmds := []tea.Cmd{
		m.loadLights(),
		m.loadRooms(),
		m.loadScenes(),
		m.query.Focus(), // Focus query input by default
	}

	// Start event streaming if supported
	if ep, ok := m.handler.(EventProvider); ok {
		cmds = append(cmds, m.subscribeToEvents(ep))
	}

	return tea.Batch(cmds...)
}

// Update handles all messages.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// If effect selector is visible, route to it
		if m.effects.Visible() {
			var cmd tea.Cmd
			m.effects, cmd = m.effects.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// If color selector is visible, route to it
		if m.colors.Visible() {
			var cmd tea.Cmd
			m.colors, cmd = m.colors.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// If scene selector is visible, route to it
		if m.scenes.Visible() {
			var cmd tea.Cmd
			m.scenes, cmd = m.scenes.Update(msg)
			if cmd != nil {
				cmds = append(cmds, cmd)
			}
			return m, tea.Batch(cmds...)
		}

		// Global keys
		switch {
		case key.Matches(msg, m.keys.Quit):
			return m, tea.Quit
		case key.Matches(msg, m.keys.FocusQuery):
			if !m.query.Focused() {
				m.lights.Blur()
				cmds = append(cmds, m.query.Focus())
			}
			return m, tea.Batch(cmds...)
		case key.Matches(msg, m.keys.Effect):
			// Open effect selector for selected/cursor lights (only when lights focused)
			if m.lights.Focused() {
				ids := m.lights.EffectiveSelection()
				if len(ids) > 0 {
					supported := m.getSupportedEffects(ids)
					m.effects.Show(ids, supported)
				}
				return m, nil
			}
		case key.Matches(msg, m.keys.Color):
			// Open color selector for selected/cursor lights (only when lights focused)
			if m.lights.Focused() {
				ids := m.lights.EffectiveSelection()
				if len(ids) > 0 {
					// Check if selected lights support color
					if m.lightsHaveColor(ids) {
						m.colors.Show(ids)
					} else {
						m.status = "Selected lights don't support color"
					}
				}
				return m, nil
			}
		case key.Matches(msg, m.keys.Scene):
			// Open scene selector (only when lights focused)
			if m.lights.Focused() {
				filteredScenes := m.getScenesForCurrentLights()
				if len(filteredScenes) > 0 {
					m.scenes.Show(filteredScenes, m.rooms)
				} else {
					m.status = "No scenes for current lights"
				}
				return m, nil
			}
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.updateSizes()

	case LightsLoadedMsg:
		if msg.Err != nil {
			m.err = msg.Err
			m.status = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.allLights = msg.Lights
			// Preserve cursor position and re-apply filter
			m.lights.UpdateLights(msg.Lights)
			// Re-apply current query filter
			currentQuery := m.query.Value()
			if currentQuery != "" {
				filtered := m.filterLights(currentQuery)
				m.lights.SetFiltered(filtered)
				m.status = fmt.Sprintf("Matched %d lights", len(filtered))
			} else {
				m.status = fmt.Sprintf("Loaded %d lights", len(msg.Lights))
			}
			// Don't steal focus from query input
		}

	case RoomsLoadedMsg:
		if msg.Err != nil {
			// Non-fatal, just log
			m.rooms = nil
		} else {
			m.rooms = msg.Rooms
		}

	case QueryChangedMsg:
		// Re-filter lights based on query
		filtered := m.filterLights(msg.Query)
		m.lights.SetFiltered(filtered)
		m.status = fmt.Sprintf("Matched %d lights", len(filtered))

	case QuerySubmittedMsg:
		// Query submitted - switch focus to lights
		m.query.Blur()
		m.lights.Focus()
		m.status = fmt.Sprintf("Query: %s (%d matches)", msg.Query, len(m.lights.filtered))

	case FocusChangedMsg:
		switch msg.Component {
		case "query":
			m.lights.Blur()
			cmds = append(cmds, m.query.Focus())
		case "lights":
			m.query.Blur()
			m.lights.Focus()
		}

	case LightToggledMsg:
		cmds = append(cmds, m.toggleLights(msg.LightIDs))

	case EffectSelectedMsg:
		cmds = append(cmds, m.setEffect(msg.LightIDs, msg.Effect))
		m.status = fmt.Sprintf("Applying effect '%s'...", msg.Effect)

	case BrightnessAdjustMsg:
		cmds = append(cmds, m.adjustBrightness(msg.LightIDs, msg.Delta))
		direction := "Increasing"
		if msg.Delta < 0 {
			direction = "Decreasing"
		}
		m.status = fmt.Sprintf("%s brightness...", direction)

	case EffectSelectorClosedMsg:
		// Just return focus to lights
		m.lights.Focus()

	case ColorSelectedMsg:
		cmds = append(cmds, m.setColor(msg.LightIDs, msg.Color))
		m.status = fmt.Sprintf("Applying color %s...", msg.Color)

	case ColorSelectorClosedMsg:
		// Just return focus to lights
		m.lights.Focus()

	case ScenesLoadedMsg:
		if msg.Err == nil {
			m.allScenes = msg.Scenes
		}

	case SceneSelectedMsg:
		cmds = append(cmds, m.activateScene(msg.SceneID, msg.SceneName))
		m.status = fmt.Sprintf("Activating scene '%s'...", msg.SceneName)
		// Keep popup open so user can try multiple scenes

	case SceneSelectorClosedMsg:
		m.lights.Focus()

	case ActionCompletedMsg:
		if msg.Err != nil {
			m.status = fmt.Sprintf("Error: %v", msg.Err)
		} else {
			m.status = fmt.Sprintf("%s completed", msg.Action)
			// Refresh lights after action
			cmds = append(cmds, m.loadLights())
		}

	case StatusMsg:
		m.status = msg.Message

	case ErrorMsg:
		m.err = msg.Err
		m.status = fmt.Sprintf("Error: %v", msg.Err)

	case eventStreamStartedMsg:
		m.liveTracking = true
		m.cancelEvents = msg.cancel
		// Start listening for the first event
		cmds = append(cmds, waitForEventCmd(msg.events))

	case lightUpdateWithChanMsg:
		m.applyLightUpdate(msg.event)
		// Continue listening for more events
		if m.liveTracking {
			cmds = append(cmds, waitForEventCmd(msg.events))
		}
	}

	// Update focused component
	if m.query.Focused() {
		var cmd tea.Cmd
		m.query, cmd = m.query.Update(msg)
		cmds = append(cmds, cmd)
	} else if m.lights.Focused() {
		var cmd tea.Cmd
		m.lights, cmd = m.lights.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

// View renders the TUI.
func (m Model) View() string {
	if !m.ready {
		return "Loading..."
	}

	var sections []string

	// Header with live tracking indicator
	headerText := "HUE CONTROL CENTER"
	if m.liveTracking {
		headerText += " [LIVE]"
	}
	header := m.styles.Header.Render(headerText)
	sections = append(sections, header)

	// Query input
	sections = append(sections, m.query.View())
	sections = append(sections, "") // Spacer

	// Light list
	sections = append(sections, m.lights.View())

	// Status bar
	status := m.styles.StatusBar.Render(m.status)
	sections = append(sections, status)

	// Help
	help := m.renderHelp()
	sections = append(sections, help)

	mainView := m.styles.App.Render(strings.Join(sections, "\n"))

	// Overlay effect selector if visible
	if m.effects.Visible() {
		effectView := m.effects.View()
		// Center the modal
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			effectView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	// Overlay color selector if visible
	if m.colors.Visible() {
		colorView := m.colors.View()
		// Center the modal
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			colorView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	// Overlay scene selector if visible
	if m.scenes.Visible() {
		sceneView := m.scenes.View()
		// Center the modal
		return lipgloss.Place(
			m.width, m.height,
			lipgloss.Center, lipgloss.Center,
			sceneView,
			lipgloss.WithWhitespaceChars(" "),
			lipgloss.WithWhitespaceForeground(lipgloss.Color("0")),
		)
	}

	return mainView
}

func (m *Model) updateSizes() {
	// Adjust component sizes based on terminal - use full width
	contentWidth := m.width - 6 // Account for padding

	m.query.SetWidth(contentWidth)
	m.lights.SetSize(contentWidth, m.height-12) // Leave room for header, query, status, help
}

func (m Model) filterLights(queryStr string) []client.Light {
	if queryStr == "" {
		return m.allLights
	}

	// Parse and execute query
	q, err := query.Parse(queryStr)
	if err != nil {
		// Invalid query - return all
		return m.allLights
	}

	// Build room context if needed
	var ctx *query.Context
	if query.NeedsRoomContext(queryStr) && len(m.rooms) > 0 {
		ctx = &query.Context{
			Rooms: m.rooms,
		}
	}

	return query.ExecuteWithContext(q, m.allLights, ctx)
}

func (m Model) loadLights() tea.Cmd {
	return func() tea.Msg {
		lights, err := m.handler.FetchLights()
		return LightsLoadedMsg{Lights: lights, Err: err}
	}
}

func (m Model) loadRooms() tea.Cmd {
	return func() tea.Msg {
		rooms, err := m.handler.FetchRooms()
		return RoomsLoadedMsg{Rooms: rooms, Err: err}
	}
}

func (m Model) loadScenes() tea.Cmd {
	return func() tea.Msg {
		scenes, err := m.handler.FetchScenes()
		return ScenesLoadedMsg{Scenes: scenes, Err: err}
	}
}

func (m Model) activateScene(id, name string) tea.Cmd {
	return func() tea.Msg {
		err := m.handler.ActivateScene(id)
		return ActionCompletedMsg{Action: fmt.Sprintf("Scene '%s'", name), Err: err}
	}
}

// eventStreamStartedMsg is sent when event streaming starts successfully
type eventStreamStartedMsg struct {
	cancel func()
	events <-chan LightUpdateEvent
}

// lightUpdateWithChanMsg wraps a light update with the channel for continued listening
type lightUpdateWithChanMsg struct {
	event  LightUpdateEvent
	events <-chan LightUpdateEvent
}

// subscribeToEvents starts the event stream and returns a command
func (m Model) subscribeToEvents(ep EventProvider) tea.Cmd {
	return func() tea.Msg {
		events, cancel, err := ep.SubscribeEvents()
		if err != nil {
			return StatusMsg{Message: "Live tracking unavailable"}
		}

		return eventStreamStartedMsg{
			cancel: cancel,
			events: events,
		}
	}
}

// waitForEventCmd creates a command that reads from the event channel
func waitForEventCmd(events <-chan LightUpdateEvent) tea.Cmd {
	return func() tea.Msg {
		event, ok := <-events
		if !ok {
			return StatusMsg{Message: "Event stream closed"}
		}
		return lightUpdateWithChanMsg{
			event:  event,
			events: events,
		}
	}
}

// applyLightUpdate applies an SSE event to the local light state
func (m *Model) applyLightUpdate(event LightUpdateEvent) {
	// Find and update the light
	for i := range m.allLights {
		if m.allLights[i].ID == event.LightID {
			if event.On != nil {
				m.allLights[i].On.On = *event.On
			}
			if event.Brightness != nil {
				m.allLights[i].Dimming.Brightness = *event.Brightness
			}
			if event.ColorXY != nil && m.allLights[i].Color != nil {
				m.allLights[i].Color.XY.X = event.ColorXY.X
				m.allLights[i].Color.XY.Y = event.ColorXY.Y
			}
			if event.Effect != nil && m.allLights[i].Effects != nil {
				m.allLights[i].Effects.Effect = *event.Effect
			}
			break
		}
	}

	// Update the displayed list
	m.lights.UpdateLights(m.allLights)
}

func (m Model) toggleLights(ids []string) tea.Cmd {
	return func() tea.Msg {
		err := m.handler.ToggleLights(ids)
		return ActionCompletedMsg{Action: "Toggle", Err: err}
	}
}

func (m Model) setEffect(ids []string, effect string) tea.Cmd {
	return func() tea.Msg {
		err := m.handler.SetEffect(ids, effect)
		action := "Effect"
		if effect == "no_effect" {
			action = "Clear effect"
		}
		return ActionCompletedMsg{Action: action, Err: err}
	}
}

func (m Model) setColor(ids []string, color string) tea.Cmd {
	return func() tea.Msg {
		err := m.handler.SetColor(ids, color)
		return ActionCompletedMsg{Action: "Color", Err: err}
	}
}

// getScenesForCurrentLights returns scenes that belong to rooms containing
// the currently filtered lights.
func (m Model) getScenesForCurrentLights() []client.Scene {
	if len(m.allScenes) == 0 {
		return nil
	}

	// Get the filtered lights (or all if no filter)
	lightsToCheck := m.lights.filtered
	if len(lightsToCheck) == 0 {
		lightsToCheck = m.allLights
	}

	// Build a set of device IDs that own our lights
	lightDevices := make(map[string]bool)
	for _, light := range lightsToCheck {
		lightDevices[light.Owner.RID] = true
	}

	// Find which rooms contain these devices
	lightRooms := make(map[string]bool)
	for _, room := range m.rooms {
		for _, child := range room.Children {
			// Room children are devices - check if any of our lights belong to this device
			if lightDevices[child.RID] {
				lightRooms[room.ID] = true
				break
			}
		}
	}

	// If no room context, show all scenes
	if len(lightRooms) == 0 {
		return m.allScenes
	}

	// Filter scenes to those belonging to the relevant rooms
	var filtered []client.Scene
	for _, scene := range m.allScenes {
		if lightRooms[scene.Group.RID] {
			filtered = append(filtered, scene)
		}
	}

	return filtered
}

// lightsHaveColor checks if at least one of the specified lights supports color.
func (m Model) lightsHaveColor(ids []string) bool {
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	for _, light := range m.allLights {
		if idSet[light.ID] && light.Color != nil {
			return true
		}
	}
	return false
}

func (m Model) adjustBrightness(ids []string, delta float64) tea.Cmd {
	// Calculate average current brightness of selected lights
	var totalBrightness float64
	var count int

	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	for _, light := range m.allLights {
		if idSet[light.ID] {
			totalBrightness += light.Dimming.Brightness
			count++
		}
	}

	if count == 0 {
		return nil
	}

	avgBrightness := totalBrightness / float64(count)
	newBrightness := avgBrightness + delta

	// Clamp to valid range (0-100)
	if newBrightness < 0 {
		newBrightness = 0
	}
	if newBrightness > 100 {
		newBrightness = 100
	}

	return func() tea.Msg {
		err := m.handler.SetBrightness(ids, newBrightness)
		return ActionCompletedMsg{Action: fmt.Sprintf("Brightness %.0f%%", newBrightness), Err: err}
	}
}

// getSupportedEffects returns effects supported by all the given lights.
func (m Model) getSupportedEffects(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}

	// Build a map of light IDs for quick lookup
	idSet := make(map[string]bool)
	for _, id := range ids {
		idSet[id] = true
	}

	// Find common effects across all selected lights
	var commonEffects map[string]bool

	for _, light := range m.allLights {
		if !idSet[light.ID] {
			continue
		}

		if light.Effects == nil || len(light.Effects.EffectValues) == 0 {
			// This light doesn't support effects - return empty
			return nil
		}

		if commonEffects == nil {
			// First light - initialize with its effects
			commonEffects = make(map[string]bool)
			for _, e := range light.Effects.EffectValues {
				commonEffects[e] = true
			}
		} else {
			// Intersect with this light's effects
			newCommon := make(map[string]bool)
			for _, e := range light.Effects.EffectValues {
				if commonEffects[e] {
					newCommon[e] = true
				}
			}
			commonEffects = newCommon
		}
	}

	if commonEffects == nil {
		return nil
	}

	var result []string
	for e := range commonEffects {
		result = append(result, e)
	}
	return result
}

func (m Model) renderHelp() string {
	keys := []struct {
		key  string
		desc string
	}{
		{"j/k", "navigate"},
		{"space", "select"},
		{"o", "on/off"},
		{"[/]", "brightness"},
		{"c", "color"},
		{"e", "effect"},
		{"s", "scene"},
		{"/", "search"},
		{"q", "quit"},
	}

	var parts []string
	for _, k := range keys {
		part := m.styles.HelpKey.Render(k.key) + " " + m.styles.HelpDesc.Render(k.desc)
		parts = append(parts, part)
	}

	return lipgloss.JoinHorizontal(lipgloss.Center, strings.Join(parts, "  "))
}
