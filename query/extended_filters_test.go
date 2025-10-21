package query

import (
	"testing"

	"github.com/kungfusheep/hue/client"
)

// Helper to make a light with specific brightness
func makeLightWithBrightness(id, name string, brightness float64, on bool) client.Light {
	return client.Light{
		ID: id,
		Metadata: client.Metadata{
			Name:      name,
			Archetype: "bulb",
		},
		On:      client.OnState{On: on},
		Dimming: client.Dimming{Brightness: brightness},
		Color:   &client.Color{XY: client.XY{X: 0.5, Y: 0.5}},
	}
}

// Helper to make a light with effect
func makeLightWithEffect(id, name, effect string) client.Light {
	light := makeLight(id, name, "bulb", true, true, true)
	if light.Effects != nil {
		light.Effects.Effect = effect
	}
	return light
}

// Helper to make a light with mode and gamut
func makeLightWithModeGamut(id, name, mode, gamut string) client.Light {
	return client.Light{
		ID: id,
		Metadata: client.Metadata{
			Name:      name,
			Archetype: "bulb",
		},
		On:   client.OnState{On: true},
		Mode: mode,
		Color: &client.Color{
			XY:        client.XY{X: 0.5, Y: 0.5},
			GamutType: gamut,
		},
	}
}

// TestEffectFilter tests effect: filter
func TestEffectFilter(t *testing.T) {
	lights := []client.Light{
		makeLightWithEffect("1", "Light 1", "fire"),
		makeLightWithEffect("2", "Light 2", "candle"),
		makeLightWithEffect("3", "Light 3", "no_effect"),
		makeLightWithEffect("4", "Light 4", ""),
		makeLightWithEffect("5", "Light 5", "sparkle"),
	}

	tests := []struct {
		query     string
		wantCount int
		wantIDs   []string
	}{
		{`@"effect:fire"`, 1, []string{"1"}},
		{`@"effect:candle"`, 1, []string{"2"}},
		{`@"effect:none"`, 2, []string{"3", "4"}},
		{`@"effect:*"`, 3, []string{"1", "2", "5"}}, // Any effect (not none)
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}

			gotIDs := make(map[string]bool)
			for _, light := range result {
				gotIDs[light.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotIDs[wantID] {
					t.Errorf("missing expected light ID: %s", wantID)
				}
			}
		})
	}
}

// TestBrightnessFilter tests brightness comparison filters
func TestBrightnessFilter(t *testing.T) {
	lights := []client.Light{
		makeLightWithBrightness("1", "Dim Light", 10.0, true),
		makeLightWithBrightness("2", "Medium Light", 50.0, true),
		makeLightWithBrightness("3", "Bright Light", 90.0, true),
		makeLightWithBrightness("4", "Very Bright", 100.0, true),
		makeLightWithBrightness("5", "Low Light", 25.0, true),
	}

	tests := []struct {
		query     string
		wantCount int
		wantIDs   []string
	}{
		{`@"brightness>50"`, 2, []string{"3", "4"}},
		{`@"brightness<30"`, 2, []string{"1", "5"}},
		{`@"brightness>80"`, 2, []string{"3", "4"}},
		{`@"brightness<20"`, 1, []string{"1"}},
		// Combined with other filters
		{`@"brightness>40 on:"`, 3, []string{"2", "3", "4"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
				for _, l := range result {
					t.Logf("  - %s (%.0f%%)", l.Metadata.Name, l.Dimming.Brightness)
				}
			}

			gotIDs := make(map[string]bool)
			for _, light := range result {
				gotIDs[light.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotIDs[wantID] {
					t.Errorf("missing expected light ID: %s", wantID)
				}
			}
		})
	}
}

// TestModeFilter tests mode: filter
func TestModeFilter(t *testing.T) {
	lights := []client.Light{
		makeLightWithModeGamut("1", "Normal 1", "normal", "C"),
		makeLightWithModeGamut("2", "Normal 2", "normal", "C"),
		makeLightWithModeGamut("3", "Streaming", "streaming", "C"),
	}

	tests := []struct {
		query     string
		wantCount int
		wantIDs   []string
	}{
		{`@"mode:normal"`, 2, []string{"1", "2"}},
		{`@"mode:streaming"`, 1, []string{"3"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// TestGamutFilter tests gamut: filter
func TestGamutFilter(t *testing.T) {
	lights := []client.Light{
		makeLightWithModeGamut("1", "Gamut A", "normal", "A"),
		makeLightWithModeGamut("2", "Gamut B", "normal", "B"),
		makeLightWithModeGamut("3", "Gamut C", "normal", "C"),
		makeLightWithModeGamut("4", "Gamut C 2", "normal", "C"),
	}

	tests := []struct {
		query     string
		wantCount int
		wantIDs   []string
	}{
		{`@"gamut:A"`, 1, []string{"1"}},
		{`@"gamut:B"`, 1, []string{"2"}},
		{`@"gamut:C"`, 2, []string{"3", "4"}},
		{`@"gamut:c"`, 2, []string{"3", "4"}}, // Case insensitive
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// TestIDFilter tests id: filter
func TestIDFilter(t *testing.T) {
	lights := []client.Light{
		makeLight("abc-123", "Light 1", "bulb", true, true, false),
		makeLight("abc-456", "Light 2", "bulb", true, true, false),
		makeLight("xyz-789", "Light 3", "bulb", true, true, false),
	}

	tests := []struct {
		query     string
		wantCount int
		wantIDs   []string
	}{
		{`@"id:abc*"`, 2, []string{"abc-123", "abc-456"}},
		{`@"id:*789"`, 1, []string{"xyz-789"}},
		{`@"id:abc-123"`, 1, []string{"abc-123"}},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// TestCombinedAdvancedFilters tests combining new filters
func TestCombinedAdvancedFilters(t *testing.T) {
	lights := []client.Light{
		makeLightWithBrightness("1", "Office Bright", 90.0, true),
		makeLightWithBrightness("2", "Office Dim", 20.0, true),
		makeLightWithBrightness("3", "Bedroom Bright", 85.0, true),
		makeLightWithBrightness("4", "Bedroom Dim", 15.0, true),
	}

	// Add effects to some
	lights[0].Effects = &client.Effects{Effect: "fire", EffectValues: []string{"fire", "candle"}}
	lights[2].Effects = &client.Effects{Effect: "candle", EffectValues: []string{"fire", "candle"}}

	tests := []struct {
		query       string
		wantCount   int
		wantIDs     []string
		description string
	}{
		{
			query:       `@"office brightness>50"`,
			wantCount:   1,
			wantIDs:     []string{"1"},
			description: "Office lights brighter than 50%",
		},
		{
			query:       `@"brightness<30"`,
			wantCount:   2,
			wantIDs:     []string{"2", "4"},
			description: "All dim lights",
		},
		{
			query:       `@"effect:* brightness>80"`,
			wantCount:   2,
			wantIDs:     []string{"1", "3"},
			description: "Lights with any effect and high brightness",
		},
		{
			query:       `@"bedroom brightness<50"`,
			wantCount:   1,
			wantIDs:     []string{"4"},
			description: "Dim bedroom lights",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("%s: got %d lights, want %d", tt.description, len(result), tt.wantCount)
				for _, l := range result {
					t.Logf("  - %s (%.0f%%)", l.Metadata.Name, l.Dimming.Brightness)
				}
			}
		})
	}
}
