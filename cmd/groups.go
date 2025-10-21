package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	turnOnGroupWithColor bool
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
	Use:   "on <group-name-or-id>",
	Short: "Turn a group on",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		// Resolve group name to ID
		groupID, err := resolveGroupID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.TurnOnGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("failed to turn on group: %w", err)
		}
		
		printMessage("Group %s turned on", args[0])
		return nil
	},
}

// groupOffCmd turns a group off
var groupOffCmd = &cobra.Command{
	Use:   "off <group-name-or-id>",
	Short: "Turn a group off",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		// Resolve group name to ID
		groupID, err := resolveGroupID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.TurnOffGroup(ctx, groupID)
		if err != nil {
			return fmt.Errorf("failed to turn off group: %w", err)
		}
		
		printMessage("Group %s turned off", args[0])
		return nil
	},
}

// groupColorCmd sets group color
var groupColorCmd = &cobra.Command{
	Use:   "color <group-name-or-id> <color>",
	Short: "Set group color (hex or name)",
	Long:  `Set group color using hex code (#FF0000) or color name (red, blue, green, etc.)

Use the --turn-on flag to turn on all lights in the group while setting color (atomic operation).`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		color := args[1]
		ctx := context.Background()

		// Resolve group name to ID
		groupID, err := resolveGroupID(ctx, args[0])
		if err != nil {
			return err
		}

		// Convert color name to hex if needed
		hexColor := namedColorToHex(color)
		if hexColor == "" {
			hexColor = color
		}

		if turnOnGroupWithColor {
			// Set color and turn on in a single atomic operation
			err = hueClient.SetGroupColorAndTurnOn(ctx, groupID, hexColor)
			if err != nil {
				return fmt.Errorf("failed to set color and turn on: %w", err)
			}
			printMessage("Group %s turned on and color set to %s", args[0], color)
		} else {
			// Just set the color
			err = hueClient.SetGroupColor(ctx, groupID, hexColor)
			if err != nil {
				return fmt.Errorf("failed to set color: %w", err)
			}
			printMessage("Group %s color set to %s", args[0], color)
		}

		return nil
	},
}

// groupBrightnessCmd sets group brightness
var groupBrightnessCmd = &cobra.Command{
	Use:   "brightness <group-name-or-id> <percent>",
	Short: "Set group brightness (0-100)",
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
		
		// Resolve group name to ID
		groupID, err := resolveGroupID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.SetGroupBrightness(ctx, groupID, brightness)
		if err != nil {
			return fmt.Errorf("failed to set brightness: %w", err)
		}
		
		printMessage("Group %s brightness set to %.0f%%", args[0], brightness)
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