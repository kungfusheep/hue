package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/kungfusheep/hue/client"
)

// scenesCmd represents the native Hue scenes command group
var scenesCmd = &cobra.Command{
	Use:   "scenes",
	Short: "Manage native Hue scenes",
	Long:  `Commands for managing native Philips Hue scenes.`,
}

var (
	showIDs           bool
	showActions       bool
	showGroups        bool
	dynamicMode       bool
	sceneSpeed        float64
	sceneTransitionMs int
)

// listHueScenesCmd lists all native Hue scenes
var listHueScenesCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List all native Hue scenes (optionally filtered)",
	Long: `List all scenes or filter using simple pattern matching.

Examples:
  hue scenes list                  # all scenes
  hue scenes list relax            # scenes with "relax" in name
  hue scenes list @"*bright*"      # scenes with "bright" in name
  hue scenes list @"room:bedroom"  # scenes for bedroom`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		allScenes, err := hueClient.GetScenes(ctx)
		if err != nil {
			return fmt.Errorf("failed to list scenes: %w", err)
		}

		// Get rooms to map group IDs to room names
		rooms, err := hueClient.GetRooms(ctx)
		if err != nil {
			return fmt.Errorf("failed to get rooms: %w", err)
		}

		// Create a map of room ID to room name
		roomIDToName := make(map[string]string)
		for _, room := range rooms {
			roomIDToName[room.ID] = room.Metadata.Name
		}

		// Also get zones if any
		zones, err := hueClient.GetZones(ctx)
		if err == nil {
			for _, zone := range zones {
				roomIDToName[zone.ID] = zone.Metadata.Name
			}
		}

		// Filter scenes if query/pattern provided
		var scenes []client.Scene
		if len(args) > 0 {
			pattern := strings.ToLower(args[0])
			roomFilter := ""

			// Check for room: prefix
			if strings.HasPrefix(pattern, "@\"room:") {
				pattern = strings.TrimPrefix(pattern, "@\"room:")
				pattern = strings.TrimSuffix(pattern, "\"")
				roomFilter = pattern
				pattern = ""
			} else if strings.HasPrefix(pattern, "@\"") {
				// Remove @ and quotes for pattern matching
				pattern = strings.TrimPrefix(pattern, "@\"")
				pattern = strings.TrimSuffix(pattern, "\"")
			}

			// Filter scenes
			for _, scene := range allScenes {
				sceneName := strings.ToLower(scene.Metadata.Name)
				roomName := ""
				if scene.Group.RType == "room" || scene.Group.RType == "zone" {
					roomName = strings.ToLower(roomIDToName[scene.Group.RID])
				}

				match := false
				if roomFilter != "" {
					// Room filter mode
					match = strings.Contains(roomName, roomFilter)
				} else if pattern != "" {
					// Pattern matching (with wildcards)
					if strings.Contains(pattern, "*") {
						// Simple wildcard matching
						if strings.HasPrefix(pattern, "*") && strings.HasSuffix(pattern, "*") {
							inner := strings.Trim(pattern, "*")
							match = strings.Contains(sceneName, inner)
						} else if strings.HasPrefix(pattern, "*") {
							suffix := strings.TrimPrefix(pattern, "*")
							match = strings.HasSuffix(sceneName, suffix)
						} else if strings.HasSuffix(pattern, "*") {
							prefix := strings.TrimSuffix(pattern, "*")
							match = strings.HasPrefix(sceneName, prefix)
						}
					} else {
						// Substring match
						match = strings.Contains(sceneName, pattern)
					}
				}

				if match {
					scenes = append(scenes, scene)
				}
			}

			if len(scenes) == 0 {
				printMessage("No scenes matched pattern: %s", args[0])
				return nil
			}
		} else {
			scenes = allScenes
		}

		if jsonOutput {
			printJSON(scenes)
			return nil
		}

		// Human-readable output
		if len(args) > 0 {
			fmt.Printf("Pattern '%s' matched %d scene(s):\n\n", args[0], len(scenes))
		} else {
			fmt.Printf("Found %d Hue scenes:\n\n", len(scenes))
		}

		for _, scene := range scenes {
			// Basic output - scene name and room
			roomName := ""
			if scene.Group.RType == "room" || scene.Group.RType == "zone" {
				roomName = roomIDToName[scene.Group.RID]
			}
			
			if roomName != "" {
				fmt.Printf("📋 %s (%s)\n", scene.Metadata.Name, roomName)
			} else {
				fmt.Printf("📋 %s\n", scene.Metadata.Name)
			}
			
			// Optional: show IDs
			if showIDs {
				fmt.Printf("   ID: %s\n", scene.ID)
				if scene.IDV1 != "" {
					fmt.Printf("   V1 ID: %s\n", scene.IDV1)
				}
			}
			
			// Optional: show group ID
			if showGroups && scene.Group.RID != "" {
				fmt.Printf("   Group ID: %s\n", scene.Group.RID)
			}
			
			// Optional: show action count
			if showActions {
				fmt.Printf("   Actions: %d\n", len(scene.Actions))
			}
			
			fmt.Println()
		}
		
		return nil
	},
}

// activateHueSceneCmd activates a native Hue scene
var activateHueSceneCmd = &cobra.Command{
	Use:   "activate <scene-name-or-id>",
	Short: "Activate a native Hue scene",
	Long: `Activate a native Hue scene by name or ID. For scenes with the same name in different rooms, use 'SceneName:RoomName' format (e.g., 'Nightlight:Master Bedroom').

Use --transition to smoothly fade to the scene over time.
Use --dynamic to activate animated/dynamic scenes that cycle through colors.
Use --speed to control animation speed (0.0 to 1.0, where 1.0 is fastest).

Examples:
  hue scenes activate "Relax"
  hue scenes activate "Concentrate:Office" --transition 3000
  hue scenes activate "Energize" --dynamic --speed 0.5`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve scene name to ID
		sceneID, err := resolveSceneID(ctx, args[0])
		if err != nil {
			return err
		}

		// Update speed if specified (must be done before activation)
		if sceneSpeed > 0 {
			// Validate speed is between 0.0 and 1.0
			if sceneSpeed < 0.0 || sceneSpeed > 1.0 {
				return fmt.Errorf("speed must be between 0.0 and 1.0")
			}
			err = hueClient.UpdateSceneSpeed(ctx, sceneID, sceneSpeed)
			if err != nil {
				return fmt.Errorf("failed to update scene speed: %w", err)
			}
			printMessage("Scene speed set to %.2f", sceneSpeed)
		}

		// Activate the scene
		if dynamicMode {
			err = hueClient.ActivateSceneDynamic(ctx, sceneID)
			if err != nil {
				return fmt.Errorf("failed to activate dynamic scene: %w", err)
			}
			if sceneSpeed > 0 {
				printMessage("Scene %s activated in dynamic mode with speed %.2f", args[0], sceneSpeed)
			} else {
				printMessage("Scene %s activated in dynamic mode", args[0])
			}
		} else {
			err = hueClient.ActivateSceneWithTransition(ctx, sceneID, sceneTransitionMs)
			if err != nil {
				return fmt.Errorf("failed to activate scene: %w", err)
			}
			if sceneTransitionMs > 0 {
				printMessage("Scene %s fading over %dms", args[0], sceneTransitionMs)
			} else {
				printMessage("Scene %s activated", args[0])
			}
		}

		return nil
	},
}

// createHueSceneCmd creates a new Hue scene
var createHueSceneCmd = &cobra.Command{
	Use:   "create <name> <group-name-or-id>",
	Short: "Create a new Hue scene for a group",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		name := args[0]
		ctx := context.Background()
		
		// Resolve group name to ID
		groupID, err := resolveGroupID(ctx, args[1])
		if err != nil {
			return err
		}
		
		// Note: This creates an empty scene. In practice, you'd want to
		// capture current light states or specify actions
		sceneCreate := client.SceneCreate{
			Type: "scene",
			Metadata: client.Metadata{
				Name: name,
			},
			Group: client.ResourceIdentifier{
				RID:   groupID,
				RType: "grouped_light",
			},
			Actions: []client.SceneAction{}, // Empty for now
		}
		
		scene, err := hueClient.CreateScene(ctx, sceneCreate)
		
		if err != nil {
			return fmt.Errorf("failed to create scene: %w", err)
		}
		
		printMessage("Scene '%s' created with ID: %s", name, scene.ID)
		return nil
	},
}

// findHueSceneCmd finds scenes by name
var findHueSceneCmd = &cobra.Command{
	Use:   "find <search-term>",
	Short: "Find scenes by name",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		searchTerm := strings.ToLower(args[0])
		ctx := context.Background()
		
		scenes, err := hueClient.GetScenes(ctx)
		if err != nil {
			return fmt.Errorf("failed to list scenes: %w", err)
		}

		// Filter scenes by name
		matchCount := 0
		fmt.Printf("Scenes matching '%s':\n\n", searchTerm)
		
		for _, scene := range scenes {
			if strings.Contains(strings.ToLower(scene.Metadata.Name), searchTerm) {
				fmt.Printf("- %s (ID: %s)\n", scene.Metadata.Name, scene.ID)
				matchCount++
			}
		}

		if matchCount == 0 {
			fmt.Printf("No scenes found matching '%s'\n", searchTerm)
			return nil
		}
		
		return nil
	},
}

func init() {
	// Add flags to list command
	listHueScenesCmd.Flags().BoolVar(&showIDs, "show-ids", false, "Show scene IDs")
	listHueScenesCmd.Flags().BoolVar(&showActions, "show-actions", false, "Show action counts")
	listHueScenesCmd.Flags().BoolVar(&showGroups, "show-groups", false, "Show group IDs")

	// Add flags to activate command
	activateHueSceneCmd.Flags().BoolVarP(&dynamicMode, "dynamic", "d", false, "Activate scene in dynamic/animated mode")
	activateHueSceneCmd.Flags().Float64VarP(&sceneSpeed, "speed", "s", 0.0, "Animation speed (0.0-1.0, requires --dynamic)")
	activateHueSceneCmd.Flags().IntVar(&sceneTransitionMs, "transition", 0, "Transition time in milliseconds (e.g., 3000 for 3 seconds)")

	// Add subcommands
	scenesCmd.AddCommand(listHueScenesCmd)
	scenesCmd.AddCommand(activateHueSceneCmd)
	scenesCmd.AddCommand(createHueSceneCmd)
	scenesCmd.AddCommand(findHueSceneCmd)

	// Add to root
	rootCmd.AddCommand(scenesCmd)
}