package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/query"
)

// resolveLightIDs takes a name, ID, or query and returns one or more light IDs
func resolveLightIDs(ctx context.Context, nameOrIDOrQuery string) ([]string, error) {
	// Check if this is a query
	if query.IsQuery(nameOrIDOrQuery) {
		return resolveLightQuery(ctx, nameOrIDOrQuery)
	}

	// Otherwise, resolve as a single light
	lightID, err := resolveSingleLight(ctx, nameOrIDOrQuery)
	if err != nil {
		return nil, err
	}
	return []string{lightID}, nil
}

// resolveLightQuery executes a query and returns matching light IDs
func resolveLightQuery(ctx context.Context, queryStr string) ([]string, error) {
	// Get all lights
	lights, err := hueClient.GetLights(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get lights: %w", err)
	}

	// Build query context with room/zone data (only if needed)
	var queryCtx *query.Context

	if query.NeedsRoomContext(queryStr) {
		queryCtx = &query.Context{}

		// Fetch rooms and zones for room: filter
		rooms, err := hueClient.GetRooms(ctx)
		if err == nil {
			queryCtx.Rooms = rooms
		}

		zones, err := hueClient.GetZones(ctx)
		if err == nil {
			queryCtx.Zones = zones
		}
	}

	// Handle dry-run mode
	if dryRun {
		// Parse and execute for dry-run
		q, err := query.Parse(queryStr)
		if err != nil {
			return nil, err
		}

		matched := query.ExecuteWithContext(q, lights, queryCtx)

		// Format and print dry-run result
		result := &query.DryRunResult{
			Query:   queryStr,
			Matched: matched,
			Count:   len(matched),
		}
		fmt.Print(query.FormatDryRun(result, verbose))

		// Return empty to indicate dry-run
		return nil, fmt.Errorf("dry-run mode: no changes made")
	}

	// Execute the query
	q, err := query.Parse(queryStr)
	if err != nil {
		return nil, fmt.Errorf("failed to parse query: %w", err)
	}

	matched := query.ExecuteWithContext(q, lights, queryCtx)

	if len(matched) == 0 {
		return nil, fmt.Errorf("query %q matched no lights", queryStr)
	}

	// Extract IDs
	var ids []string
	for _, light := range matched {
		ids = append(ids, light.ID)
	}

	// Show what matched if verbose
	if verbose && !quiet {
		fmt.Printf("Query %q matched %d light(s):\n", queryStr, len(matched))
		for _, light := range matched {
			fmt.Printf("  ✓ %s\n", light.Metadata.Name)
		}
	}

	return ids, nil
}

// resolveSingleLight takes a name or ID and returns the actual light ID (legacy function)
func resolveSingleLight(ctx context.Context, nameOrID string) (string, error) {
	// If it looks like a UUID, return it as-is
	if strings.Contains(nameOrID, "-") && len(nameOrID) > 30 {
		return nameOrID, nil
	}
	
	// Otherwise, search for the light by name
	lights, err := hueClient.GetLights(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get lights: %w", err)
	}
	
	// Try exact match first (case-insensitive)
	for _, light := range lights {
		if strings.EqualFold(light.Metadata.Name, nameOrID) {
			return light.ID, nil
		}
	}
	
	// Try partial match
	var matches []struct {
		ID   string
		Name string
	}
	
	searchLower := strings.ToLower(nameOrID)
	for _, light := range lights {
		if strings.Contains(strings.ToLower(light.Metadata.Name), searchLower) {
			matches = append(matches, struct {
				ID   string
				Name string
			}{
				ID:   light.ID,
				Name: light.Metadata.Name,
			})
		}
	}
	
	if len(matches) == 0 {
		return "", fmt.Errorf("no light found matching '%s'", nameOrID)
	}
	
	if len(matches) == 1 {
		return matches[0].ID, nil
	}
	
	// Multiple matches - show them to the user
	return "", fmt.Errorf("multiple lights match '%s':\n%s\nPlease be more specific", 
		nameOrID, formatMatches(matches))
}

// resolveGroupID takes a name or ID and returns the actual group ID
func resolveGroupID(ctx context.Context, nameOrID string) (string, error) {
	// If it looks like a UUID, return it as-is
	if strings.Contains(nameOrID, "-") && len(nameOrID) > 30 {
		return nameOrID, nil
	}
	
	// Search in rooms first (they have names)
	rooms, err := hueClient.GetRooms(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get rooms: %w", err)
	}
	
	// Try exact match first
	for _, room := range rooms {
		if strings.EqualFold(room.Metadata.Name, nameOrID) {
			// Find the grouped_light for this room
			for _, service := range room.Services {
				if service.RType == "grouped_light" {
					return service.RID, nil
				}
			}
		}
	}
	
	// Try partial match
	var matches []struct {
		ID       string
		Name     string
		GroupID  string
	}
	
	searchLower := strings.ToLower(nameOrID)
	for _, room := range rooms {
		if strings.Contains(strings.ToLower(room.Metadata.Name), searchLower) {
			// Find the grouped_light for this room
			groupID := ""
			for _, service := range room.Services {
				if service.RType == "grouped_light" {
					groupID = service.RID
					break
				}
			}
			if groupID != "" {
				matches = append(matches, struct {
					ID      string
					Name    string
					GroupID string
				}{
					ID:      room.ID,
					Name:    room.Metadata.Name,
					GroupID: groupID,
				})
			}
		}
	}
	
	if len(matches) == 0 {
		return "", fmt.Errorf("no room/group found matching '%s'", nameOrID)
	}
	
	if len(matches) == 1 {
		return matches[0].GroupID, nil
	}
	
	// Multiple matches
	var matchInfo []struct {
		ID   string
		Name string
	}
	for _, m := range matches {
		matchInfo = append(matchInfo, struct {
			ID   string
			Name string
		}{
			ID:   m.GroupID,
			Name: m.Name,
		})
	}
	
	return "", fmt.Errorf("multiple rooms match '%s':\n%s\nPlease be more specific", 
		nameOrID, formatMatches(matchInfo))
}

// resolveSceneID takes a name or ID and returns the actual scene ID
func resolveSceneID(ctx context.Context, nameOrID string) (string, error) {
	// If it looks like a UUID, return it as-is
	if strings.Contains(nameOrID, "-") && len(nameOrID) > 30 {
		return nameOrID, nil
	}
	
	// Check if input contains room specifier like "Nightlight:Master Bedroom"
	parts := strings.Split(nameOrID, ":")
	sceneName := strings.TrimSpace(parts[0])
	roomFilter := ""
	if len(parts) == 2 {
		roomFilter = strings.TrimSpace(parts[1])
	}
	
	// Get scenes
	scenes, err := hueClient.GetScenes(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get scenes: %w", err)
	}
	
	// Get rooms and zones for room name lookup
	rooms, err := hueClient.GetRooms(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get rooms: %w", err)
	}
	
	roomIDToName := make(map[string]string)
	for _, room := range rooms {
		roomIDToName[room.ID] = room.Metadata.Name
	}
	
	zones, err := hueClient.GetZones(ctx)
	if err == nil {
		for _, zone := range zones {
			roomIDToName[zone.ID] = zone.Metadata.Name
		}
	}
	
	// Helper to get room name for a scene
	getRoomName := func(scene client.Scene) string {
		if scene.Group.RType == "room" || scene.Group.RType == "zone" {
			return roomIDToName[scene.Group.RID]
		}
		return ""
	}
	
	// If room filter specified, try to find matching scene
	if roomFilter != "" {
		var roomFilterMatches []struct {
			ID       string
			Name     string
			RoomName string
		}
		
		roomFilterLower := strings.ToLower(roomFilter)
		for _, scene := range scenes {
			roomName := getRoomName(scene)
			if strings.EqualFold(scene.Metadata.Name, sceneName) && 
			   strings.Contains(strings.ToLower(roomName), roomFilterLower) {
				roomFilterMatches = append(roomFilterMatches, struct {
					ID       string
					Name     string
					RoomName string
				}{
					ID:       scene.ID,
					Name:     scene.Metadata.Name,
					RoomName: roomName,
				})
			}
		}
		
		if len(roomFilterMatches) == 1 {
			return roomFilterMatches[0].ID, nil
		}
		
		if len(roomFilterMatches) > 1 {
			return "", fmt.Errorf("multiple scenes match '%s' in rooms containing '%s':\n%s\nPlease be more specific", 
				sceneName, roomFilter, formatSceneMatches(roomFilterMatches))
		}
		// If no matches with room filter, continue to show all matches
	}
	
	// Try exact match first (no room filter)
	var exactMatches []struct {
		ID       string
		Name     string
		RoomName string
	}
	
	for _, scene := range scenes {
		if strings.EqualFold(scene.Metadata.Name, sceneName) {
			exactMatches = append(exactMatches, struct {
				ID       string
				Name     string
				RoomName string
			}{
				ID:       scene.ID,
				Name:     scene.Metadata.Name,
				RoomName: getRoomName(scene),
			})
		}
	}
	
	if len(exactMatches) == 1 {
		return exactMatches[0].ID, nil
	}
	
	if len(exactMatches) > 1 {
		// Multiple exact matches - show with room names
		return "", fmt.Errorf("multiple scenes named '%s':\n%s\nSpecify the room like: '%s:Room Name'", 
			sceneName, formatSceneMatches(exactMatches), sceneName)
	}
	
	// Try partial match
	var partialMatches []struct {
		ID       string
		Name     string
		RoomName string
	}
	
	searchLower := strings.ToLower(sceneName)
	for _, scene := range scenes {
		if strings.Contains(strings.ToLower(scene.Metadata.Name), searchLower) {
			partialMatches = append(partialMatches, struct {
				ID       string
				Name     string
				RoomName string
			}{
				ID:       scene.ID,
				Name:     scene.Metadata.Name,
				RoomName: getRoomName(scene),
			})
		}
	}
	
	if len(partialMatches) == 0 {
		return "", fmt.Errorf("no scene found matching '%s'", nameOrID)
	}
	
	if len(partialMatches) == 1 {
		return partialMatches[0].ID, nil
	}
	
	// Multiple matches
	return "", fmt.Errorf("multiple scenes match '%s':\n%s\nPlease be more specific", 
		nameOrID, formatSceneMatches(partialMatches))
}

// formatMatches formats multiple matches for display
func formatMatches(matches []struct {
	ID   string
	Name string
}) string {
	var lines []string
	for _, match := range matches {
		lines = append(lines, fmt.Sprintf("  - %s (ID: %s)", match.Name, match.ID))
	}
	return strings.Join(lines, "\n")
}

// formatSceneMatches formats multiple scene matches with room info
func formatSceneMatches(matches []struct {
	ID       string
	Name     string
	RoomName string
}) string {
	var lines []string
	for _, match := range matches {
		if match.RoomName != "" {
			lines = append(lines, fmt.Sprintf("  - %s (%s) [ID: %s]", match.Name, match.RoomName, match.ID))
		} else {
			lines = append(lines, fmt.Sprintf("  - %s [ID: %s]", match.Name, match.ID))
		}
	}
	return strings.Join(lines, "\n")
}

// resolveLightID is kept for backward compatibility - resolves to single light only
func resolveLightID(ctx context.Context, nameOrID string) (string, error) {
	if query.IsQuery(nameOrID) {
		return "", fmt.Errorf("query syntax not supported for this command (use a single light name or ID)")
	}
	return resolveSingleLight(ctx, nameOrID)
}


// resolveGroupIDs takes a name, ID, or simple pattern and returns one or more group IDs
// Note: Groups use simple pattern matching, not full query syntax like lights
func resolveGroupIDs(ctx context.Context, nameOrIDOrPattern string) ([]string, error) {
	// For now, groups just use single resolution
	// Could be extended with pattern matching in the future
	groupID, err := resolveGroupID(ctx, nameOrIDOrPattern)
	if err != nil {
		return nil, err
	}
	return []string{groupID}, nil
}
