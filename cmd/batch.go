package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/kungfusheep/hue/mcp"
)

var (
	batchDelay       int
	batchAsync       bool
	batchCacheName   string
	batchDescription string
	batchFile        string
)

// batchCmd represents the batch command
var batchCmd = &cobra.Command{
	Use:   "batch",
	Short: "Execute multiple commands in sequence",
	Long: `Execute a batch of lighting commands from JSON.
	
Example JSON format:
[
  {"action": "light_on", "target_id": "abc123"},
  {"action": "light_color", "target_id": "abc123", "value": "#FF0000"},
  {"action": "light_brightness", "target_id": "abc123", "value": "75"}
]

You can provide commands inline or from a file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var commandsJSON string
		
		// Read from file if specified
		if batchFile != "" {
			data, err := os.ReadFile(batchFile)
			if err != nil {
				return fmt.Errorf("failed to read file: %w", err)
			}
			commandsJSON = string(data)
		} else if len(args) > 0 {
			// Use inline JSON
			commandsJSON = args[0]
		} else {
			return fmt.Errorf("provide commands as JSON string or use --file flag")
		}
		
		// Parse commands
		var commands []map[string]interface{}
		if err := json.Unmarshal([]byte(commandsJSON), &commands); err != nil {
			return fmt.Errorf("failed to parse commands JSON: %v", err)
		}
		
		// Save to cache if requested
		if batchCacheName != "" {
			err := mcp.GetSceneCache().SaveScene(batchCacheName, commands, batchDelay, batchDescription)
			if err != nil {
				return fmt.Errorf("failed to cache scene: %v", err)
			}
			printMessage("Scene cached as '%s'", batchCacheName)
		}
		
		// Execute commands
		if batchAsync {
			// Async execution (fire and forget)
			batchID := fmt.Sprintf("cli_batch_%d", time.Now().Unix())
			go mcp.ExecuteBatchAsync(cmd.Context(), hueClient, commands, batchDelay, batchID)
			printMessage("Batch started asynchronously (ID: %s)", batchID)
			printMessage("Commands: %d | Delay: %dms", len(commands), batchDelay)
		} else {
			// Sync execution - execute each command
			printMessage("Executing %d commands...", len(commands))
			results := mcp.ExecuteBatch(cmd.Context(), hueClient, commands, batchDelay)
			
			// Report results
			successful := 0
			for _, result := range results {
				if result.Success {
					successful++
				}
			}
			
			printMessage("Batch completed: %d/%d successful", successful, len(commands))
			
			// Show failures if any
			if successful < len(commands) {
				fmt.Println("\nFailed commands:")
				for i, result := range results {
					if !result.Success {
						fmt.Printf("- Command %d: %v\n", i, result.Error)
					}
				}
			}
		}
		
		return nil
	},
}

func init() {
	batchCmd.Flags().IntVar(&batchDelay, "delay", 100, "Delay between commands in milliseconds")
	batchCmd.Flags().BoolVar(&batchAsync, "async", false, "Run asynchronously (don't wait for completion)")
	batchCmd.Flags().StringVar(&batchCacheName, "cache-name", "", "Save this batch as a cached scene")
	batchCmd.Flags().StringVar(&batchDescription, "cache-desc", "", "Description for cached scene")
	batchCmd.Flags().StringVarP(&batchFile, "file", "f", "", "Read commands from JSON file")

	rootCmd.AddCommand(batchCmd)
}

// MaxConcurrentRequests defines the maximum number of concurrent API requests
// to avoid overwhelming the Hue bridge
const MaxConcurrentRequests = 10
