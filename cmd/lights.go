package cmd

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/query"
	"github.com/spf13/cobra"
)

var (
	turnOnWithColor      bool
	turnOnWithBrightness bool
	transitionMs         int // Transition time in milliseconds
)

// lightsCmd represents the lights command group
var lightsCmd = &cobra.Command{
	Use:   "lights",
	Short: "Control individual lights",
	Long:  `Commands for controlling individual Philips Hue lights.`,
}

// listLightsCmd lists all available lights
var listLightsCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List all available lights (optionally filtered by query)",
	Long: `List all lights or filter using query syntax.

Examples:
  hue lights list                    # all lights
  hue lights list @"office"          # office lights only
  hue lights list @"on:"             # currently on lights
  hue lights list @"with:effects"    # effect-capable lights
  hue lights list @"type:go,strip"   # go or strip lights`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		allLights, err := hueClient.GetLights(ctx)
		if err != nil {
			return fmt.Errorf("failed to get lights: %w", err)
		}

		// Filter lights if query provided
		var lights []client.Light
		if len(args) > 0 && query.IsQuery(args[0]) {
			// Build query context with room/zone data (only if needed)
			var queryCtx *query.Context

			if query.NeedsRoomContext(args[0]) {
				queryCtx = &query.Context{}
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
				q, err := query.Parse(args[0])
				if err != nil {
					return err
				}
				matched := query.ExecuteWithContext(q, allLights, queryCtx)
				result := &query.DryRunResult{
					Query:   args[0],
					Matched: matched,
					Count:   len(matched),
				}
				fmt.Print(query.FormatDryRun(result, verbose))
				return nil
			}

			// Execute query with context
			q, err := query.Parse(args[0])
			if err != nil {
				return fmt.Errorf("failed to parse query: %w", err)
			}
			lights = query.ExecuteWithContext(q, allLights, queryCtx)

			if len(lights) == 0 {
				printMessage("No lights matched query: %s", args[0])
				return nil
			}
		} else if len(args) > 0 {
			// Single light name/ID lookup
			lightID, err := resolveLightID(ctx, args[0])
			if err != nil {
				return err
			}
			light, err := hueClient.GetLight(ctx, lightID)
			if err != nil {
				return fmt.Errorf("failed to get light: %w", err)
			}
			lights = []client.Light{*light}
		} else {
			// No filter, show all
			lights = allLights
		}

		if jsonOutput {
			printJSON(lights)
			return nil
		}

		// Human-readable output
		if len(args) > 0 && query.IsQuery(args[0]) {
			fmt.Printf("Query '%s' matched %d light(s):\n\n", args[0], len(lights))
		} else {
			fmt.Printf("Found %d lights:\n\n", len(lights))
		}

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
			if light.Effects != nil && light.Effects.Effect != "" && light.Effects.Effect != "no_effect" {
				fmt.Printf("  Effect: %s\n", light.Effects.Effect)
			}
			fmt.Println()
		}
		return nil
	},
}

// lightOnCmd turns a light on
var lightOnCmd = &cobra.Command{
	Use:   "on <light-name-or-query>",
	Short: "Turn a light on",
	Long: `Turn one or more lights on.

Supports queries using @"..." syntax:
  hue lights on @"office"
  hue lights on @"*lamp"
  hue lights on @"room:bedroom type:go"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.TurnOnLight(ctx, id)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			printMessage("✓ Turned on %d/%d lights", successCount, len(lightIDs))
		} else if successCount > 0 {
			printMessage("Light %s turned on", args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

// lightOffCmd turns a light off
var lightOffCmd = &cobra.Command{
	Use:   "off <light-name-or-query>",
	Short: "Turn a light off",
	Long: `Turn one or more lights off.

Supports queries using @"..." syntax:
  hue lights off @"office"
  hue lights off @"*lamp"
  hue lights off @"on: room:bedroom"`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.TurnOffLight(ctx, id)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			printMessage("✓ Turned off %d/%d lights", successCount, len(lightIDs))
		} else if successCount > 0 {
			printMessage("Light %s turned off", args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

// lightColorCmd sets light color
var lightColorCmd = &cobra.Command{
	Use:   "color <light-name-or-query> <color>",
	Short: "Set light color (hex, name, or xy)",
	Long:  `Set light color using hex code (#FF0000), color name (red, blue, etc.), or XY coordinates (0.5,0.4).

Use the --turn-on flag to turn on the light while setting its color (atomic operation).
Use --transition to smoothly fade to the target color over time.

Supports queries using @"..." syntax:
  hue lights color @"office" red
  hue lights color @"*lamp" "#FF5500" --transition 2000
  hue lights color @"room:bedroom" 0.5,0.4`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		color := args[1]
		ctx := context.Background()

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Check if color is XY format (e.g., "0.5,0.4")
		var isXY bool
		var xyX, xyY float64
		if parts := splitXY(color); parts != nil {
			isXY = true
			xyX, xyY = parts[0], parts[1]
		}

		// Convert color name to hex if needed (and not XY)
		hexColor := color
		if !isXY {
			if named := namedColorToHex(color); named != "" {
				hexColor = named
			}
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				// Acquire semaphore
				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var err error
				if isXY {
					// Use XY directly (no transition support for XY yet)
					err = hueClient.SetLightColorXY(ctx, id, xyX, xyY)
				} else if turnOnWithColor {
					err = hueClient.SetLightColorAndTurnOn(ctx, id, hexColor)
				} else {
					err = hueClient.SetLightColorWithTransition(ctx, id, hexColor, transitionMs)
				}

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			if transitionMs > 0 {
				printMessage("✓ Fading color to %s over %dms on %d/%d lights", color, transitionMs, successCount, len(lightIDs))
			} else if turnOnWithColor {
				printMessage("✓ Set color to %s and turned on %d/%d lights", color, successCount, len(lightIDs))
			} else {
				printMessage("✓ Set color to %s on %d/%d lights", color, successCount, len(lightIDs))
			}
		} else if successCount > 0 {
			if transitionMs > 0 {
				printMessage("Light %s fading to %s over %dms", args[0], color, transitionMs)
			} else if turnOnWithColor {
				printMessage("Light %s turned on and color set to %s", args[0], color)
			} else {
				printMessage("Light %s color set to %s", args[0], color)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

// splitXY parses an XY color string like "0.5,0.4" and returns the coordinates
func splitXY(s string) []float64 {
	parts := strings.Split(s, ",")
	if len(parts) != 2 {
		return nil
	}
	x, err1 := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	y, err2 := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err1 != nil || err2 != nil {
		return nil
	}
	// Sanity check - XY values should be between 0 and 1
	if x < 0 || x > 1 || y < 0 || y > 1 {
		return nil
	}
	return []float64{x, y}
}

// lightBrightnessCmd sets light brightness
var lightBrightnessCmd = &cobra.Command{
	Use:   "brightness <light-name-or-query> <percent>",
	Short: "Set light brightness (0-100)",
	Long: `Set brightness for one or more lights.

Use the --turn-on flag to turn on the light while setting brightness (atomic operation).
Use --transition to smoothly fade to the target brightness over time.

Supports queries using @"..." syntax:
  hue lights brightness @"office" 80
  hue lights brightness @"*lamp" 50 --transition 2000
  hue lights brightness @"on: room:bedroom" 10 --turn-on`,
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

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				var err error
				if turnOnWithBrightness {
					err = hueClient.SetLightBrightnessAndTurnOn(ctx, id, brightness, transitionMs)
				} else {
					err = hueClient.SetLightBrightnessWithTransition(ctx, id, brightness, transitionMs)
				}

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			if transitionMs > 0 {
				printMessage("✓ Fading brightness to %.0f%% over %dms on %d/%d lights", brightness, transitionMs, successCount, len(lightIDs))
			} else if turnOnWithBrightness {
				printMessage("✓ Set brightness to %.0f%% and turned on %d/%d lights", brightness, successCount, len(lightIDs))
			} else {
				printMessage("✓ Set brightness to %.0f%% on %d/%d lights", brightness, successCount, len(lightIDs))
			}
		} else if successCount > 0 {
			if transitionMs > 0 {
				printMessage("Light %s fading to %.0f%% over %dms", args[0], brightness, transitionMs)
			} else if turnOnWithBrightness {
				printMessage("Light %s turned on and brightness set to %.0f%%", args[0], brightness)
			} else {
				printMessage("Light %s brightness set to %.0f%%", args[0], brightness)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

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
	Use:   "effect <light-name-or-query> <effect-name>",
	Short: "Set a built-in effect on a light (fire, candle, sparkle, etc.)",
	Long:  `Set a built-in Hue effect on one or more lights. Use 'hue lights effects-list' to see available effects.

Common effects include: fire, candle, sparkle, opal, glisten, no_effect (to stop)

Supports queries using @"..." syntax:
  hue lights effect @"with:effects" fire
  hue lights effect @"room:bedroom" candle
  hue lights effect @"office type:go" sparkle`,
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		effectName := args[1]

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.SetLightEffect(ctx, id, effectName, 0)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			printMessage("✓ Set effect '%s' on %d/%d lights", effectName, successCount, len(lightIDs))
		} else if successCount > 0 {
			printMessage("Effect '%s' set on %s", effectName, args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

// lightAlertCmd triggers an alert on a light
var lightAlertCmd = &cobra.Command{
	Use:   "alert <light-name-or-query> <action>",
	Short: "Trigger an alert effect (breathe or lselect)",
	Long: `Trigger a built-in alert effect on one or more lights.

Actions:
  breathe  - Single pulse/flash (great for notifications)
  lselect  - 15 seconds of pulsing (attention-grabbing)

Supports queries using @"..." syntax:
  hue lights alert @"office" breathe
  hue lights alert "Desk Lamp" lselect`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := context.Background()
		action := args[1]

		// Validate action
		if action != "breathe" && action != "lselect" {
			return fmt.Errorf("invalid action '%s': must be 'breathe' or 'lselect'", action)
		}

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.SetLightAlert(ctx, id, action)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			printMessage("✓ Triggered '%s' alert on %d/%d lights", action, successCount, len(lightIDs))
		} else if successCount > 0 {
			printMessage("Alert '%s' triggered on %s", action, args[0])
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

// lightTemperatureCmd sets light color temperature
var lightTemperatureCmd = &cobra.Command{
	Use:   "temperature <light-name-or-query> <mirek>",
	Short: "Set color temperature (153-500 mirek)",
	Long: `Set color temperature for one or more lights using mirek values.

Mirek values (153-500):
  153  - Coolest (6500K daylight, blue-white)
  250  - Neutral (4000K)
  370  - Warm (2700K soft white)
  500  - Warmest (2000K candlelight, orange)

Use --transition to smoothly fade to the target temperature.

Supports queries using @"..." syntax:
  hue lights temperature @"office" 250
  hue lights temperature @"bedroom" 400 --transition 3000`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		mirek, err := strconv.Atoi(args[1])
		if err != nil {
			return fmt.Errorf("invalid mirek value: %w", err)
		}

		if mirek < 153 || mirek > 500 {
			return fmt.Errorf("mirek must be between 153 (cool) and 500 (warm)")
		}

		ctx := context.Background()

		// Resolve light name/query to IDs
		lightIDs, err := resolveLightIDs(ctx, args[0])
		if err != nil {
			return err
		}

		// Apply to all matched lights concurrently
		semaphore := make(chan struct{}, MaxConcurrentRequests)
		var wg sync.WaitGroup
		var mu sync.Mutex
		var errors []error
		successCount := 0

		for _, lightID := range lightIDs {
			wg.Add(1)

			go func(id string) {
				defer wg.Done()

				semaphore <- struct{}{}
				defer func() { <-semaphore }()

				err := hueClient.SetLightColorTemperatureWithTransition(ctx, id, mirek, transitionMs)

				mu.Lock()
				if err != nil {
					errors = append(errors, err)
				} else {
					successCount++
				}
				mu.Unlock()
			}(lightID)
		}

		wg.Wait()

		// Report results
		if len(lightIDs) > 1 {
			if transitionMs > 0 {
				printMessage("✓ Fading temperature to %d mirek over %dms on %d/%d lights", mirek, transitionMs, successCount, len(lightIDs))
			} else {
				printMessage("✓ Set temperature to %d mirek on %d/%d lights", mirek, successCount, len(lightIDs))
			}
		} else if successCount > 0 {
			if transitionMs > 0 {
				printMessage("Light %s fading to %d mirek over %dms", args[0], mirek, transitionMs)
			} else {
				printMessage("Light %s temperature set to %d mirek", args[0], mirek)
			}
		}

		if len(errors) > 0 {
			return fmt.Errorf("%d lights failed", len(errors))
		}

		return nil
	},
}

func init() {
	// Add flags
	lightColorCmd.Flags().BoolVar(&turnOnWithColor, "turn-on", false, "Turn on the light when setting color (atomic operation)")
	lightColorCmd.Flags().IntVar(&transitionMs, "transition", 0, "Transition time in milliseconds (e.g., 2000 for 2 seconds)")
	lightBrightnessCmd.Flags().BoolVar(&turnOnWithBrightness, "turn-on", false, "Turn on the light when setting brightness (atomic operation)")
	lightBrightnessCmd.Flags().IntVar(&transitionMs, "transition", 0, "Transition time in milliseconds (e.g., 2000 for 2 seconds)")
	lightTemperatureCmd.Flags().IntVar(&transitionMs, "transition", 0, "Transition time in milliseconds (e.g., 2000 for 2 seconds)")

	// Add subcommands
	lightsCmd.AddCommand(listLightsCmd)
	lightsCmd.AddCommand(lightOnCmd)
	lightsCmd.AddCommand(lightOffCmd)
	lightsCmd.AddCommand(lightColorCmd)
	lightsCmd.AddCommand(lightBrightnessCmd)
	lightsCmd.AddCommand(lightStateCmd)
	lightsCmd.AddCommand(lightEffectsListCmd)
	lightsCmd.AddCommand(lightEffectCmd)
	lightsCmd.AddCommand(lightAlertCmd)
	lightsCmd.AddCommand(lightTemperatureCmd)

	// Add to root
	rootCmd.AddCommand(lightsCmd)
}