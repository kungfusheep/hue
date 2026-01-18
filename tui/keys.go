package tui

import "github.com/charmbracelet/bubbles/key"

// KeyMap defines all keybindings for the TUI.
// Components can use subsets of these.
type KeyMap struct {
	// Navigation
	Up     key.Binding
	Down   key.Binding
	Left   key.Binding
	Right  key.Binding
	Top    key.Binding
	Bottom key.Binding

	// Selection
	Select    key.Binding
	SelectAll key.Binding

	// Actions
	Toggle         key.Binding // On/off
	BrightnessUp   key.Binding
	BrightnessDown key.Binding
	Color          key.Binding
	Effect         key.Binding
	Scene          key.Binding

	// Query
	ClearQuery key.Binding
	FocusQuery key.Binding

	// General
	Help   key.Binding
	Quit   key.Binding
	Escape key.Binding
	Enter  key.Binding
	Tab    key.Binding
}

// DefaultKeyMap returns the default keybindings.
func DefaultKeyMap() KeyMap {
	return KeyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "left"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "right"),
		),
		Top: key.NewBinding(
			key.WithKeys("g", "home"),
			key.WithHelp("g", "top"),
		),
		Bottom: key.NewBinding(
			key.WithKeys("G", "end"),
			key.WithHelp("G", "bottom"),
		),
		Select: key.NewBinding(
			key.WithKeys(" "),
			key.WithHelp("space", "select"),
		),
		SelectAll: key.NewBinding(
			key.WithKeys("a"),
			key.WithHelp("a", "select all"),
		),
		Toggle: key.NewBinding(
			key.WithKeys("o"),
			key.WithHelp("o", "on/off"),
		),
		BrightnessUp: key.NewBinding(
			key.WithKeys("]"),
			key.WithHelp("]", "brighter"),
		),
		BrightnessDown: key.NewBinding(
			key.WithKeys("["),
			key.WithHelp("[", "dimmer"),
		),
		Color: key.NewBinding(
			key.WithKeys("c"),
			key.WithHelp("c", "color"),
		),
		Effect: key.NewBinding(
			key.WithKeys("e"),
			key.WithHelp("e", "effect"),
		),
		Scene: key.NewBinding(
			key.WithKeys("s"),
			key.WithHelp("s", "scene"),
		),
		ClearQuery: key.NewBinding(
			key.WithKeys("ctrl+u"),
			key.WithHelp("ctrl+u", "clear"),
		),
		FocusQuery: key.NewBinding(
			key.WithKeys("/"),
			key.WithHelp("/", "search"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
		Escape: key.NewBinding(
			key.WithKeys("esc"),
			key.WithHelp("esc", "back"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "confirm"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "next"),
		),
	}
}

// ShortHelp returns a condensed help string.
func (k KeyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Up, k.Down, k.Toggle, k.FocusQuery, k.Help, k.Quit}
}

// FullHelp returns all keybindings grouped.
func (k KeyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right, k.Top, k.Bottom},
		{k.Select, k.SelectAll, k.Toggle},
		{k.BrightnessDown, k.BrightnessUp, k.Color, k.Effect, k.Scene},
		{k.FocusQuery, k.ClearQuery, k.Help, k.Quit},
	}
}
