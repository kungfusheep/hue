package cmd

import (
	"context"

	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/tui"
	"github.com/spf13/cobra"
)

// Ensure cliActionHandler implements EventProvider
var _ tui.EventProvider = (*cliActionHandler)(nil)

// tuiCmd launches the interactive TUI
var tuiCmd = &cobra.Command{
	Use:     "tui",
	Aliases: []string{"ui", "control"},
	Short:   "Launch interactive TUI for controlling lights",
	Long: `Launch an interactive terminal UI for browsing, filtering, and controlling lights.

Features:
  - Live query filtering with full query syntax support
  - Navigate lights with vim-style keys (j/k)
  - Toggle lights on/off
  - See real-time state (on/off, brightness, color)

Keybindings:
  j/k       Navigate up/down
  /         Focus query input
  space     Select/deselect light
  o         Toggle on/off
  q         Quit`,
	RunE: func(cmd *cobra.Command, args []string) error {
		handler := &cliActionHandler{client: hueClient}
		return tui.Run(handler)
	},
}

// cliActionHandler wraps the Hue client to implement tui.ActionHandler.
// All actions go through the existing client API - no new functionality here.
type cliActionHandler struct {
	client *client.Client
}

func (h *cliActionHandler) FetchLights() ([]client.Light, error) {
	return h.client.GetLights(context.Background())
}

func (h *cliActionHandler) FetchRooms() ([]client.Room, error) {
	return h.client.GetRooms(context.Background())
}

func (h *cliActionHandler) FetchScenes() ([]client.Scene, error) {
	return h.client.GetScenes(context.Background())
}

func (h *cliActionHandler) ToggleLights(ids []string) error {
	ctx := context.Background()

	for _, id := range ids {
		light, err := h.client.GetLight(ctx, id)
		if err != nil {
			return err
		}

		if light.On.On {
			if err := h.client.TurnOffLight(ctx, id); err != nil {
				return err
			}
		} else {
			if err := h.client.TurnOnLight(ctx, id); err != nil {
				return err
			}
		}
	}
	return nil
}

func (h *cliActionHandler) SetBrightness(ids []string, brightness float64) error {
	ctx := context.Background()
	// brightness is already in 0-100 range from TUI

	for _, id := range ids {
		if err := h.client.SetLightBrightness(ctx, id, brightness); err != nil {
			return err
		}
	}
	return nil
}

func (h *cliActionHandler) SetColor(ids []string, color string) error {
	ctx := context.Background()

	for _, id := range ids {
		if err := h.client.SetLightColor(ctx, id, color); err != nil {
			return err
		}
	}
	return nil
}

func (h *cliActionHandler) SetEffect(ids []string, effect string) error {
	ctx := context.Background()

	for _, id := range ids {
		if err := h.client.SetLightEffect(ctx, id, effect, 0); err != nil {
			return err
		}
	}
	return nil
}

func (h *cliActionHandler) ActivateScene(id string) error {
	return h.client.ActivateScene(context.Background(), id)
}

// SubscribeEvents implements tui.EventProvider for real-time light updates.
func (h *cliActionHandler) SubscribeEvents() (events <-chan tui.LightUpdateEvent, cancel func(), err error) {
	ctx, cancelCtx := context.WithCancel(context.Background())

	stream, err := h.client.StreamEvents(ctx)
	if err != nil {
		cancelCtx()
		return nil, nil, err
	}

	// Create a channel to forward converted events
	eventChan := make(chan tui.LightUpdateEvent, 100)

	// Start a goroutine to convert client events to TUI events
	go func() {
		defer close(eventChan)
		for event := range stream.Events() {
			for _, data := range event.Data {
				// Only process light updates
				if data.Type != "light" {
					continue
				}

				update := tui.LightUpdateEvent{
					LightID: data.ID,
				}

				// Extract on state
				if data.On != nil {
					on := data.On.On
					update.On = &on
				}

				// Extract brightness
				if data.Dimming != nil {
					bri := data.Dimming.Brightness
					update.Brightness = &bri
				}

				// Extract color
				if data.Color != nil {
					update.ColorXY = &struct{ X, Y float64 }{
						X: data.Color.XY.X,
						Y: data.Color.XY.Y,
					}
				}

				// Extract effect
				if data.Effects != nil && data.Effects.Effect != "" {
					effect := data.Effects.Effect
					update.Effect = &effect
				}

				// Send if we have any updates
				if update.On != nil || update.Brightness != nil || update.ColorXY != nil || update.Effect != nil {
					select {
					case eventChan <- update:
					default:
						// Channel full, skip
					}
				}
			}
		}
	}()

	cancel = func() {
		stream.Close()
		cancelCtx()
	}

	return eventChan, cancel, nil
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
