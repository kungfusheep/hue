package query

import (
	"fmt"
	"strings"

	"github.com/kungfusheep/hue/client"
)

// DryRunResult contains information about a dry-run query execution
type DryRunResult struct {
	Query   string
	Matched []client.Light
	Count   int
}

// FormatDryRun formats a dry-run result for display
func FormatDryRun(result *DryRunResult, verbose bool) string {
	var output strings.Builder

	if result.Count == 0 {
		output.WriteString(fmt.Sprintf("Query %q matched no lights\n", result.Query))
		return output.String()
	}

	output.WriteString(fmt.Sprintf("Query %q matched %d light(s):\n", result.Query, result.Count))

	if verbose || result.Count <= 10 {
		// Show all lights if verbose or count is small
		for _, light := range result.Matched {
			status := "off"
			if light.On.On {
				status = fmt.Sprintf("on, %.0f%%", light.Dimming.Brightness)
			}

			effectInfo := ""
			if light.Effects != nil && light.Effects.Effect != "" && light.Effects.Effect != "no_effect" {
				effectInfo = fmt.Sprintf(" [effect: %s]", light.Effects.Effect)
			}

			output.WriteString(fmt.Sprintf("  ✓ %s (ID: %s, %s, %s)%s\n",
				light.Metadata.Name,
				light.ID,
				light.Metadata.Archetype,
				status,
				effectInfo,
			))
		}
	} else {
		// Show first few and summarize
		for i := 0; i < 5; i++ {
			light := result.Matched[i]
			output.WriteString(fmt.Sprintf("  ✓ %s (%s)\n",
				light.Metadata.Name,
				light.Metadata.Archetype,
			))
		}
		output.WriteString(fmt.Sprintf("  ... and %d more\n", result.Count-5))
		output.WriteString("\nUse --verbose or -v to see all matches\n")
	}

	return output.String()
}

// DryRun executes a query and returns the result without performing any actions
func DryRun(queryStr string, lights []client.Light) (*DryRunResult, error) {
	q, err := Parse(queryStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	matched := Execute(q, lights)

	return &DryRunResult{
		Query:   queryStr,
		Matched: matched,
		Count:   len(matched),
	}, nil
}
