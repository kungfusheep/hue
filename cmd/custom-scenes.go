package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/kungfusheep/hue/mcp"
)

// customScenesCmd represents the custom-scenes command group
var customScenesCmd = &cobra.Command{
	Use:   "custom-scenes",
	Short: "Manage cached lighting scenes",
	Long:  `Commands for managing cached lighting scenes for instant recall.`,
}

// listScenesCmd lists all cached scenes
var listScenesCmd = &cobra.Command{
	Use:   "list",
	Short: "List all cached scenes",
	RunE: func(cmd *cobra.Command, args []string) error {
		scenes := mcp.GetSceneCache().ListScenes()
		
		if jsonOutput {
			printJSON(scenes)
			return nil
		}
		
		if len(scenes) == 0 {
			fmt.Println("No cached scenes available")
			return nil
		}
		
		// Sort by usage count (most used first)
		for i := 0; i < len(scenes); i++ {
			for j := i + 1; j < len(scenes); j++ {
				if scenes[j].UsageCount > scenes[i].UsageCount {
					scenes[i], scenes[j] = scenes[j], scenes[i]
				}
			}
		}
		
		fmt.Printf("Cached scenes (%d):\n\n", len(scenes))
		for _, scene := range scenes {
			fmt.Printf("📦 %s\n", scene.Name)
			if scene.Description != "" {
				fmt.Printf("   Description: %s\n", scene.Description)
			}
			fmt.Printf("   Commands: %d | Delay: %dms | Used: %d times\n",
				len(scene.Commands), scene.DelayMs, scene.UsageCount)
			fmt.Printf("   Created: %s\n\n", scene.CreatedAt.Format("2006-01-02 15:04:05"))
		}
		
		return nil
	},
}

// recallSceneCmd recalls a cached scene
var recallSceneCmd = &cobra.Command{
	Use:   "recall <scene-name>",
	Short: "Recall a cached scene",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sceneName := args[0]
		
		scene, err := mcp.GetSceneCache().GetScene(sceneName)
		if err != nil {
			return fmt.Errorf("failed to get scene: %w", err)
		}
		
		// Generate batch ID for tracking
		batchID := fmt.Sprintf("cli_recall_%s_%d", scene.Name, scene.CreatedAt.Unix())
		
		// Execute the scene asynchronously
		ctx := cmd.Context()
		go mcp.ExecuteBatchAsync(ctx, hueClient, scene.Commands, scene.DelayMs, batchID)
		
		printMessage("Recalling atmosphere: %s...", scene.Name)
		if scene.Description != "" && !quiet {
			fmt.Printf("Description: %s\n", scene.Description)
		}
		printMessage("Commands: %d | Delay: %dms | Usage count: %d",
			len(scene.Commands), scene.DelayMs, scene.UsageCount)
		
		return nil
	},
}

// clearSceneCmd removes a cached scene
var clearSceneCmd = &cobra.Command{
	Use:   "clear <scene-name>",
	Short: "Remove a cached scene",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sceneName := args[0]
		
		err := mcp.GetSceneCache().DeleteScene(sceneName)
		if err != nil {
			return fmt.Errorf("failed to clear scene: %w", err)
		}
		
		printMessage("Scene '%s' has been cleared from cache", sceneName)
		return nil
	},
}

// exportSceneCmd exports a scene as JSON
var exportSceneCmd = &cobra.Command{
	Use:   "export <scene-name>",
	Short: "Export a scene as JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		sceneName := args[0]
		
		scene, err := mcp.GetSceneCache().GetScene(sceneName)
		if err != nil {
			return fmt.Errorf("failed to get scene: %w", err)
		}
		
		// Always output JSON for export
		jsonData, err := json.MarshalIndent(scene, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to serialize scene: %w", err)
		}
		
		fmt.Println(string(jsonData))
		return nil
	},
}

func init() {
	// Add subcommands
	customScenesCmd.AddCommand(listScenesCmd)
	customScenesCmd.AddCommand(recallSceneCmd)
	customScenesCmd.AddCommand(clearSceneCmd)
	customScenesCmd.AddCommand(exportSceneCmd)

	// Add to root
	rootCmd.AddCommand(customScenesCmd)
}