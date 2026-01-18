package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"slices"
	"time"

	"github.com/kungfusheep/hue/client"
	"github.com/kungfusheep/hue/cmd"
	"github.com/kungfusheep/hue/effects"
	mcpserver "github.com/kungfusheep/hue/mcp"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// If no args or first arg is not a known command, run MCP server
	if len(os.Args) == 1 {
		runMCPServer()
		return
	}

	// Check if it's a CLI command
	cliCommands := []string{"lights", "groups", "effects", "scenes", "custom-scenes", "sensors", "batch", "stream", "discover", "tui", "ui", "control"}
	if slices.Contains(cliCommands, os.Args[1]) {
		runCLI()
		return
	}

	// Check for help or --help flag
	if os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h" {
		runCLI()
		return
	}

	// Otherwise run MCP server (for unknown commands)
	runMCPServer()
}

// runCLI initializes and runs the CLI interface
func runCLI() {
	// For help commands, don't initialize the client
	if len(os.Args) > 1 && (os.Args[1] == "help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		cmd.Execute(nil)
		return
	}

	// Initialize Hue client
	hueClient := initHueClient()

	// Initialize scheduler
	mcpserver.InitScheduler(hueClient)

	// Run CLI with the initialized client
	cmd.Execute(hueClient)
}

// initHueClient creates and initializes a Hue client (shared by MCP and CLI)
func initHueClient() *client.Client {
	// Get configuration from environment
	bridgeIP := os.Getenv("HUE_BRIDGE_IP")
	if bridgeIP == "" {
		bridgeIP = "192.168.87.30" // Default bridge IP
	}

	username := os.Getenv("HUE_USERNAME")
	if username == "" {
		log.Fatal("HUE_USERNAME environment variable is required")
	}

	// Create HTTP client that skips certificate verification for self-signed certs
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
	}

	// Initialize Hue client
	hueClient := client.NewClient(bridgeIP, username, httpClient)

	// Test connection
	ctx := context.Background()
	if err := hueClient.TestConnection(ctx); err != nil {
		log.Fatalf("Failed to connect to Hue bridge: %v", err)
	}

	return hueClient
}

// runMCPServer runs the MCP server (original main function)
func runMCPServer() {
	// Initialize Hue client using shared function
	hueClient := initHueClient()

	// Initialize scheduler
	mcpserver.InitScheduler(hueClient)

	// Create MCP server
	srv := server.NewMCPServer(
		"Philips Hue v2 MCP Server",
		"1.0.0",
		server.WithToolCapabilities(true),
		server.WithResourceCapabilities(true, false),
	)

	// Register tools
	registerLightTools(srv, hueClient)
	registerGroupTools(srv, hueClient)
	registerSceneTools(srv, hueClient)
	registerEffectTools(srv, hueClient)
	registerSystemTools(srv, hueClient)
	registerRoomTools(srv, hueClient)
	registerSensorTools(srv, hueClient)
	registerEntertainmentTools(srv, hueClient)
	registerBatchTools(srv, hueClient)
	registerSchedulerTools(srv, hueClient)
	registerEventTools(srv, hueClient)
	registerCRUDTools(srv, hueClient)

	// Start server in stdio mode for Claude Desktop
	log.Println("Starting Hue MCP server...")
	if err := server.ServeStdio(srv); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

// registerLightTools adds individual light control tools
func registerLightTools(srv *server.MCPServer, client *client.Client) {
	// Light on/off
	lightOnTool := mcp.NewTool("light_on",
		mcp.WithDescription("Turn a light on"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
	)
	srv.AddTool(lightOnTool, mcpserver.HandleLightOn(client))

	lightOffTool := mcp.NewTool("light_off",
		mcp.WithDescription("Turn a light off"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
	)
	srv.AddTool(lightOffTool, mcpserver.HandleLightOff(client))

	// Brightness control
	brightnessTool := mcp.NewTool("light_brightness",
		mcp.WithDescription("Set light brightness"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
		mcp.WithNumber("brightness", mcp.Required(), mcp.Description("Brightness percentage (0-100)")),
	)
	srv.AddTool(brightnessTool, mcpserver.HandleLightBrightness(client))

	// Color control
	colorTool := mcp.NewTool("light_color",
		mcp.WithDescription("Set light color"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
		mcp.WithString("color", mcp.Required(), mcp.Description("Color as hex code (e.g., #FF0000) or color name")),
	)
	srv.AddTool(colorTool, mcpserver.HandleLightColor(client))
}

// registerGroupTools adds group control tools
func registerGroupTools(srv *server.MCPServer, client *client.Client) {
	// Group on/off
	groupOnTool := mcp.NewTool("group_on",
		mcp.WithDescription("Turn a group of lights on"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The ID of the group")),
	)
	srv.AddTool(groupOnTool, mcpserver.HandleGroupOn(client))

	groupOffTool := mcp.NewTool("group_off",
		mcp.WithDescription("Turn a group of lights off"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The ID of the group")),
	)
	srv.AddTool(groupOffTool, mcpserver.HandleGroupOff(client))

	// Group brightness
	groupBrightnessTool := mcp.NewTool("group_brightness",
		mcp.WithDescription("Set group brightness"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Brightness percentage (0-100)")),
	)
	srv.AddTool(groupBrightnessTool, mcpserver.HandleGroupBrightness(client))

	// Group color
	groupColorTool := mcp.NewTool("group_color",
		mcp.WithDescription("Set group color"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The ID of the group")),
		mcp.WithString("color", mcp.Required(), mcp.Description("Color as hex code or name")),
	)
	srv.AddTool(groupColorTool, mcpserver.HandleGroupColor(client))
}

// registerSceneTools adds scene management tools
func registerSceneTools(srv *server.MCPServer, client *client.Client) {
	// List scenes
	listScenesTool := mcp.NewTool("list_scenes",
		mcp.WithDescription("List all available scenes"),
	)
	srv.AddTool(listScenesTool, mcpserver.HandleListScenes(client))

	// Activate scene
	activateSceneTool := mcp.NewTool("activate_scene",
		mcp.WithDescription("Activate a scene"),
		mcp.WithString("scene_id", mcp.Required(), mcp.Description("The ID of the scene")),
	)
	srv.AddTool(activateSceneTool, mcpserver.HandleActivateScene(client))

	// Create scene
	createSceneTool := mcp.NewTool("create_scene",
		mcp.WithDescription("Create a new scene from current light states"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the scene")),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group to capture")),
	)
	srv.AddTool(createSceneTool, mcpserver.HandleCreateScene(client))
}

// registerEffectTools adds native effect tools
func registerEffectTools(srv *server.MCPServer, client *client.Client) {
	// Get supported effects dynamically
	ctx := context.Background()
	supportedEffects, err := client.GetAllSupportedEffects(ctx)
	if err != nil {
		log.Printf("Warning: Could not get supported effects, using defaults: %v", err)
		supportedEffects = effects.GetAllEffects()
	}

	// Set effect on light
	lightEffectTool := mcp.NewTool("light_effect",
		mcp.WithDescription("Set a dynamic effect on a light"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
		mcp.WithString("effect", mcp.Required(),
			mcp.Description("Effect to apply"),
			mcp.Enum(supportedEffects...),
		),
		mcp.WithNumber("duration", mcp.Description("Duration in seconds (0 for infinite)")),
	)
	srv.AddTool(lightEffectTool, mcpserver.HandleLightEffect(client))

	// Set effect on group
	groupEffectTool := mcp.NewTool("group_effect",
		mcp.WithDescription("Set a dynamic effect on a group"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("The ID of the group")),
		mcp.WithString("effect", mcp.Required(),
			mcp.Description("Effect to apply"),
			mcp.Enum(supportedEffects...),
		),
		mcp.WithNumber("duration", mcp.Description("Duration in seconds (0 for infinite)")),
	)
	srv.AddTool(groupEffectTool, mcpserver.HandleGroupEffect(client))
}

// registerSystemTools adds system and discovery tools
func registerSystemTools(srv *server.MCPServer, client *client.Client) {
	// List lights
	listLightsTool := mcp.NewTool("list_lights",
		mcp.WithDescription("List all available lights"),
	)
	srv.AddTool(listLightsTool, mcpserver.HandleListLights(client))

	// List groups
	listGroupsTool := mcp.NewTool("list_groups",
		mcp.WithDescription("List all available groups/rooms"),
	)
	srv.AddTool(listGroupsTool, mcpserver.HandleListGroups(client))

	// Get light state
	getLightStateTool := mcp.NewTool("get_light_state",
		mcp.WithDescription("Get current state of a light"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
	)
	srv.AddTool(getLightStateTool, mcpserver.HandleGetLightState(client))

	// Bridge info
	bridgeInfoTool := mcp.NewTool("bridge_info",
		mcp.WithDescription("Get bridge information and capabilities"),
	)
	srv.AddTool(bridgeInfoTool, mcpserver.HandleBridgeInfo(client))

	// Identify light
	identifyLightTool := mcp.NewTool("identify_light",
		mcp.WithDescription("Make a light blink to identify it"),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("The ID of the light")),
	)
	srv.AddTool(identifyLightTool, mcpserver.HandleIdentifyLight(client))
}

// registerRoomTools adds room and zone control tools
func registerRoomTools(srv *server.MCPServer, client *client.Client) {
	// List rooms
	listRoomsTool := mcp.NewTool("list_rooms",
		mcp.WithDescription("List all rooms with their lights"),
	)
	srv.AddTool(listRoomsTool, mcpserver.HandleListRooms(client))

	// List zones
	listZonesTool := mcp.NewTool("list_zones",
		mcp.WithDescription("List all zones"),
	)
	srv.AddTool(listZonesTool, mcpserver.HandleListZones(client))

	// List devices
	listDevicesTool := mcp.NewTool("list_devices",
		mcp.WithDescription("List all devices with their details"),
	)
	srv.AddTool(listDevicesTool, mcpserver.HandleListDevices(client))

	// Get device details
	getDeviceTool := mcp.NewTool("get_device",
		mcp.WithDescription("Get detailed information about a device"),
		mcp.WithString("device_id", mcp.Required(), mcp.Description("The ID of the device")),
	)
	srv.AddTool(getDeviceTool, mcpserver.HandleGetDevice(client))
}

// registerSensorTools adds sensor reading tools
func registerSensorTools(srv *server.MCPServer, client *client.Client) {
	// Motion sensors
	listMotionTool := mcp.NewTool("list_motion_sensors",
		mcp.WithDescription("List all motion sensors and their states"),
	)
	srv.AddTool(listMotionTool, mcpserver.HandleListMotionSensors(client))

	// Temperature sensors
	listTempTool := mcp.NewTool("list_temperature_sensors",
		mcp.WithDescription("List all temperature sensors and their readings"),
	)
	srv.AddTool(listTempTool, mcpserver.HandleListTemperatureSensors(client))

	// Light level sensors
	listLightLevelTool := mcp.NewTool("list_light_level_sensors",
		mcp.WithDescription("List all light level sensors and their readings"),
	)
	srv.AddTool(listLightLevelTool, mcpserver.HandleListLightLevelSensors(client))

	// Buttons
	listButtonsTool := mcp.NewTool("list_buttons",
		mcp.WithDescription("List all buttons (dimmer switches) and their last events"),
	)
	srv.AddTool(listButtonsTool, mcpserver.HandleListButtons(client))
}

// registerEntertainmentTools adds entertainment configuration tools
func registerEntertainmentTools(srv *server.MCPServer, client *client.Client) {
	// List entertainment configurations
	listEntTool := mcp.NewTool("list_entertainment",
		mcp.WithDescription("List all entertainment configurations"),
	)
	srv.AddTool(listEntTool, mcpserver.HandleListEntertainment(client))

	// Start entertainment
	startEntTool := mcp.NewTool("start_entertainment",
		mcp.WithDescription("Start entertainment mode for a configuration"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
	)
	srv.AddTool(startEntTool, mcpserver.HandleStartEntertainment(client))

	// Stop entertainment
	stopEntTool := mcp.NewTool("stop_entertainment",
		mcp.WithDescription("Stop entertainment mode for a configuration"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
	)
	srv.AddTool(stopEntTool, mcpserver.HandleStopEntertainment(client))

	// Start streaming
	startStreamTool := mcp.NewTool("start_streaming",
		mcp.WithDescription("Start UDP streaming for real-time color updates"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
		mcp.WithString("update_rate_ms", mcp.Description("Update rate in milliseconds (default: 50)")),
	)
	srv.AddTool(startStreamTool, mcpserver.HandleStartStreaming(client))

	// Stop streaming
	stopStreamTool := mcp.NewTool("stop_streaming",
		mcp.WithDescription("Stop UDP streaming"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
	)
	srv.AddTool(stopStreamTool, mcpserver.HandleStopStreaming(client))

	// Send colors
	sendColorsTool := mcp.NewTool("send_colors",
		mcp.WithDescription("Send color updates to streaming lights"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
		mcp.WithString("colors", mcp.Required(), mcp.Description("Colors in format: 'lightID1:r,g,b;lightID2:r,g,b' (RGB 0-255)")),
	)
	srv.AddTool(sendColorsTool, mcpserver.HandleSendColors(client))

	// Streaming status
	streamStatusTool := mcp.NewTool("streaming_status",
		mcp.WithDescription("Get status of active streaming sessions"),
	)
	srv.AddTool(streamStatusTool, mcpserver.HandleStreamingStatus(client))

	// Rainbow effect
	rainbowTool := mcp.NewTool("rainbow_effect",
		mcp.WithDescription("Create a rainbow effect on streaming lights"),
		mcp.WithString("config_id", mcp.Required(), mcp.Description("The ID of the entertainment configuration")),
		mcp.WithString("duration", mcp.Description("Duration in seconds (default: 10)")),
	)
	srv.AddTool(rainbowTool, mcpserver.HandleRainbowEffect(client))
}

// registerBatchTools adds batch request capability for efficiency
func registerBatchTools(srv *server.MCPServer, client *client.Client) {
	// Batch commands
	batchTool := mcp.NewTool("batch_commands",
		mcp.WithDescription("Execute multiple lighting commands in sequence with timing control. By default runs asynchronously (returns immediately) so you can continue working while lights change. Perfect for creating simple animations or coordinated lighting changes across multiple lights. Can optionally cache complex scenes for instant recall later!"),
		mcp.WithString("commands", mcp.Required(), mcp.Description("JSON array of commands. Example: [{\"action\":\"light_on\",\"target_id\":\"abc123\"}, {\"action\":\"light_color\",\"target_id\":\"abc123\",\"value\":\"#FF0000\"}, {\"action\":\"light_brightness\",\"target_id\":\"abc123\",\"value\":\"75\"}]")),
		mcp.WithNumber("delay_ms", mcp.Description("Milliseconds to wait between each command - use for timing effects (default: 100)")),
		mcp.WithBoolean("async", mcp.Description("Run in background (true) or wait for completion (false). Default true = non-blocking")),
		mcp.WithString("cache_name", mcp.Description("Optional: Save this sequence as a named custom scene for instant recall later (e.g., 'alien_artifact_discovery')")),
		mcp.WithString("cache_description", mcp.Description("Optional: Description of the custom scene to help remember its purpose")),
	)
	srv.AddTool(batchTool, mcpserver.HandleBatchCommands(client))
}

// registerSchedulerTools adds scheduler and sequence tools
func registerSchedulerTools(srv *server.MCPServer, client *client.Client) {
	// Flash effect
	flashTool := mcp.NewTool("flash_effect",
		mcp.WithDescription("Create a flashing/blinking effect on lights - great for alerts, notifications, or party effects. The light will flash on and off with your chosen color."),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Light or group ID to flash")),
		mcp.WithString("color", mcp.Description("Flash color in hex format, e.g. #FF0000 for red, #00FF00 for green (default: #FFFFFF white)")),
		mcp.WithNumber("flash_count", mcp.Description("How many times to flash (default: 3)")),
		mcp.WithNumber("flash_duration_ms", mcp.Description("How long each flash lasts in milliseconds - shorter = more strobe-like (default: 200)")),
	)
	srv.AddTool(flashTool, mcpserver.HandleFlashEffect(client))

	// Pulse effect
	pulseTool := mcp.NewTool("pulse_effect",
		mcp.WithDescription("Create a smooth breathing/heartbeat effect by fading brightness up and down. Perfect for ambient lighting, meditation spaces, or subtle notifications."),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Light or group ID to pulse")),
		mcp.WithNumber("min_brightness", mcp.Description("How dim to go (0-100%, default: 10)")),
		mcp.WithNumber("max_brightness", mcp.Description("How bright to go (0-100%, default: 100)")),
		mcp.WithNumber("pulse_duration_ms", mcp.Description("Time for one complete pulse cycle in milliseconds - longer = slower breathing (default: 2000)")),
		mcp.WithNumber("pulse_count", mcp.Description("Number of pulse cycles to perform (default: 5)")),
	)
	srv.AddTool(pulseTool, mcpserver.HandlePulseEffect(client))

	// Color loop effect
	colorLoopTool := mcp.NewTool("color_loop",
		mcp.WithDescription("Cycle through multiple colors in a continuous loop. Create rainbow effects, team colors, seasonal themes, or any custom color sequence. Loops until stopped."),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Light or group ID to animate")),
		mcp.WithString("colors", mcp.Description("JSON array of hex colors to cycle through, e.g. [\"#FF0000\",\"#00FF00\",\"#0000FF\"] for RGB. Leave empty for rainbow!")),
		mcp.WithNumber("transition_time_ms", mcp.Description("Smooth transition time between colors in milliseconds (default: 1000)")),
	)
	srv.AddTool(colorLoopTool, mcpserver.HandleColorLoopEffect(client))

	// Strobe effect
	strobeTool := mcp.NewTool("strobe_effect",
		mcp.WithDescription("Create a rapid strobe/disco effect. ⚠️ Warning: Very fast flashing - not suitable for those sensitive to strobing lights. Great for parties or dramatic effects!"),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Light or group ID to strobe")),
		mcp.WithString("color", mcp.Description("Strobe color in hex format (default: #FFFFFF white)")),
		mcp.WithNumber("strobe_rate_ms", mcp.Description("Time between flashes in milliseconds - lower = faster strobe (default: 100, minimum safe: 50)")),
		mcp.WithNumber("duration_ms", mcp.Description("How long to run the strobe effect in milliseconds (default: 5000 = 5 seconds)")),
	)
	srv.AddTool(strobeTool, mcpserver.HandleStrobeEffect(client))

	// Alert effect
	alertTool := mcp.NewTool("alert_effect",
		mcp.WithDescription("Get immediate attention with a pre-programmed alert pattern - rapid flashes followed by return to normal. Perfect for notifications, alarms, or signaling."),
		mcp.WithString("target_id", mcp.Required(), mcp.Description("Light or group ID to alert with")),
		mcp.WithString("alert_color", mcp.Description("Alert flash color in hex format (default: #FF0000 red for urgency)")),
		mcp.WithString("normal_color", mcp.Description("Color to return to after alert (default: #FFFFFF white)")),
	)
	srv.AddTool(alertTool, mcpserver.HandleAlertEffect(client))

	// Stop sequence
	stopSequenceTool := mcp.NewTool("stop_sequence",
		mcp.WithDescription("Stop one or more running light sequences or effects. Use list_sequences first to see active sequence IDs."),
		mcp.WithString("sequence_id", mcp.Description("ID of a single sequence to stop (for backward compatibility)")),
		mcp.WithString("sequence_ids", mcp.Description("JSON array of sequence IDs to stop, e.g. [\"seq1\",\"seq2\",\"seq3\"]")),
	)
	srv.AddTool(stopSequenceTool, mcpserver.HandleStopSequence(client))

	// List sequences
	listSequencesTool := mcp.NewTool("list_sequences",
		mcp.WithDescription("Show all currently running light effects and sequences with their IDs. Useful for managing multiple effects."),
	)
	srv.AddTool(listSequencesTool, mcpserver.HandleListSequences(client))

	// Custom sequence
	customSequenceTool := mcp.NewTool("custom_sequence",
		mcp.WithDescription("Create complex custom lighting sequences with precise timing. Build sunrise simulations, scene transitions, party modes, or any multi-step lighting choreography. Sequences can include color changes, brightness fades, on/off states, and delays."),
		mcp.WithString("sequence", mcp.Required(), mcp.Description("JSON sequence definition. Example: {\"name\":\"Sunrise\",\"loop\":false,\"commands\":[{\"type\":\"light\",\"action\":\"color\",\"target\":\"light_id\",\"params\":{\"color\":\"#FF4500\"},\"delay\":1000},{\"type\":\"light\",\"action\":\"brightness\",\"target\":\"light_id\",\"params\":{\"brightness\":100},\"delay\":2000}]}")),
	)
	srv.AddTool(customSequenceTool, mcpserver.HandleCustomSequence(client))

	// Scene cache tools
	recallSceneTool := mcp.NewTool("recall_custom_scene",
		mcp.WithDescription("Instantly recall a previously cached custom lighting scene. Perfect for quickly setting up complex atmospheres in RPGs or recreating favorite lighting moods."),
		mcp.WithString("scene_name", mcp.Required(), mcp.Description("Name of the cached scene to recall (e.g., 'alien_artifact_discovery')")),
	)
	srv.AddTool(recallSceneTool, mcpserver.HandleRecallScene(client))

	listCachedScenesTool := mcp.NewTool("list_custom_scenes",
		mcp.WithDescription("List all available custom lighting scenes with their descriptions and usage statistics. Helps you remember what atmospheres you've created."),
	)
	srv.AddTool(listCachedScenesTool, mcpserver.HandleListCachedScenes(client))

	clearCachedSceneTool := mcp.NewTool("clear_custom_scene",
		mcp.WithDescription("Remove a custom scene from memory. Use this to clean up scenes you no longer need."),
		mcp.WithString("scene_name", mcp.Required(), mcp.Description("Name of the cached scene to remove")),
	)
	srv.AddTool(clearCachedSceneTool, mcpserver.HandleClearCachedScene(client))

	exportSceneTool := mcp.NewTool("export_custom_scene",
		mcp.WithDescription("Export a custom scene as JSON for sharing or backup. Great for saving your favorite atmospheric setups."),
		mcp.WithString("scene_name", mcp.Required(), mcp.Description("Name of the cached scene to export")),
	)
	srv.AddTool(exportSceneTool, mcpserver.HandleExportScene(client))
}

// registerEventTools adds event streaming tools
func registerEventTools(srv *server.MCPServer, client *client.Client) {
	// Initialize event manager
	mcpserver.InitEventManager(client)

	// Start event stream
	startEventTool := mcp.NewTool("start_event_stream",
		mcp.WithDescription("Start real-time event streaming from Hue bridge"),
		mcp.WithString("filter", mcp.Description("Comma-separated event types to filter (e.g., 'light,motion,button')")),
	)
	srv.AddTool(startEventTool, mcpserver.HandleStartEventStream(client))

	// Stop event stream
	stopEventTool := mcp.NewTool("stop_event_stream",
		mcp.WithDescription("Stop the event stream"),
	)
	srv.AddTool(stopEventTool, mcpserver.HandleStopEventStream(client))

	// Get recent events
	recentEventsTool := mcp.NewTool("get_recent_events",
		mcp.WithDescription("Get recent events from the stream"),
		mcp.WithNumber("limit", mcp.Description("Maximum number of events to return (default: 50)")),
		mcp.WithString("type", mcp.Description("Filter by event type (e.g., 'light', 'motion', 'button')")),
	)
	srv.AddTool(recentEventsTool, mcpserver.HandleGetRecentEvents(client))

	// Get stream status
	streamStatusTool := mcp.NewTool("get_event_stream_status",
		mcp.WithDescription("Get the current status of the event stream"),
	)
	srv.AddTool(streamStatusTool, mcpserver.HandleGetEventStreamStatus(client))
}

// registerCRUDTools adds create, update, delete tools
func registerCRUDTools(srv *server.MCPServer, client *client.Client) {
	// Scene CRUD
	createSceneFromStateTool := mcp.NewTool("create_scene_from_state",
		mcp.WithDescription("Create a new scene capturing current light states"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the scene")),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group/room ID to capture")),
	)
	srv.AddTool(createSceneFromStateTool, mcpserver.HandleCreateSceneFromState(client))

	updateSceneTool := mcp.NewTool("update_scene",
		mcp.WithDescription("Update a scene's metadata"),
		mcp.WithString("scene_id", mcp.Required(), mcp.Description("Scene ID to update")),
		mcp.WithString("name", mcp.Description("New name for the scene")),
		mcp.WithNumber("speed", mcp.Description("Transition speed (0.0-1.0)")),
	)
	srv.AddTool(updateSceneTool, mcpserver.HandleUpdateScene(client))

	deleteSceneTool := mcp.NewTool("delete_scene",
		mcp.WithDescription("Delete a scene"),
		mcp.WithString("scene_id", mcp.Required(), mcp.Description("Scene ID to delete")),
	)
	srv.AddTool(deleteSceneTool, mcpserver.HandleDeleteScene(client))

	// Group management
	addLightToGroupTool := mcp.NewTool("add_light_to_group",
		mcp.WithDescription("Add a light to a group/room"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group ID")),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("Light ID to add")),
	)
	srv.AddTool(addLightToGroupTool, mcpserver.HandleAddLightToGroup(client))

	removeLightFromGroupTool := mcp.NewTool("remove_light_from_group",
		mcp.WithDescription("Remove a light from a group/room"),
		mcp.WithString("group_id", mcp.Required(), mcp.Description("Group ID")),
		mcp.WithString("light_id", mcp.Required(), mcp.Description("Light ID to remove")),
	)
	srv.AddTool(removeLightFromGroupTool, mcpserver.HandleRemoveLightFromGroup(client))

	// Zone CRUD
	createZoneTool := mcp.NewTool("create_zone",
		mcp.WithDescription("Create a new zone with specified lights"),
		mcp.WithString("name", mcp.Required(), mcp.Description("Name for the zone")),
		mcp.WithString("light_ids", mcp.Required(), mcp.Description("Comma-separated light IDs")),
	)
	srv.AddTool(createZoneTool, mcpserver.HandleCreateZone(client))

	updateZoneTool := mcp.NewTool("update_zone",
		mcp.WithDescription("Update a zone"),
		mcp.WithString("zone_id", mcp.Required(), mcp.Description("Zone ID to update")),
		mcp.WithString("name", mcp.Description("New name for the zone")),
		mcp.WithString("light_ids", mcp.Description("Comma-separated light IDs to set")),
	)
	srv.AddTool(updateZoneTool, mcpserver.HandleUpdateZone(client))

	deleteZoneTool := mcp.NewTool("delete_zone",
		mcp.WithDescription("Delete a zone"),
		mcp.WithString("zone_id", mcp.Required(), mcp.Description("Zone ID to delete")),
	)
	srv.AddTool(deleteZoneTool, mcpserver.HandleDeleteZone(client))

	// Room update
	updateRoomTool := mcp.NewTool("update_room",
		mcp.WithDescription("Update a room's name"),
		mcp.WithString("room_id", mcp.Required(), mcp.Description("Room ID to update")),
		mcp.WithString("name", mcp.Required(), mcp.Description("New name for the room")),
	)
	srv.AddTool(updateRoomTool, mcpserver.HandleUpdateRoom(client))
}
