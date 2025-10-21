package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/scheduler"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// Global scheduler instance
var globalScheduler *scheduler.Scheduler

// InitScheduler initializes the global scheduler
func InitScheduler(client *client.Client) {
	globalScheduler = scheduler.NewScheduler(client)
}

// GetScheduler returns the global scheduler instance
func GetScheduler() *scheduler.Scheduler {
	return globalScheduler
}

// isGroupID checks if the given ID refers to a group rather than a light
func isGroupID(ctx context.Context, hueClient *client.Client, targetID string) bool {
	// Try to get it as a group first
	_, err := hueClient.GetGroup(ctx, targetID)
	return err == nil
}

// HandleFlashEffect creates a flash effect
func HandleFlashEffect(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		targetID, ok := args["target_id"].(string)
		if !ok {
			return mcp.NewToolResultError("target_id is required"), nil
		}
		
		color, ok := args["color"].(string)
		if !ok {
			color = "#FFFFFF" // Default to white
		}
		
		flashCount := 3
		if fc, ok := args["flash_count"].(float64); ok {
			flashCount = int(fc)
		}
		
		flashDuration := 200 * time.Millisecond
		if fd, ok := args["flash_duration_ms"].(float64); ok {
			flashDuration = time.Duration(fd) * time.Millisecond
		}
		
		// Create the flash effect
		seq := scheduler.CreateFlashEffect(targetID, color, flashCount, flashDuration)

		// Check if target is a group and convert if needed
		if isGroupID(ctx, hueClient, targetID) {
			seq = scheduler.CreateGroupEffect(seq, targetID)
		}

		seqID, err := globalScheduler.ExecuteSequence(seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start flash effect: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Flash effect started on %s\nSequence ID: %s\nColor: %s\nFlashes: %d", 
			targetID, seqID, color, flashCount)), nil
	}
}

// HandlePulseEffect creates a pulse effect
func HandlePulseEffect(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		targetID, ok := args["target_id"].(string)
		if !ok {
			return mcp.NewToolResultError("target_id is required"), nil
		}
		
		minBrightness := 10.0
		if mb, ok := args["min_brightness"].(float64); ok {
			minBrightness = mb
		}
		
		maxBrightness := 100.0
		if mb, ok := args["max_brightness"].(float64); ok {
			maxBrightness = mb
		}
		
		pulseDuration := 2 * time.Second
		if pd, ok := args["pulse_duration_ms"].(float64); ok {
			pulseDuration = time.Duration(pd) * time.Millisecond
		}
		
		pulseCount := 5
		if pc, ok := args["pulse_count"].(float64); ok {
			pulseCount = int(pc)
		}
		
		// Create the pulse effect
		seq := scheduler.CreatePulseEffect(targetID, minBrightness, maxBrightness, pulseDuration, pulseCount)

		// Check if target is a group and convert if needed
		if isGroupID(ctx, hueClient, targetID) {
			seq = scheduler.CreateGroupEffect(seq, targetID)
		}

		seqID, err := globalScheduler.ExecuteSequence(seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start pulse effect: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Pulse effect started on %s\nSequence ID: %s\nBrightness: %.0f%% - %.0f%%\nPulses: %d", 
			targetID, seqID, minBrightness, maxBrightness, pulseCount)), nil
	}
}

// HandleColorLoopEffect creates a color loop effect
func HandleColorLoopEffect(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		targetID, ok := args["target_id"].(string)
		if !ok {
			return mcp.NewToolResultError("target_id is required"), nil
		}
		
		// Get colors array from JSON string
		colorsJSON, ok := args["colors"].(string)
		if !ok {
			// Use default rainbow colors
			colorsJSON = `["#FF0000","#FF7F00","#FFFF00","#00FF00","#0000FF","#4B0082","#9400D3"]`
		}
		
		var colors []string
		if err := json.Unmarshal([]byte(colorsJSON), &colors); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse colors JSON: %v", err)), nil
		}
		
		transitionTime := 1 * time.Second
		if tt, ok := args["transition_time_ms"].(float64); ok {
			transitionTime = time.Duration(tt) * time.Millisecond
		}
		
		// Create the color loop effect
		seq := scheduler.CreateColorLoopEffect(targetID, colors, transitionTime)

		// Check if target is a group and convert if needed
		if isGroupID(ctx, hueClient, targetID) {
			seq = scheduler.CreateGroupEffect(seq, targetID)
		}

		seqID, err := globalScheduler.ExecuteSequence(seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start color loop: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Color loop started on %s\nSequence ID: %s\nColors: %d\nTransition time: %v", 
			targetID, seqID, len(colors), transitionTime)), nil
	}
}

// HandleStrobeEffect creates a strobe effect
func HandleStrobeEffect(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		targetID, ok := args["target_id"].(string)
		if !ok {
			return mcp.NewToolResultError("target_id is required"), nil
		}
		
		color, ok := args["color"].(string)
		if !ok {
			color = "#FFFFFF" // Default to white
		}
		
		strobeRate := 100 * time.Millisecond
		if sr, ok := args["strobe_rate_ms"].(float64); ok {
			strobeRate = time.Duration(sr) * time.Millisecond
		}
		
		duration := 5 * time.Second
		if d, ok := args["duration_ms"].(float64); ok {
			duration = time.Duration(d) * time.Millisecond
		}
		
		// Create the strobe effect
		seq := scheduler.CreateStrobeEffect(targetID, color, strobeRate, duration)

		// Check if target is a group and convert if needed
		if isGroupID(ctx, hueClient, targetID) {
			seq = scheduler.CreateGroupEffect(seq, targetID)
		}

		seqID, err := globalScheduler.ExecuteSequence(seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start strobe effect: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Strobe effect started on %s\nSequence ID: %s\nColor: %s\nRate: %v", 
			targetID, seqID, color, strobeRate)), nil
	}
}

// HandleAlertEffect creates an alert effect
func HandleAlertEffect(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		targetID, ok := args["target_id"].(string)
		if !ok {
			return mcp.NewToolResultError("target_id is required"), nil
		}
		
		alertColor, ok := args["alert_color"].(string)
		if !ok {
			alertColor = "#FF0000" // Default to red
		}
		
		normalColor, ok := args["normal_color"].(string)
		if !ok {
			normalColor = "#FFFFFF" // Default to white
		}
		
		// Create the alert effect
		seq := scheduler.CreateAlertEffect(targetID, alertColor, normalColor)

		// Check if target is a group and convert if needed
		if isGroupID(ctx, hueClient, targetID) {
			seq = scheduler.CreateGroupEffect(seq, targetID)
		}

		seqID, err := globalScheduler.ExecuteSequence(seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start alert effect: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Alert effect started on %s\nSequence ID: %s\nAlert color: %s", 
			targetID, seqID, alertColor)), nil
	}
}

// HandleStopSequence stops one or more running sequences
func HandleStopSequence(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		// Try to get sequence_ids first (array format)
		if sequenceIDsJSON, ok := args["sequence_ids"].(string); ok {
			// Parse JSON array of IDs
			var sequenceIDs []string
			if err := json.Unmarshal([]byte(sequenceIDsJSON), &sequenceIDs); err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Failed to parse sequence_ids JSON: %v", err)), nil
			}
			
			// Stop all sequences
			var stopped []string
			var failed []string
			
			for _, id := range sequenceIDs {
				err := globalScheduler.StopSequence(id)
				if err != nil {
					failed = append(failed, fmt.Sprintf("%s (%v)", id, err))
				} else {
					stopped = append(stopped, id)
				}
			}
			
			// Build response
			var result strings.Builder
			if len(stopped) > 0 {
				result.WriteString(fmt.Sprintf("Stopped %d sequences:\n", len(stopped)))
				for _, id := range stopped {
					result.WriteString(fmt.Sprintf("✅ %s\n", id))
				}
			}
			if len(failed) > 0 {
				result.WriteString(fmt.Sprintf("\nFailed to stop %d sequences:\n", len(failed)))
				for _, failure := range failed {
					result.WriteString(fmt.Sprintf("❌ %s\n", failure))
				}
			}
			
			return mcp.NewToolResultText(result.String()), nil
		}
		
		// Fall back to single sequence_id for backward compatibility
		sequenceID, ok := args["sequence_id"].(string)
		if !ok {
			return mcp.NewToolResultError("sequence_id or sequence_ids is required"), nil
		}
		
		err := globalScheduler.StopSequence(sequenceID)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to stop sequence: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Sequence %s stopped", sequenceID)), nil
	}
}

// HandleListSequences lists all sequences
func HandleListSequences(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		sequences := globalScheduler.GetSequences()
		
		if len(sequences) == 0 {
			return mcp.NewToolResultText("No active sequences"), nil
		}
		
		result := fmt.Sprintf("Active sequences (%d):\n", len(sequences))
		for id, seq := range sequences {
			status := "stopped"
			if seq.Running {
				status = "running"
			}
			result += fmt.Sprintf("- %s: %s [%s]\n", id, seq.Name, status)
		}
		
		return mcp.NewToolResultText(result), nil
	}
}

// HandleCustomSequence executes a custom sequence from JSON
func HandleCustomSequence(hueClient *client.Client) server.ToolHandlerFunc {
	return func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		args := request.GetArguments()
		
		sequenceJSON, ok := args["sequence"].(string)
		if !ok {
			return mcp.NewToolResultError("sequence JSON is required"), nil
		}
		
		var seq scheduler.Sequence
		if err := json.Unmarshal([]byte(sequenceJSON), &seq); err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to parse sequence JSON: %v", err)), nil
		}
		
		if seq.Name == "" {
			seq.Name = "Custom Sequence"
		}
		
		seqID, err := globalScheduler.ExecuteSequence(&seq)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to start custom sequence: %v", err)), nil
		}
		
		return mcp.NewToolResultText(fmt.Sprintf("Custom sequence started: %s\nSequence ID: %s\nCommands: %d\nLoop: %v", 
			seq.Name, seqID, len(seq.Commands), seq.Loop)), nil
	}
}