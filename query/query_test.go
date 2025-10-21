package query

import (
	"strings"
	"testing"

	"github.com/kungfusheep/hue/client"
)

// Test helper to create a light
func makeLight(id, name, archetype string, on bool, hasColor, hasEffects bool) client.Light {
	light := client.Light{
		ID: id,
		Metadata: client.Metadata{
			Name:      name,
			Archetype: archetype,
		},
		On: client.OnState{On: on},
	}

	if hasColor {
		light.Color = &client.Color{
			XY: client.XY{X: 0.5, Y: 0.5},
		}
	}

	if hasEffects {
		light.Effects = &client.Effects{
			EffectValues: []string{"fire", "candle"},
		}
	}

	return light
}

// TestIsQuery tests query detection
func TestIsQuery(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{`@"office"`, true},
		{`@office`, true},
		{`office`, false},
		{``, false},
		{`room:office`, false},
	}

	for _, tt := range tests {
		got := IsQuery(tt.input)
		if got != tt.want {
			t.Errorf("IsQuery(%q) = %v, want %v", tt.input, got, tt.want)
		}
	}
}

// TestParseBasic tests basic query parsing
func TestParseBasic(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantTerms int
	}{
		{"empty", `@""`, 0},
		{"single term", `@"office"`, 1},
		{"two terms", `@"office lamp"`, 2},
		{"three terms", `@"office lamp desk"`, 3},
		{"with quotes removed", `@"office"`, 1},
		{"without quotes", `@office`, 1},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if len(q.Terms) != tt.wantTerms {
				t.Errorf("Parse(%q) got %d terms, want %d", tt.input, len(q.Terms), tt.wantTerms)
			}
		})
	}
}

// TestParseOR tests OR logic with commas
func TestParseOR(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		wantTerms   int
		wantFilters int // filters in first term
	}{
		{"single OR", `@"room:office,bedroom"`, 1, 2},
		{"multiple OR", `@"lamp,ceiling,floor"`, 1, 3},
		{"OR with spaces", `@"office,bedroom,kitchen"`, 1, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("Parse(%q) error = %v", tt.input, err)
			}
			if len(q.Terms) != tt.wantTerms {
				t.Errorf("Parse(%q) got %d terms, want %d", tt.input, len(q.Terms), tt.wantTerms)
			}
			if len(q.Terms[0].Filters) != tt.wantFilters {
				t.Errorf("Parse(%q) got %d filters in first term, want %d",
					tt.input, len(q.Terms[0].Filters), tt.wantFilters)
			}
		})
	}
}

// TestParseNegation tests exclusion with dash
func TestParseNegation(t *testing.T) {
	q, err := Parse(`@"office -desk"`)
	if err != nil {
		t.Fatalf("Parse error = %v", err)
	}

	if len(q.Terms) != 2 {
		t.Fatalf("got %d terms, want 2", len(q.Terms))
	}

	if q.Terms[0].Negate {
		t.Errorf("first term should not be negated")
	}

	if !q.Terms[1].Negate {
		t.Errorf("second term should be negated")
	}
}

// TestParseFilterTypes tests different filter type parsing
func TestParseFilterTypes(t *testing.T) {
	tests := []struct {
		input      string
		wantType   FilterType
		wantValue  string
	}{
		{"office", FilterFuzzy, "office"},
		{"name:office", FilterName, "office"},
		{"room:office", FilterRoom, "office"},
		{"type:go", FilterArchetype, "go"},
		{"on:", FilterStateOn, ""},
		{"off:", FilterStateOff, ""},
		{"with:effects", FilterCapabilityEffects, ""},
		{"with:color", FilterCapabilityColor, ""},
		{"all", FilterAll, ""},
		{"*", FilterAll, ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			filter := parseFilter(tt.input)
			if filter.Type != tt.wantType {
				t.Errorf("parseFilter(%q).Type = %v, want %v", tt.input, filter.Type, tt.wantType)
			}
			if filter.Value != tt.wantValue {
				t.Errorf("parseFilter(%q).Value = %q, want %q", tt.input, filter.Value, tt.wantValue)
			}
		})
	}
}

// TestFuzzyMatch tests wildcard and substring matching
func TestFuzzyMatch(t *testing.T) {
	tests := []struct {
		s       string
		pattern string
		want    bool
	}{
		// Substring
		{"Office Lamp", "office", true},
		{"Office Lamp", "lamp", true},
		{"Office Lamp", "desk", false},

		// Wildcard prefix
		{"Office Lamp", "*lamp", true},
		{"Office Lamp", "*office", false},

		// Wildcard suffix
		{"Office Lamp", "office*", true},
		{"Office Lamp", "lamp*", false},

		// Wildcard both
		{"Office Lamp", "*lamp*", true},
		{"Office Lamp", "*fice*", true},
		{"Office Lamp", "*desk*", false},

		// Case insensitive
		{"Office Lamp", "OFFICE", true},
		{"Office Lamp", "LAMP", true},
	}

	for _, tt := range tests {
		t.Run(tt.s+"~"+tt.pattern, func(t *testing.T) {
			// fuzzyMatch expects lowercase inputs
			got := fuzzyMatch(strings.ToLower(tt.s), strings.ToLower(tt.pattern))
			if got != tt.want {
				t.Errorf("fuzzyMatch(%q, %q) = %v, want %v", tt.s, tt.pattern, got, tt.want)
			}
		})
	}
}

// TestExecute tests query execution with real data
func TestExecute(t *testing.T) {
	// Create test lights
	lights := []client.Light{
		makeLight("1", "Office Desk Lamp", "desk_lamp", true, true, false),
		makeLight("2", "Office Ceiling", "ceiling_round", true, true, true),
		makeLight("3", "Bedroom Lamp", "table_lamp", false, true, false),
		makeLight("4", "Kitchen Strip", "hue_lightstrip", true, true, true),
		makeLight("5", "Living Room Go", "hue_go", true, true, true),
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
		wantIDs   []string
	}{
		{
			name:      "fuzzy name match",
			query:     `@"office"`,
			wantCount: 2,
			wantIDs:   []string{"1", "2"},
		},
		{
			name:      "wildcard match",
			query:     `@"*lamp"`,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "type filter",
			query:     `@"type:go"`,
			wantCount: 1,
			wantIDs:   []string{"5"},
		},
		{
			name:      "state filter - on",
			query:     `@"on:"`,
			wantCount: 4,
			wantIDs:   []string{"1", "2", "4", "5"},
		},
		{
			name:      "state filter - off",
			query:     `@"off:"`,
			wantCount: 1,
			wantIDs:   []string{"3"},
		},
		{
			name:      "capability filter - effects",
			query:     `@"with:effects"`,
			wantCount: 3,
			wantIDs:   []string{"2", "4", "5"},
		},
		{
			name:      "AND logic",
			query:     `@"office lamp"`,
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name:      "OR logic",
			query:     `@"office,bedroom"`,
			wantCount: 3,
			wantIDs:   []string{"1", "2", "3"},
		},
		{
			name:      "AND + OR logic",
			query:     `@"office,bedroom lamp"`,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "exclusion",
			query:     `@"office -desk"`,
			wantCount: 1,
			wantIDs:   []string{"2"},
		},
		{
			name:      "complex query",
			query:     `@"with:effects on:"`,
			wantCount: 3,
			wantIDs:   []string{"2", "4", "5"},
		},
		{
			name:      "all",
			query:     `@"all"`,
			wantCount: 5,
			wantIDs:   []string{"1", "2", "3", "4", "5"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error = %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("Execute(%q) returned %d lights, want %d", tt.query, len(result), tt.wantCount)
			}

			// Check IDs
			gotIDs := make(map[string]bool)
			for _, light := range result {
				gotIDs[light.ID] = true
			}

			for _, wantID := range tt.wantIDs {
				if !gotIDs[wantID] {
					t.Errorf("Execute(%q) missing light ID %s", tt.query, wantID)
				}
			}
		})
	}
}

// TestExecuteEmpty tests edge cases
func TestExecuteEmpty(t *testing.T) {
	lights := []client.Light{
		makeLight("1", "Office", "desk_lamp", true, true, false),
	}

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{"no match", `@"nonexistent"`, 0},
		{"empty query", `@""`, 0},
		{"exclude all", `@"all -office"`, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error = %v", err)
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("Execute(%q) returned %d lights, want %d", tt.query, len(result), tt.wantCount)
			}
		})
	}
}
