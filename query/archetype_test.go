package query

import (
	"testing"

	"github.com/kungfusheep/hue/client"
)

// TestArchetypeMatching tests that fuzzy search checks both name and archetype
func TestArchetypeMatching(t *testing.T) {
	lights := []client.Light{
		makeLight("1", "Living Room Lamp", "table_lamp", true, true, false),
		makeLight("2", "Kitchen Light", "sultan_bulb", true, true, true),
		makeLight("3", "Office Ceiling", "ceiling_round", true, true, true),
		makeLight("4", "Bedroom Go", "hue_go", false, true, true),
	}

	tests := []struct {
		name        string
		query       string
		wantCount   int
		wantLights  []string
		description string
	}{
		{
			name:        "match by archetype only",
			query:       `@"sultan"`,
			wantCount:   1,
			wantLights:  []string{"Kitchen Light"},
			description: "Should find light with sultan_bulb archetype",
		},
		{
			name:        "match lamp in name or archetype",
			query:       `@"lamp"`,
			wantCount:   1,
			wantLights:  []string{"Living Room Lamp"},
			description: "Should find lights with 'lamp' in name OR archetype (table_lamp)",
		},
		{
			name:        "explicit name only",
			query:       `@"name:lamp"`,
			wantCount:   1,
			wantLights:  []string{"Living Room Lamp"},
			description: "name: prefix should only check name field",
		},
		{
			name:        "explicit type only",
			query:       `@"type:lamp"`,
			wantCount:   1,
			wantLights:  []string{"Living Room Lamp"},
			description: "type: prefix should only check archetype field (table_lamp)",
		},
		{
			name:        "match go in name or archetype",
			query:       `@"go"`,
			wantCount:   1,
			wantLights:  []string{"Bedroom Go"},
			description: "Should find light with 'go' in name or hue_go archetype",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("%s: got %d lights, want %d", tt.description, len(result), tt.wantCount)
				t.Logf("Matched:")
				for _, light := range result {
					t.Logf("  - %s (%s)", light.Metadata.Name, light.Metadata.Archetype)
				}
			}

			// Check expected lights are in result
			resultNames := make(map[string]bool)
			for _, light := range result {
				resultNames[light.Metadata.Name] = true
			}

			for _, wantName := range tt.wantLights {
				if !resultNames[wantName] {
					t.Errorf("%s: expected %q in results but not found", tt.description, wantName)
				}
			}
		})
	}
}
