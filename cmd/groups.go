package cmd

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

var (
	turnOnGroupWithColor bool
	groupTransitionMs    int
)

// groupsCmd represents the groups command group
var groupsCmd = &cobra.Command{
	Use:   "groups",
	Short: "Control light groups",
	Long:  `Commands for controlling groups of lights (rooms, zones, etc).`,
}

// listGroupsCmd lists all available groups
var listGroupsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available groups",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		groups, err := hueClient.GetGroups(ctx)
		if err != nil {
			return fmt.Errorf("failed to get groups: %w", err)
		}

		if jsonOutput {
			printJSON(groups)
			return nil
		}

		// Human-readable output
		fmt.Printf("Found %d groups:\n\n", len(groups))
		for _, group := range groups {
			status := "off"
			if group.On.On {
				status = fmt.Sprintf("on (brightness: %.0f%%)", group.Dimming.Brightness)
			}
			fmt.Printf("%-30s %s\n", group.Metadata.Name, status)
			fmt.Printf("  ID: %s\n", group.ID)
			fmt.Printf("  Type: %s\n", group.Type)
			fmt.Println()
		}
		return nil
	},
}

// groupOnCmd turns a group on
var groupOnCmd = &cobra.Command{
	Use:   "on <group-name-or-pattern>",
	Short: "Turn a group on",
	Long: `Turn one or more groups on.

Supports patterns using @"..." syntax:
  hue groups on @"bedroom"
  hue groups on @"*stairs"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve group name/pattern to IDs
		groupIDs, err := resolveGroupIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched groups concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, groupID := range groupIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.TurnOnGroup(ctx, id)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(groupID)
		}

		wg.Wait()

		// Report results
		if len(groupIDs) > 1 {
			printMessage("✓ Turned on %d/%d groups", successCount, len(groupIDs))
		} else if successCount > 0 {
			printMessage("Group %s turned on", args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d groups failed", len(errors))
		}

		return nil
	},
}

// groupOffCmd turns a group off
var groupOffCmd = &cobra.Command{
	Use:   "off <group-name-or-pattern>",
	Short: "Turn a group off",
	Long: `Turn one or more groups off.

Supports patterns using @"..." syntax:
  hue groups off @"bedroom"
  hue groups off @"down*"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve group name/pattern to IDs
		groupIDs, err := resolveGroupIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched groups concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, groupID := range groupIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.TurnOffGroup(ctx, id)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(groupID)
		}

		wg.Wait()

		// Report results
		if len(groupIDs) > 1 {
			printMessage("✓ Turned off %d/%d groups", successCount, len(groupIDs))
		} else if successCount > 0 {
			printMessage("Group %s turned off", args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d groups failed", len(errors))
		}

		return nil
	},
}

// groupColorCmd sets group color
var groupColorCmd = &cobra.Command{
	Use:   "color <group-name-or-pattern> <color>",
	Short: "Set group color (hex or name)",
	Long:  `Set group color using hex code (#FF0000) or color name (red, blue, green, etc.)

Use the --turn-on flag to turn on all lights in the group while setting color (atomic operation).
Use --transition to smoothly fade to the target color over time.

Supports patterns using @"..." syntax:
  hue groups color @"bedroom" red
  hue groups color @"down*" blue --transition 2000`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		color := args[1]
		ctx := context.Background()

		// Resolve group name/pattern to IDs
		groupIDs, err := resolveGroupIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Convert color name to hex if needed
		hexColor := namedColorToHex(color)
		if hexColor == "" {
			hexColor = color
		}

		// Apply to all matched groups concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, groupID := range groupIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var err error
				if turnOnGroupWithColor {
					err = hueClient.SetGroupColorAndTurnOn(ctx, id, hexColor)
				} else {
					err = hueClient.SetGroupColorWithTransition(ctx, id, hexColor, groupTransitionMs)
				}

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(groupID)
		}

		wg.Wait()

		// Report results
		if len(groupIDs) > 1 {
			if groupTransitionMs > 0 {
				printMessage("✓ Fading color to %s over %dms on %d/%d groups", color, groupTransitionMs, successCount, len(groupIDs))
			} else if turnOnGroupWithColor {
				printMessage("✓ Set color to %s and turned on %d/%d groups", color, successCount, len(groupIDs))
			} else {
				printMessage("✓ Set color to %s on %d/%d groups", color, successCount, len(groupIDs))
			}
		} else if successCount > 0 {
			if groupTransitionMs > 0 {
				printMessage("Group %s fading to %s over %dms", args[0], color, groupTransitionMs)
			} else if turnOnGroupWithColor {
				printMessage("Group %s turned on and color set to %s", args[0], color)
			} else {
				printMessage("Group %s color set to %s", args[0], color)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d groups failed", len(errors))
		}

		return nil
	},
}

// groupBrightnessCmd sets group brightness
var groupBrightnessCmd = &cobra.Command{
	Use:   "brightness <group-name-or-pattern> <percent>",
	Short: "Set group brightness (0-100)",
	Long: `Set brightness for one or more groups.

Use --transition to smoothly fade to the target brightness over time.

Supports patterns using @"..." syntax:
  hue groups brightness @"bedroom" 50
  hue groups brightness @"*stairs" 20 --transition 3000`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		brightness, err := strconv.ParseFloat(args[1], 64)
		if err != nil {
			return fmt.Errorf("invalid brightness value: %w", err)
		}

		if brightness < 0 || brightness > 100 {
			return fmt.Errorf("brightness must be between 0 and 100")
		}

		ctx := context.Background()

		// Resolve group name/pattern to IDs
		groupIDs, err := resolveGroupIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched groups concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, groupID := range groupIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.SetGroupBrightnessWithTransition(ctx, id, brightness, groupTransitionMs)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(groupID)
		}

		wg.Wait()

		// Report results
		if len(groupIDs) > 1 {
			if groupTransitionMs > 0 {
				printMessage("✓ Fading brightness to %.0f%% over %dms on %d/%d groups", brightness, groupTransitionMs, successCount, len(groupIDs))
			} else {
				printMessage("✓ Set brightness to %.0f%% on %d/%d groups", brightness, successCount, len(groupIDs))
			}
		} else if successCount > 0 {
			if groupTransitionMs > 0 {
				printMessage("Group %s fading to %.0f%% over %dms", args[0], brightness, groupTransitionMs)
			} else {
				printMessage("Group %s brightness set to %.0f%%", args[0], brightness)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d groups failed", len(errors))
		}

		return nil
	},
}

// listRoomsCmd lists all rooms
var listRoomsCmd = &cobra.Command{
	Use:   "rooms",
	Short: "List all rooms with their lights",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		rooms, err := hueClient.GetRooms(ctx)
		if err != nil {
			return fmt.Errorf("failed to get rooms: %w", err)
		}

		if jsonOutput {
			printJSON(rooms)
			return nil
		}

		// Human-readable output
		fmt.Printf("Found %d rooms:\n\n", len(rooms))
		for _, room := range rooms {
			fmt.Printf("🏠 %s\n", room.Metadata.Name)
			fmt.Printf("   ID: %s\n", room.ID)
			fmt.Printf("   Archetype: %s\n", room.Metadata.Archetype)
			
			// List devices in room
			if len(room.Children) > 0 {
				fmt.Printf("   Devices: %d\n", len(room.Children))
				for _, child := range room.Children {
					fmt.Printf("     - %s (%s)\n", child.RID, child.RType)
				}
			}
			fmt.Println()
		}
		return nil
	},
}

func init() {
	// Add flags
	groupColorCmd.Flags().BoolVar(&turnOnGroupWithColor, "turn-on", false, "Turn on the group when setting color (atomic operation)")
	groupColorCmd.Flags().IntVar(&groupTransitionMs, "transition", 0, "Transition time in milliseconds (e.g., 2000 for 2 seconds)")
	groupBrightnessCmd.Flags().IntVar(&groupTransitionMs, "transition", 0, "Transition time in milliseconds (e.g., 2000 for 2 seconds)")

	// Add subcommands
	groupsCmd.AddCommand(listGroupsCmd)
	groupsCmd.AddCommand(groupOnCmd)
	groupsCmd.AddCommand(groupOffCmd)
	groupsCmd.AddCommand(groupColorCmd)
	groupsCmd.AddCommand(groupBrightnessCmd)
	groupsCmd.AddCommand(listRoomsCmd)

	// Add to root
	rootCmd.AddCommand(groupsCmd)
}