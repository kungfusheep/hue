package query

import (
	"testing"

	"github.com/kungfusheep/hue/client"
)

// TestTokenize tests the tokenizer
func TestTokenize(t *testing.T) {
	tests := []struct {
		input      string
		wantTokens int
		wantErr    bool
	}{
		{"office", 1, false},
		{"office lamp", 2, false},
		{"(office)", 3, false}, // (, office, )
		{"(office lamp)", 4, false},
		{"(office) , (bedroom)", 7, false},
		{"((office))", 5, false},
		{"(office", 0, true}, // Unmatched
		{"office)", 0, true}, // Unmatched
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := tokenize(tt.input)

			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if len(tokens) != tt.wantTokens {
				t.Errorf("got %d tokens, want %d", len(tokens), tt.wantTokens)
				for i, tok := range tokens {
					t.Logf("  [%d] %v: %q", i, tok.typ, tok.value)
				}
			}
		})
	}
}

// TestBracketParsing tests bracket-based queries
func TestBracketParsing(t *testing.T) {
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
			name:      "simple brackets",
			query:     `@"(office)"`,
			wantCount: 2,
			wantIDs:   []string{"1", "2"},
		},
		{
			name:      "AND in brackets",
			query:     `@"(office lamp)"`,
			wantCount: 1,
			wantIDs:   []string{"1"},
		},
		{
			name:      "OR with brackets",
			query:     `@"(office) , (bedroom)"`,
			wantCount: 3,
			wantIDs:   []string{"1", "2", "3"},
		},
		{
			name:      "complex OR",
			query:     `@"(office lamp) , (bedroom)"`,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "AND of ORs",
			query:     `@"(office,bedroom) (lamp,ceiling)"`,
			wantCount: 3,
			wantIDs:   []string{"1", "2", "3"},
		},
		{
			name:      "nested brackets",
			query:     `@"((office lamp) , bedroom)"`,
			wantCount: 2,
			wantIDs:   []string{"1", "3"},
		},
		{
			name:      "brackets with type filter",
			query:     `@"(office type:go) , (bedroom type:strip)"`,
			wantCount: 0, // No matches
		},
		{
			name:      "brackets with state",
			query:     `@"(office on:) , (bedroom)"`,
			wantCount: 3,
			wantIDs:   []string{"1", "2", "3"},
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
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
				t.Logf("Query: %q", tt.query)
				for _, light := range result {
					t.Logf("  - %s", light.Metadata.Name)
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

// TestBracketNegation tests negation with brackets
func TestBracketNegation(t *testing.T) {
	lights := GetTestLights()

	tests := []struct {
		name      string
		query     string
		wantCount int
	}{
		{
			name:      "NOT a bracketed group",
			query:     `@"-(office lamp)"`,
			wantCount: 23, // All except office lamp
		},
		{
			name:      "AND with NOT bracket",
			query:     `@"with:effects -(bedroom)"`,
			wantCount: 12, // Effect lights excluding bedroom (adjusted)
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
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}
		})
	}
}

// TestBackwardCompatibility ensures non-bracket queries still work
func TestBackwardCompatibility(t *testing.T) {
	lights := GetTestLights()

	tests := []struct {
		query     string
		wantCount int
	}{
		{`@"office"`, 4},
		{`@"office lamp"`, 1},
		{`@"office,bedroom"`, 8},
		{`@"with:effects on:"`, 8},
		{`@"office -desk"`, 3},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			q, err := Parse(tt.query)
			if err != nil {
				t.Fatalf("Parse error: %v", err)
			}

			// Should NOT have AST (no brackets)
			if q.AST != nil {
				t.Errorf("query without brackets should not use AST")
			}

			result := Execute(q, lights)

			if len(result) != tt.wantCount {
				t.Errorf("got %d lights, want %d", len(result), tt.wantCount)
			}
		})
	}
}
