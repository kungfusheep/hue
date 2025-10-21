package query

import (
	"strings"
	"testing"
)

// TestRealisticQueries tests real-world query scenarios
func TestRealisticQueries(t *testing.T) {
	lights := GetTestLights()

	tests := []struct {
		name          string
		query         string
		wantCount     int
		wantContains  []string // light names that should be in results
		wantExcludes  []string // light names that should NOT be in results
		description   string
	}{
		{
			name:         "all office lights",
			query:        `@"office"`,
			wantCount:    4,
			wantContains: []string{"Office Desk Lamp", "Office Ceiling", "Office Strip", "Office Hue Go"},
			description:  "Should match all lights with 'office' in name",
		},
		{
			name:         "all lamps regardless of room",
			query:        `@"*lamp"`,
			wantCount:    5,
			wantContains: []string{"Office Desk Lamp", "Bedroom Lamp 1", "Bedroom Lamp 2", "Living Room Floor Lamp", "Living Room Table Lamp"},
			wantExcludes: []string{"Office Ceiling", "Kitchen Counter"},
			description:  "Should match lights with 'lamp' in name OR archetype",
		},
		{
			name:         "all hue go lights",
			query:        `@"type:go"`,
			wantCount:    3,
			wantContains: []string{"Office Hue Go", "Living Room Hue Go 1", "Living Room Hue Go 2"},
			wantExcludes: []string{"Office Desk Lamp", "Kitchen"},
			description:  "Should match all hue_go archetype lights",
		},
		{
			name:         "all strip lights",
			query:        `@"type:strip"`,
			wantCount:    6,
			wantContains: []string{"Office Strip", "Bedroom Strip", "Kitchen Counter Left", "Kitchen Counter Right", "Living Room Strip", "Patio Strip"},
			description:  "Should match all lightstrip archetype lights",
		},
		{
			name:         "all currently on lights",
			query:        `@"on:"`,
			wantCount:    11,
			wantContains: []string{"Office Desk Lamp", "Kitchen Ceiling", "Living Room Floor Lamp"},
			wantExcludes: []string{"Bedroom Lamp 1", "Hallway Light 1"},
			description:  "Should match only lights that are currently on",
		},
		{
			name:         "all currently off lights",
			query:        `@"off:"`,
			wantCount:    13,
			wantContains: []string{"Bedroom Lamp 1", "Hallway Light 1", "Bathroom Ceiling"},
			wantExcludes: []string{"Office Desk Lamp", "Kitchen Ceiling"},
			description:  "Should match only lights that are currently off",
		},
		{
			name:         "lights that support effects",
			query:        `@"with:effects"`,
			wantCount:    14,
			wantContains: []string{"Office Ceiling", "Office Strip", "Kitchen Counter Left", "Living Room Hue Go 1", "Garden Spot 1"},
			wantExcludes: []string{"Office Desk Lamp", "Bedroom Lamp 1", "Hallway Light 1"},
			description:  "Should match only lights with effects capability",
		},
		{
			name:         "lights that support color",
			query:        `@"with:color"`,
			wantCount:    20,
			wantContains: []string{"Office Desk Lamp", "Kitchen Counter Left"},
			wantExcludes: []string{"Hallway Light 1", "Bathroom Ceiling"},
			description:  "Should match only color-capable lights",
		},
		{
			name:         "office OR bedroom lights",
			query:        `@"office,bedroom"`,
			wantCount:    8,
			wantContains: []string{"Office Desk Lamp", "Bedroom Lamp 1", "Bedroom Ceiling"},
			wantExcludes: []string{"Kitchen", "Living"},
			description:  "Should match lights from either office or bedroom",
		},
		{
			name:         "office lights that are lamps",
			query:        `@"office lamp"`,
			wantCount:    1,
			wantContains: []string{"Office Desk Lamp"},
			wantExcludes: []string{"Office Ceiling", "Bedroom"},
			description:  "Should match office AND lamp (AND logic)",
		},
		{
			name:         "kitchen OR living room strips",
			query:        `@"kitchen,living strip"`,
			wantCount:    3,
			wantContains: []string{"Kitchen Counter Left", "Kitchen Counter Right", "Living Room Strip"},
			wantExcludes: []string{"Office Strip", "Bedroom Strip"},
			description:  "Should match lights with (kitchen OR living) AND strip (in name or archetype)",
		},
		{
			name:         "office lights excluding desk",
			query:        `@"office -desk"`,
			wantCount:    3,
			wantContains: []string{"Office Ceiling", "Office Strip", "Office Hue Go"},
			wantExcludes: []string{"Office Desk Lamp"},
			description:  "Should match office lights but exclude anything with 'desk'",
		},
		{
			name:         "all lights except bedroom",
			query:        `@"* -bedroom"`,
			wantCount:    20,
			wantContains: []string{"Office", "Kitchen", "Living"},
			wantExcludes: []string{"Bedroom Lamp 1", "Bedroom Lamp 2", "Bedroom Ceiling"},
			description:  "Should match all lights except bedroom",
		},
		{
			name:         "effect-capable office lights that are on",
			query:        `@"office with:effects on:"`,
			wantCount:    2,
			wantContains: []string{"Office Ceiling", "Office Hue Go"},
			wantExcludes: []string{"Office Desk Lamp", "Office Strip"},
			description:  "Should combine room, capability, and state filters",
		},
		{
			name:         "go or strip lights that are currently off",
			query:        `@"type:go,strip off:"`,
			wantCount:    4,
			wantContains: []string{"Office Strip", "Bedroom Strip", "Living Room Hue Go 2", "Patio Strip"},
			wantExcludes: []string{"Office Hue Go", "Kitchen Counter Left"},
			description:  "Should match go OR strip lights that are off",
		},
		{
			name:         "ceiling lights in kitchen or bedroom",
			query:        `@"kitchen,bedroom ceiling"`,
			wantCount:    2,
			wantContains: []string{"Kitchen Ceiling", "Bedroom Ceiling"},
			wantExcludes: []string{"Office Ceiling", "Hallway Light 1"},
			description:  "Should match ceiling lights in specific rooms",
		},
		{
			name:         "all go lights excluding living room",
			query:        `@"type:go -living"`,
			wantCount:    1,
			wantContains: []string{"Office Hue Go"},
			wantExcludes: []string{"Living Room Hue Go 1", "Living Room Hue Go 2"},
			description:  "Should match go lights but exclude living room",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.query, err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("%s: got %d lights, want %d", tt.description, len(result), tt.wantCount)
				t.Logf("Query: %q", tt.query)
				t.Logf("Matched lights:")
				for _, light := range result {
					t.Logf("  - %s", light.Metadata.Name)
				}
			}

			// Check that expected lights are in results
			resultNames := make(map[string]bool)
			for _, light := range result {
				resultNames[light.Metadata.Name] = true
			}

			for _, wantName := range tt.wantContains {
				found := false
				for name := range resultNames {
					if strings.Contains(name, wantName) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("%s: expected to find light containing %q, but didn't", tt.description, wantName)
				}
			}

			// Check that excluded lights are NOT in results
			for _, excludeName := range tt.wantExcludes {
				for name := range resultNames {
					if strings.Contains(name, excludeName) {
						t.Errorf("%s: expected NOT to find light containing %q, but found: %s", tt.description, excludeName, name)
					}
				}
			}
		})
	}
}

// TestEdgeCases tests edge cases and error conditions
func TestEdgeCases(t *testing.T) {
	lights := GetTestLights()

	tests := []struct {
		name        string
		query       string
		wantCount   int
		description string
	}{
		{
			name:        "query that matches nothing",
			query:       `@"nonexistent"`,
			wantCount:   0,
			description: "Should return empty result for non-matching query",
		},
		{
			name:        "empty query string",
			query:       `@""`,
			wantCount:   0,
			description: "Should return empty for empty query",
		},
		{
			name:        "all then exclude all",
			query:       `@"* -*"`,
			wantCount:   0,
			description: "Should return empty when excluding everything",
		},
		{
			name:        "all lights",
			query:       `@"all"`,
			wantCount:   24,
			description: "Should match all lights",
		},
		{
			name:        "wildcard all",
			query:       `@"*"`,
			wantCount:   24,
			description: "Should match all lights with wildcard",
		},
		{
			name:        "case insensitive matching",
			query:       `@"OFFICE"`,
			wantCount:   4,
			description: "Should match regardless of case",
		},
		{
			name:        "query with multiple exclusions",
			query:       `@"* -bedroom -kitchen -living"`,
			wantCount:   11,
			description: "Should handle multiple exclusions",
		},
		{
			name:        "complex multi-filter query",
			query:       `@"with:color on: type:go,strip"`,
			wantCount:   5,
			description: "Should handle complex combination of filters (color AND on AND (go OR strip))",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.query, err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("%s: got %d lights, want %d", tt.description, len(result), tt.wantCount)
				t.Logf("Query: %q", tt.query)
			}
		})
	}
}

// TestQueryPerformance tests that queries are reasonably efficient
func TestQueryPerformance(t *testing.T) {
	lights := GetTestLights()

	// Run a complex query many times to ensure it's reasonably fast
	query := `@"with:effects on: type:go,strip -bedroom"`

	for i := 0; i < 1000; i++ {
		q, err := Parse(query)
		if err != nil {
			t.Fatalf("Parse error: %v", err)
		}

		result := Execute(q, lights)
		if len(result) == 0 {
			// Just to prevent optimization from removing the loop
			t.Logf("Iteration %d: %d results", i, len(result))
		}
	}
}

// TestDryRunWithRealisticData tests dry-run with realistic dataset
func TestDryRunWithRealisticData(t *testing.T) {
	lights := GetTestLights()

	tests := []struct {
		name  string
		query string
	}{
		{"simple query", `@"office"`},
		{"complex query", `@"with:effects on: type:go,strip"`},
		{"many matches", `@"with:color"`},
		{"few matches", `@"office lamp"`},
		{"no matches", `@"nonexistent"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DryRun(tt.query, lights)
			if err != nil {
				t.Fatalf("DryRun error: %v", err)
			}

			// Verify output formatting
			output := FormatDryRun(result, false)
			if output == "" {
				t.Error("FormatDryRun returned empty string")
			}

			verboseOutput := FormatDryRun(result, true)
			if verboseOutput == "" {
				t.Error("FormatDryRun verbose returned empty string")
			}

			// Verbose should contain more info for many results
			if result.Count > 5 {
				if len(verboseOutput) <= len(output) {
					t.Error("Verbose output should be longer for many results")
				}
			}
		})
	}
}
