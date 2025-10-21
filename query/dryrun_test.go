package query

import (
	"strings"
	"testing"

	"github.com/kungfusheep/hue/client"
)

func TestDryRun(t *testing.T) {
	lights := []client.Light{
		makeLight("1", "Office Desk Lamp", "desk_lamp", true, true, false),
		makeLight("2", "Office Ceiling", "ceiling_round", true, true, true),
		makeLight("3", "Bedroom Lamp", "table_lamp", false, true, false),
	}

	result, err := DryRun(`@"office"`, lights)
	if err != nil {
		t.Fatalf("DryRun error: %v", err)
	}

	if result.Count != 2 {
		t.Errorf("DryRun count = %d, want 2", result.Count)
	}

	if len(result.Matched) != 2 {
		t.Errorf("DryRun matched %d lights, want 2", len(result.Matched))
	}
}

func TestFormatDryRun(t *testing.T) {
	lights := []client.Light{
		makeLight("1", "Office Desk Lamp", "desk_lamp", true, true, false),
		makeLight("2", "Office Ceiling", "ceiling_round", false, true, true),
	}

	result := &DryRunResult{
		Query:   `@"office"`,
		Matched: lights,
		Count:   2,
	}

	output := FormatDryRun(result, false)

	// Check that output contains expected elements
	if !strings.Contains(output, "matched 2 light(s)") {
		t.Errorf("FormatDryRun output missing count: %s", output)
	}

	if !strings.Contains(output, "Office Desk Lamp") {
		t.Errorf("FormatDryRun output missing light name: %s", output)
	}

	if !strings.Contains(output, "Office Ceiling") {
		t.Errorf("FormatDryRun output missing light name: %s", output)
	}
}

func TestFormatDryRunNoMatches(t *testing.T) {
	result := &DryRunResult{
		Query:   `@"nonexistent"`,
		Matched: []client.Light{},
		Count:   0,
	}

	output := FormatDryRun(result, false)

	if !strings.Contains(output, "matched no lights") {
		t.Errorf("FormatDryRun output should indicate no matches: %s", output)
	}
}

func TestFormatDryRunManyLights(t *testing.T) {
	// Create many lights to test summarization
	lights := make([]client.Light, 20)
	for i := 0; i < 20; i++ {
		lights[i] = makeLight(
			string(rune('A'+i)),
			"Light "+string(rune('A'+i)),
			"bulb",
			true,
			true,
			false,
		)
	}

	result := &DryRunResult{
		Query:   `@"light"`,
		Matched: lights,
		Count:   20,
	}

	// Non-verbose should summarize
	output := FormatDryRun(result, false)
	if !strings.Contains(output, "and 15 more") {
		t.Errorf("FormatDryRun should summarize many lights: %s", output)
	}
	if !strings.Contains(output, "--verbose") {
		t.Errorf("FormatDryRun should suggest verbose mode: %s", output)
	}

	// Verbose should show all
	verboseOutput := FormatDryRun(result, true)
	if strings.Contains(verboseOutput, "and 15 more") {
		t.Errorf("FormatDryRun verbose should not summarize: %s", verboseOutput)
	}
}
