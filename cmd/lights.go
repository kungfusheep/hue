package cmd

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
)

var (
	turnOnWithColor bool
)

// lightsCmd represents the lights command group
var lightsCmd = &cobra.Command{
	Use:   "lights",
	Short: "Control individual lights",
	Long:  `Commands for controlling individual Philips Hue lights.`,
}

// listLightsCmd lists all available lights
var listLightsCmd = &cobra.Command{
	Use:   "list",
	Short: "List all available lights",
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		lights, err := hueClient.GetLights(ctx)
		if err != nil {
			return fmt.Errorf("failed to get lights: %w", err)
		}

		if jsonOutput {
			printJSON(lights)
			return nil
		}

		// Human-readable output
		fmt.Printf("Found %d lights:\n\n", len(lights))
		for _, light := range lights {
			status := "off"
			if light.On.On {
				status = fmt.Sprintf("on (brightness: %.0f%%)", light.Dimming.Brightness)
			}
			fmt.Printf("%-30s %s\n", light.Metadata.Name, status)
			fmt.Printf("  ID: %s\n", light.ID)
			fmt.Printf("  Type: %s\n", light.Metadata.Archetype)
			if light.Color != nil {
				fmt.Printf("  Color: X=%.3f Y=%.3f\n", light.Color.XY.X, light.Color.XY.Y)
			}
			fmt.Println()
		}
		return nil
	},
}

// lightOnCmd turns a light on
var lightOnCmd = &cobra.Command{
	Use:   "on <light-name-or-id>",
	Short: "Turn a light on",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.TurnOnLight(ctx, lightID)
		if err != nil {
			return fmt.Errorf("failed to turn on light: %w", err)
		}
		
		printMessage("Light %s turned on", args[0])
		return nil
	},
}

// lightOffCmd turns a light off
var lightOffCmd = &cobra.Command{
	Use:   "off <light-name-or-id>",
	Short: "Turn a light off",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		
		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.TurnOffLight(ctx, lightID)
		if err != nil {
			return fmt.Errorf("failed to turn off light: %w", err)
		}
		
		printMessage("Light %s turned off", args[0])
		return nil
	},
}

// lightColorCmd sets light color
var lightColorCmd = &cobra.Command{
	Use:   "color <light-name-or-id> <color>",
	Short: "Set light color (hex or name)",
	Long:  `Set light color using hex code (#FF0000) or color name (red, blue, green, etc.)

Use the --turn-on flag to turn on the light while setting its color (atomic operation).`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		color := args[1]
		ctx := context.Background()

		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}

		// Convert color name to hex if needed
		hexColor := namedColorToHex(color)
		if hexColor == "" {
			hexColor = color
		}

		if turnOnWithColor {
			// Set color and turn on in a single atomic operation
			err = hueClient.SetLightColorAndTurnOn(ctx, lightID, hexColor)
			if err != nil {
				return fmt.Errorf("failed to set color and turn on: %w", err)
			}
			printMessage("Light %s turned on and color set to %s", args[0], color)
		} else {
			// Just set the color
			err = hueClient.SetLightColor(ctx, lightID, hexColor)
			if err != nil {
				return fmt.Errorf("failed to set color: %w", err)
			}
			printMessage("Light %s color set to %s", args[0], color)
		}

		return nil
	},
}

// lightBrightnessCmd sets light brightness
var lightBrightnessCmd = &cobra.Command{
	Use:   "brightness <light-name-or-id> <percent>",
	Short: "Set light brightness (0-100)",
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
		
		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}
		
		err = hueClient.SetLightBrightness(ctx, lightID, brightness)
		if err != nil {
			return fmt.Errorf("failed to set brightness: %w", err)
		}
		
		printMessage("Light %s brightness set to %.0f%%", args[0], brightness)
		return nil
	},
}

// lightStateCmd shows current state of a light
var lightStateCmd = &cobra.Command{
	Use:   "state <light-name-or-id>",
	Short: "Show current state of a light",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}

		light, err := hueClient.GetLight(ctx, lightID)
		if err != nil {
			return fmt.Errorf("failed to get light: %w", err)
		}

		if jsonOutput {
			printJSON(light)
			return nil
		}

		// Human-readable output
		fmt.Printf("Light: %s\n", light.Metadata.Name)
		fmt.Printf("Type: %s\n", light.Metadata.Archetype)
		fmt.Printf("On: %v\n", light.On.On)
		if light.On.On {
			fmt.Printf("Brightness: %.0f%%\n", light.Dimming.Brightness)
		}
		if light.Color != nil {
			fmt.Printf("Color XY: (%.3f, %.3f)\n", light.Color.XY.X, light.Color.XY.Y)
		}
		if light.ColorTemperature != nil && light.ColorTemperature.MirekValid {
			fmt.Printf("Color Temperature: %d mirek\n", light.ColorTemperature.Mirek)
		}
		if light.Effects != nil && light.Effects.Effect != "" {
			fmt.Printf("Effect: %s\n", light.Effects.Effect)
		}

		return nil
	},
}

// lightEffectsListCmd lists available effects for a light
var lightEffectsListCmd = &cobra.Command{
	Use:   "effects-list <light-name-or-id>",
	Short: "List available built-in effects for a light",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}

		light, err := hueClient.GetLight(ctx, lightID)
		if err != nil {
			return fmt.Errorf("failed to get light: %w", err)
		}

		if light.Effects == nil || len(light.Effects.EffectValues) == 0 {
			printMessage("No effects available for this light")
			return nil
		}

		if jsonOutput {
			printJSON(light.Effects.EffectValues)
			return nil
		}

		fmt.Printf("Available effects for %s:\n\n", light.Metadata.Name)
		for _, effect := range light.Effects.EffectValues {
			currentMarker := ""
			if light.Effects.Effect == effect {
				currentMarker = " (current)"
			}
			fmt.Printf("  - %s%s\n", effect, currentMarker)
		}

		return nil
	},
}

// lightEffectCmd sets a built-in effect on a light
var lightEffectCmd = &cobra.Command{
	Use:   "effect <light-name-or-id> <effect-name>",
	Short: "Set a built-in effect on a light (fire, candle, sparkle, etc.)",
	Long:  `Set a built-in Hue effect on a light. Use 'hue lights effects-list' to see available effects.

Common effects include: fire, candle, sparkle, opal, glisten, no_effect (to stop)`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		effectName := args[1]

		// Resolve light name to ID
		lightID, err := resolveLightID(ctx, args[0])
		if err != nil {
			return err
		}

		// Set effect with 0 duration (indefinite)
		err = hueClient.SetLightEffect(ctx, lightID, effectName, 0)
		if err != nil {
			return fmt.Errorf("failed to set effect: %w", err)
		}

		printMessage("Effect '%s' set on %s", effectName, args[0])
		return nil
	},
}

func init() {
	// Add flags
	lightColorCmd.Flags().BoolVar(&turnOnWithColor, "turn-on", false, "Turn on the light when setting color (atomic operation)")

	// Add subcommands
	lightsCmd.AddCommand(listLightsCmd)
	lightsCmd.AddCommand(lightOnCmd)
	lightsCmd.AddCommand(lightOffCmd)
	lightsCmd.AddCommand(lightColorCmd)
	lightsCmd.AddCommand(lightBrightnessCmd)
	lightsCmd.AddCommand(lightStateCmd)
	lightsCmd.AddCommand(lightEffectsListCmd)
	lightsCmd.AddCommand(lightEffectCmd)

	// Add to root
	rootCmd.AddCommand(lightsCmd)
}