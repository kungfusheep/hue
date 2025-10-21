package query

import "github.com/kungfusheep/hue/client"

// GetTestLights returns a realistic set of lights for testing
// Anonymized but representative of typical home setups
func GetTestLights() []client.Light {
	return []client.Light{
		// Office - mixed types
		makeLight("office-1", "Office Desk Lamp", "desk_lamp", true, true, false),
		makeLight("office-2", "Office Ceiling", "ceiling_round", true, true, true),
		makeLight("office-3", "Office Strip", "hue_lightstrip", false, true, true),
		makeLight("office-4", "Office Hue Go", "hue_go", true, true, true),

		// Bedroom - various states
		makeLight("bedroom-1", "Bedroom Lamp 1", "table_lamp", false, true, false),
		makeLight("bedroom-2", "Bedroom Lamp 2", "table_lamp", false, true, false),
		makeLight("bedroom-3", "Bedroom Ceiling", "ceiling_round", true, true, true),
		makeLight("bedroom-4", "Bedroom Strip", "hue_lightstrip", false, true, true),

		// Kitchen - mostly on
		makeLight("kitchen-1", "Kitchen Counter Left", "hue_lightstrip", true, true, true),
		makeLight("kitchen-2", "Kitchen Counter Right", "hue_lightstrip", true, true, true),
		makeLight("kitchen-3", "Kitchen Ceiling", "ceiling_round", true, true, true),
		makeLight("kitchen-4", "Kitchen Pendant", "pendant_round", true, true, false),

		// Living Room - mixed
		makeLight("living-1", "Living Room Floor Lamp", "floor_lamp", true, true, false),
		makeLight("living-2", "Living Room Table Lamp", "table_lamp", false, true, false),
		makeLight("living-3", "Living Room Strip", "hue_lightstrip", true, true, true),
		makeLight("living-4", "Living Room Hue Go 1", "hue_go", true, true, true),
		makeLight("living-5", "Living Room Hue Go 2", "hue_go", false, true, true),

		// Hallway - simple setup
		makeLight("hall-1", "Hallway Light 1", "ceiling_round", false, false, false),
		makeLight("hall-2", "Hallway Light 2", "ceiling_round", false, false, false),

		// Bathroom - white only
		makeLight("bath-1", "Bathroom Ceiling", "ceiling_round", false, false, false),
		makeLight("bath-2", "Bathroom Mirror", "single_spot", false, false, false),

		// Outdoor/Patio
		makeLight("outdoor-1", "Patio Strip", "hue_lightstrip", false, true, true),
		makeLight("outdoor-2", "Garden Spot 1", "bollard", false, true, true),
		makeLight("outdoor-3", "Garden Spot 2", "bollard", false, true, true),
	}
}

// GetTestLightsByRoom returns lights grouped by room for room-based tests
func GetTestLightsByRoom() map[string][]client.Light {
	lights := GetTestLights()
	rooms := make(map[string][]client.Light)

	for _, light := range lights {
		// Extract room from name (simplified - in reality would use room metadata)
		var room string
		name := light.Metadata.Name
		switch {
		case contains(name, "Office"):
			room = "Office"
		case contains(name, "Bedroom"):
			room = "Bedroom"
		case contains(name, "Kitchen"):
			room = "Kitchen"
		case contains(name, "Living"):
			room = "Living Room"
		case contains(name, "Hallway"):
			room = "Hallway"
		case contains(name, "Bathroom"):
			room = "Bathroom"
		case contains(name, "Patio") || contains(name, "Garden"):
			room = "Outdoor"
		default:
			room = "Unknown"
		}
		rooms[room] = append(rooms[room], light)
	}

	return rooms
}

// contains is a helper for simple substring check
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
