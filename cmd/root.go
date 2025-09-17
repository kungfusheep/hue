package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/mcp"
)

var (
	// Global flags
	jsonOutput bool
	quiet      bool
	
	// Shared Hue client
	hueClient *client.Client
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "hue",
	Short: "CLI for controlling Philips Hue lights",
	Long: `Hue CLI provides command-line access to all Philips Hue functionality.
	
Control lights, groups, scenes, and effects directly from your terminal.
Perfect for scripting, testing, or quick light adjustments.`,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Skip client init for help commands
		if cmd.Name() == "help" || cmd.Name() == "hue" && cmd.CalledAs() == "help" {
			return
		}
		// Skip for discover command
		if cmd.Name() == "discover" {
			return
		}
		// Skip if no client provided (help was called)
		if hueClient == nil {
			return
		}

		// Initialize client and scheduler for all other commands
		initializeClient()
	},
}

// Execute runs the CLI
func Execute(client *client.Client) {
	// Store client for use in commands
	hueClient = client
	
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initializeClient sets up the Hue client and scheduler
func initializeClient() {
	if hueClient == nil {
		// This should not happen if Execute is called correctly
		fmt.Fprintln(os.Stderr, "Error: Hue client not initialized")
		os.Exit(1)
	}
	
	// Initialize scheduler
	mcp.InitScheduler(hueClient)
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "Suppress non-essential output")
}

// Helper functions for output
func printJSON(data interface{}) {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		printError("failed to marshal JSON: %v", err)
		return
	}
	fmt.Println(string(jsonData))
}

func printMessage(format string, args ...interface{}) {
	if !quiet {
		fmt.Printf(format+"\n", args...)
	}
}

func printError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}