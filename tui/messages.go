package tui

import (
	"github.com/kungfusheep/hue/client"
)

// Messages are how components communicate with the parent model.
// The parent (Model) handles these and dispatches actions via the CLI/client.

// LightsLoadedMsg is sent when lights have been fetched.
type LightsLoadedMsg struct {
	Lights []client.Light
	Err    error
}

// RoomsLoadedMsg is sent when rooms have been fetched.
type RoomsLoadedMsg struct {
	Rooms []client.Room
	Err   error
}

// ScenesLoadedMsg is sent when scenes have been fetched.
type ScenesLoadedMsg struct {
	Scenes []client.Scene
	Err    error
}

// QueryChangedMsg is sent when the query input changes.
type QueryChangedMsg struct {
	Query string
}

// QuerySubmittedMsg is sent when the user presses enter on the query.
type QuerySubmittedMsg struct {
	Query string
}

// LightSelectedMsg is sent when a light is selected/deselected.
type LightSelectedMsg struct {
	LightID  string
	Selected bool
}

// LightToggledMsg requests toggling a light on/off.
type LightToggledMsg struct {
	LightIDs []string
}

// BrightnessChangedMsg requests changing brightness.
type BrightnessChangedMsg struct {
	LightIDs   []string
	Brightness float64 // 0.0 - 100.0
}

// BrightnessAdjustMsg requests adjusting brightness by a delta.
type BrightnessAdjustMsg struct {
	LightIDs []string
	Delta    float64 // -100 to +100
}

// ColorChangedMsg requests changing color.
type ColorChangedMsg struct {
	LightIDs []string
	Color    string // hex color
}

// EffectChangedMsg requests changing effect.
type EffectChangedMsg struct {
	LightIDs []string
	Effect   string
}

// ActionCompletedMsg is sent when an action finishes.
type ActionCompletedMsg struct {
	Action string
	Err    error
}

// FocusChangedMsg signals focus should move to a different component.
type FocusChangedMsg struct {
	Component string // "query", "lights", "actions", etc.
}

// ErrorMsg represents an error to display.
type ErrorMsg struct {
	Err error
}

// StatusMsg represents a status message to display.
type StatusMsg struct {
	Message string
}

// TickMsg is sent for periodic updates (e.g., live tracking).
type TickMsg struct{}

// LightStateUpdateMsg is sent when light state changes (from SSE).
type LightStateUpdateMsg struct {
	LightID string
	Light   client.Light
}
