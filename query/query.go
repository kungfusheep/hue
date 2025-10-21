package query

import (
	"fmt"
	"strings"

	"github.com/kungfusheep/hue/client"
)

// Query represents a parsed light query
type Query struct {
	Raw   string
	Terms []Term
}

// Context provides additional data for query execution
type Context struct {
	Rooms  []client.Room
	Zones  []client.Zone
	Devices map[string][]string // deviceID -> lightIDs mapping
}

// Term represents a single query term (can be AND or OR)
type Term struct {
	Filters []Filter // OR-ed together within a term
	Negate  bool     // If true, this is an exclusion term
}

// Filter represents a single filter criterion
type Filter struct {
	Type  FilterType
	Value string
}

// FilterType indicates what kind of filter this is
type FilterType int

const (
	FilterFuzzy         FilterType = iota // Default: fuzzy match on name AND archetype
	FilterName                            // name:value - explicit name only
	FilterRoom                            // room:value
	FilterArchetype                       // type:value (archetype)
	FilterStateOn                         // on:
	FilterStateOff                        // off:
	FilterCapabilityEffects               // with:effects
	FilterCapabilityColor                 // with:color
	FilterEffect                          // effect:value - current effect
	FilterBrightnessGreater               // brightness>N
	FilterBrightnessLess                  // brightness<N
	FilterMode                            // mode:value
	FilterGamut                           // gamut:value
	FilterID                              // id:value - match by ID pattern
	FilterAll                             // all or *
)

// IsQuery checks if a string is a query (starts with @)
func IsQuery(s string) bool {
	return strings.HasPrefix(s, "@")
}

// NeedsRoomContext checks if a query requires room/zone data
func NeedsRoomContext(queryStr string) bool {
	// Quick check without full parsing
	return strings.Contains(queryStr, "room:")
}

// Parse parses a query string into a Query object
func Parse(queryStr string) (*Query, error) {
	// Remove @ prefix and quotes
	queryStr = strings.TrimPrefix(queryStr, "@")
	queryStr = strings.Trim(queryStr, `"`)

	q := &Query{
		Raw:   queryStr,
		Terms: []Term{},
	}

	// Split by space for AND terms
	andTerms := strings.Fields(queryStr)

	for _, andTerm := range andTerms {
		term := parseTerm(andTerm)
		q.Terms = append(q.Terms, term)
	}

	return q, nil
}

// parseTerm parses a single AND term (which may contain OR filters)
func parseTerm(termStr string) Term {
	term := Term{
		Filters: []Filter{},
		Negate:  false,
	}

	// Check for negation
	if strings.HasPrefix(termStr, "-") {
		term.Negate = true
		termStr = strings.TrimPrefix(termStr, "-")
	}

	// Split by comma for OR filters
	orFilters := strings.Split(termStr, ",")

	for _, filterStr := range orFilters {
		filterStr = strings.TrimSpace(filterStr)
		if filterStr == "" {
			continue
		}

		filter := parseFilter(filterStr)
		term.Filters = append(term.Filters, filter)
	}

	return term
}

// parseFilter parses a single filter string
func parseFilter(filterStr string) Filter {
	// Check for brightness comparisons (brightness>N or brightness<N)
	if strings.HasPrefix(filterStr, "brightness>") {
		value := strings.TrimPrefix(filterStr, "brightness>")
		return Filter{Type: FilterBrightnessGreater, Value: value}
	}
	if strings.HasPrefix(filterStr, "brightness<") {
		value := strings.TrimPrefix(filterStr, "brightness<")
		return Filter{Type: FilterBrightnessLess, Value: value}
	}

	// Check for typed filters (prefix:value)
	if strings.Contains(filterStr, ":") {
		parts := strings.SplitN(filterStr, ":", 2)
		prefix := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		switch prefix {
		case "name":
			return Filter{Type: FilterName, Value: value}
		case "room":
			return Filter{Type: FilterRoom, Value: value}
		case "type":
			return Filter{Type: FilterArchetype, Value: value}
		case "effect":
			return Filter{Type: FilterEffect, Value: value}
		case "mode":
			return Filter{Type: FilterMode, Value: value}
		case "gamut":
			return Filter{Type: FilterGamut, Value: value}
		case "id":
			return Filter{Type: FilterID, Value: value}
		case "on":
			return Filter{Type: FilterStateOn, Value: ""}
		case "off":
			return Filter{Type: FilterStateOff, Value: ""}
		case "with":
			switch value {
			case "effects":
				return Filter{Type: FilterCapabilityEffects, Value: ""}
			case "color":
				return Filter{Type: FilterCapabilityColor, Value: ""}
			}
		}
	}

	// Check for special keywords
	if filterStr == "all" || filterStr == "*" {
		return Filter{Type: FilterAll, Value: ""}
	}

	// Default to fuzzy search (name AND archetype)
	return Filter{Type: FilterFuzzy, Value: filterStr}
}

// Execute executes a query against a list of lights
func Execute(q *Query, lights []client.Light) []client.Light {
	return ExecuteWithContext(q, lights, nil)
}

// ExecuteWithContext executes a query with additional context (rooms, devices, etc.)
func ExecuteWithContext(q *Query, lights []client.Light, ctx *Context) []client.Light {
	// Handle empty query
	if len(q.Terms) == 0 {
		return []client.Light{}
	}

	result := lights

	// Apply each AND term sequentially
	for _, term := range q.Terms {
		result = applyTermWithContext(term, result, ctx)
	}

	return result
}

// applyTerm applies a single term (with OR filters) to the light list
func applyTerm(term Term, lights []client.Light) []client.Light {
	return applyTermWithContext(term, lights, nil)
}

// applyTermWithContext applies a term with additional context
func applyTermWithContext(term Term, lights []client.Light, ctx *Context) []client.Light {
	if term.Negate {
		// Exclusion: remove lights that match ANY of the OR filters
		matched := matchFiltersWithContext(term.Filters, lights, ctx)
		return exclude(lights, matched)
	}

	// Normal term: keep lights that match ANY of the OR filters
	return matchFiltersWithContext(term.Filters, lights, ctx)
}

// matchFilters returns lights that match ANY of the filters (OR logic)
func matchFilters(filters []Filter, lights []client.Light) []client.Light {
	return matchFiltersWithContext(filters, lights, nil)
}

// matchFiltersWithContext returns lights that match ANY of the filters with context
func matchFiltersWithContext(filters []Filter, lights []client.Light, ctx *Context) []client.Light {
	if len(filters) == 0 {
		return lights
	}

	var result []client.Light
	seen := make(map[string]bool)

	for _, filter := range filters {
		matched := applyFilterWithContext(filter, lights, ctx)
		for _, light := range matched {
			if !seen[light.ID] {
				result = append(result, light)
				seen[light.ID] = true
			}
		}
	}

	return result
}

// applyFilter applies a single filter to the light list
func applyFilter(filter Filter, lights []client.Light) []client.Light {
	return applyFilterWithContext(filter, lights, nil)
}

// applyFilterWithContext applies a filter with additional context
func applyFilterWithContext(filter Filter, lights []client.Light, ctx *Context) []client.Light {
	var result []client.Light

	switch filter.Type {
	case FilterAll:
		return lights

	case FilterFuzzy:
		// Fuzzy search - checks BOTH name AND archetype (unless explicit match found)
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			name := strings.ToLower(light.Metadata.Name)
			archetype := strings.ToLower(light.Metadata.Archetype)
			if fuzzyMatch(name, pattern) || fuzzyMatch(archetype, pattern) {
				result = append(result, light)
			}
		}

	case FilterName:
		// Explicit name-only match
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			name := strings.ToLower(light.Metadata.Name)
			if fuzzyMatch(name, pattern) {
				result = append(result, light)
			}
		}

	case FilterRoom:
		// Match by actual room membership
		if ctx != nil && len(ctx.Rooms) > 0 {
			// Build device-to-light mapping if needed
			if ctx.Devices == nil {
				ctx.Devices = buildDeviceLightMap(lights)
			}

			// Find rooms matching the pattern
			pattern := strings.ToLower(filter.Value)
			var roomLightIDs []string

			for _, room := range ctx.Rooms {
				roomName := strings.ToLower(room.Metadata.Name)
				if fuzzyMatch(roomName, pattern) {
					// Get all lights in this room via children (devices)
					for _, child := range room.Children {
						if child.RType == "device" {
							if lightIDs, ok := ctx.Devices[child.RID]; ok {
								roomLightIDs = append(roomLightIDs, lightIDs...)
							}
						}
					}
				}
			}

			// Also check zones
			for _, zone := range ctx.Zones {
				zoneName := strings.ToLower(zone.Metadata.Name)
				if fuzzyMatch(zoneName, pattern) {
					for _, child := range zone.Children {
						if child.RType == "device" {
							if lightIDs, ok := ctx.Devices[child.RID]; ok {
								roomLightIDs = append(roomLightIDs, lightIDs...)
							}
						}
					}
				}
			}

			// Filter lights by IDs
			roomLightSet := make(map[string]bool)
			for _, id := range roomLightIDs {
				roomLightSet[id] = true
			}

			for _, light := range lights {
				if roomLightSet[light.ID] {
					result = append(result, light)
				}
			}
		} else {
			// Fallback to name matching if no context
			pattern := strings.ToLower(filter.Value)
			for _, light := range lights {
				name := strings.ToLower(light.Metadata.Name)
				if fuzzyMatch(name, pattern) {
					result = append(result, light)
				}
			}
		}

	case FilterArchetype:
		// Match by archetype/type
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			archetype := strings.ToLower(light.Metadata.Archetype)
			if fuzzyMatch(archetype, pattern) {
				result = append(result, light)
			}
		}

	case FilterStateOn:
		// Only lights that are currently on
		for _, light := range lights {
			if light.On.On {
				result = append(result, light)
			}
		}

	case FilterStateOff:
		// Only lights that are currently off
		for _, light := range lights {
			if !light.On.On {
				result = append(result, light)
			}
		}

	case FilterCapabilityEffects:
		// Only lights that support effects
		for _, light := range lights {
			if light.Effects != nil && len(light.Effects.EffectValues) > 0 {
				result = append(result, light)
			}
		}

	case FilterCapabilityColor:
		// Only lights that support color
		for _, light := range lights {
			if light.Color != nil {
				result = append(result, light)
			}
		}

	case FilterEffect:
		// Match by current effect
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			if light.Effects == nil {
				continue
			}
			currentEffect := strings.ToLower(light.Effects.Effect)
			// Handle "none" or "no_effect" or empty
			if currentEffect == "" {
				currentEffect = "none"
			}
			// Also normalize no_effect to none for matching
			if currentEffect == "no_effect" {
				currentEffect = "none"
			}
			// Support wildcard matching for "any effect"
			if pattern == "*" {
				if currentEffect != "none" {
					result = append(result, light)
				}
			} else if fuzzyMatch(currentEffect, pattern) {
				result = append(result, light)
			}
		}

	case FilterBrightnessGreater:
		// Brightness greater than threshold
		threshold := parseFloat(filter.Value)
		for _, light := range lights {
			if light.Dimming.Brightness > threshold {
				result = append(result, light)
			}
		}

	case FilterBrightnessLess:
		// Brightness less than threshold
		threshold := parseFloat(filter.Value)
		for _, light := range lights {
			if light.Dimming.Brightness < threshold {
				result = append(result, light)
			}
		}

	case FilterMode:
		// Match by mode
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			mode := strings.ToLower(light.Mode)
			if fuzzyMatch(mode, pattern) {
				result = append(result, light)
			}
		}

	case FilterGamut:
		// Match by gamut type
		pattern := strings.ToUpper(filter.Value) // Gamut is uppercase (A, B, C)
		for _, light := range lights {
			if light.Color != nil {
				gamut := strings.ToUpper(light.Color.GamutType)
				if gamut == pattern {
					result = append(result, light)
				}
			}
		}

	case FilterID:
		// Match by ID pattern
		pattern := strings.ToLower(filter.Value)
		for _, light := range lights {
			id := strings.ToLower(light.ID)
			if fuzzyMatch(id, pattern) {
				result = append(result, light)
			}
		}
	}

	return result
}

// parseFloat parses a float from string, returns 0 on error
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// fuzzyMatch checks if a pattern matches a string (substring or wildcard)
// Both s and pattern should already be lowercased by caller
func fuzzyMatch(s, pattern string) bool {
	// Handle wildcards
	if strings.Contains(pattern, "*") {
		// Simple wildcard matching
		if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
			// *pattern* - contains
			inner := strings.Trim(pattern, "*")
			return strings.Contains(s, inner)
		} else if strings.HasPrefix(pattern, "*") {
			// *pattern - ends with
			suffix := strings.TrimPrefix(pattern, "*")
			return strings.HasSuffix(s, suffix)
		} else if strings.HasSuffix(pattern, "*") {
			// pattern* - starts with
			prefix := strings.TrimSuffix(pattern, "*")
			return strings.HasPrefix(s, prefix)
		}
	}

	// Default: substring match
	return strings.Contains(s, pattern)
}

// exclude removes all lights in 'toExclude' from 'lights'
func exclude(lights []client.Light, toExclude []client.Light) []client.Light {
	excludeMap := make(map[string]bool)
	for _, light := range toExclude {
		excludeMap[light.ID] = true
	}

	var result []client.Light
	for _, light := range lights {
		if !excludeMap[light.ID] {
			result = append(result, light)
		}
	}

	return result
}

// buildDeviceLightMap creates a mapping from device ID to light IDs
func buildDeviceLightMap(lights []client.Light) map[string][]string {
	deviceMap := make(map[string][]string)
	for _, light := range lights {
		deviceID := light.Owner.RID
		deviceMap[deviceID] = append(deviceMap[deviceID], light.ID)
	}
	return deviceMap
}
