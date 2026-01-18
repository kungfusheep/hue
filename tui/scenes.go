package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/kungfusheep/hue/client"
)

// SceneItem represents a scene option in the selector.
type SceneItem struct {
	ID        string
	Name      string
	GroupName string // Room/Zone name
}

// SceneSelector is a modal component for selecting and activating scenes.
type SceneSelector struct {
	items   []SceneItem
	cursor  int
	styles  Styles
	keys    KeyMap
	visible bool
	width   int
	height  int
	loading bool
}

// NewSceneSelector creates a new scene selector component.
func NewSceneSelector(styles Styles, keys KeyMap) SceneSelector {
	return SceneSelector{
		items:  nil,
		cursor: 0,
		styles: styles,
		keys:   keys,
		width:  60,
		height: 20,
	}
}

// Init initialises the component.
func (s SceneSelector) Init() tea.Cmd {
	return nil
}

// Update handles messages and returns commands.
func (s SceneSelector) Update(msg tea.Msg) (SceneSelector, tea.Cmd) {
	if !s.visible {
		return s, nil
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case key.Matches(msg, s.keys.Up):
			s.moveUp()
		case key.Matches(msg, s.keys.Down):
			s.moveDown()
		case key.Matches(msg, s.keys.Top):
			s.cursor = 0
		case key.Matches(msg, s.keys.Bottom):
			if len(s.items) > 0 {
				s.cursor = len(s.items) - 1
			}
		case key.Matches(msg, s.keys.Enter):
			return s, s.selectScene()
		case key.Matches(msg, s.keys.Escape):
			s.visible = false
			return s, func() tea.Msg {
				return SceneSelectorClosedMsg{}
			}
		}
	}

	return s, nil
}

// View renders the component.
func (s SceneSelector) View() string {
	if !s.visible {
		return ""
	}

	var lines []string

	// Title
	title := s.styles.Title.Render("Select Scene")
	lines = append(lines, title)
	lines = append(lines, "")

	if s.loading {
		lines = append(lines, s.styles.Subtle.Render("Loading scenes..."))
	} else if len(s.items) == 0 {
		lines = append(lines, s.styles.Subtle.Render("No scenes available"))
	} else {
		// Group scenes by room
		currentGroup := ""
		for i, item := range s.items {
			// Add group header if changed
			if item.GroupName != currentGroup {
				if currentGroup != "" {
					lines = append(lines, "") // Spacer between groups
				}
				currentGroup = item.GroupName
				groupHeader := s.styles.Subtle.Render(fmt.Sprintf("── %s ──", item.GroupName))
				lines = append(lines, groupHeader)
			}

			line := s.renderItem(item, i)
			lines = append(lines, line)
		}
	}

	lines = append(lines, "")

	// Help
	help := s.styles.Subtle.Render("j/k navigate • enter activate • esc cancel")
	lines = append(lines, help)

	content := strings.Join(lines, "\n")

	// Modal style
	modal := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("69")).
		Padding(1, 2).
		Width(s.width)

	return modal.Render(content)
}

func (s SceneSelector) renderItem(item SceneItem, idx int) string {
	cursor := "  "
	if idx == s.cursor {
		cursor = s.styles.Accent.Render("▸ ")
	}

	// Style based on selection
	var nameStyle lipgloss.Style
	if idx == s.cursor {
		nameStyle = s.styles.LightSelected
	} else {
		nameStyle = s.styles.LightUnselected
	}

	nameStr := nameStyle.Render(item.Name)

	return fmt.Sprintf("%s%s", cursor, nameStr)
}

// Show displays the selector with the given scenes.
func (s *SceneSelector) Show(scenes []client.Scene, rooms []client.Room) {
	s.visible = true
	s.loading = false

	// Build room lookup
	roomNames := make(map[string]string)
	for _, room := range rooms {
		roomNames[room.ID] = room.Metadata.Name
	}

	// Convert scenes to items, sorted by room
	s.items = make([]SceneItem, 0, len(scenes))
	for _, scene := range scenes {
		groupName := "Unknown"
		if name, ok := roomNames[scene.Group.RID]; ok {
			groupName = name
		}

		s.items = append(s.items, SceneItem{
			ID:        scene.ID,
			Name:      scene.Metadata.Name,
			GroupName: groupName,
		})
	}

	// Sort by group name, then scene name
	sortSceneItems(s.items)

	s.cursor = 0
}

// sortSceneItems sorts scene items by group name, then by scene name.
func sortSceneItems(items []SceneItem) {
	// Simple bubble sort - scenes list is typically small
	for i := 0; i < len(items)-1; i++ {
		for j := 0; j < len(items)-i-1; j++ {
			// Compare group names first
			if items[j].GroupName > items[j+1].GroupName {
				items[j], items[j+1] = items[j+1], items[j]
			} else if items[j].GroupName == items[j+1].GroupName {
				// Same group, compare scene names
				if items[j].Name > items[j+1].Name {
					items[j], items[j+1] = items[j+1], items[j]
				}
			}
		}
	}
}

// Hide hides the selector.
func (s *SceneSelector) Hide() {
	s.visible = false
}

// Visible returns whether the selector is visible.
func (s SceneSelector) Visible() bool {
	return s.visible
}

// SetSize sets the dimensions.
func (s *SceneSelector) SetSize(width, height int) {
	s.width = width
	if s.width > 70 {
		s.width = 70
	}
	s.height = height
}

func (s *SceneSelector) moveUp() {
	if s.cursor > 0 {
		s.cursor--
	}
}

func (s *SceneSelector) moveDown() {
	if s.cursor < len(s.items)-1 {
		s.cursor++
	}
}

func (s *SceneSelector) selectScene() tea.Cmd {
	if s.cursor < 0 || s.cursor >= len(s.items) {
		return nil
	}

	item := s.items[s.cursor]

	return func() tea.Msg {
		return SceneSelectedMsg{
			SceneID:   item.ID,
			SceneName: item.Name,
		}
	}
}

// SceneSelectedMsg is sent when a scene is selected.
type SceneSelectedMsg struct {
	SceneID   string
	SceneName string
}

// SceneSelectorClosedMsg is sent when the selector is closed without selection.
type SceneSelectorClosedMsg struct{}
