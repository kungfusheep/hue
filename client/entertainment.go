package client

import (
	"context"
	"encoding/binary"
	"fmt"
	"net"
	"sync"
	"time"
)

// Entertainment represents an entertainment configuration
type Entertainment struct {
	ID               string                    `json:"id"`
	IDV1             string                    `json:"id_v1"`
	Type             string                    `json:"type"`
	Metadata         Metadata                  `json:"metadata"`
	ConfigurationType string                   `json:"configuration_type"`
	Status           string                    `json:"status"`
	ActiveStreamer   *ResourceIdentifier       `json:"active_streamer,omitempty"`
	StreamProxy      StreamProxy               `json:"stream_proxy"`
	Channels         []EntertainmentChannel    `json:"channels"`
	Locations        *EntertainmentLocations   `json:"locations,omitempty"`
	LightServices    []ResourceIdentifier      `json:"light_services"`
}

// StreamProxy contains streaming proxy information
type StreamProxy struct {
	Mode string `json:"mode"`
	Node ResourceIdentifier `json:"node"`
}

// EntertainmentChannel represents a channel configuration
type EntertainmentChannel struct {
	ChannelID     int                    `json:"channel_id"`
	Position      EntertainmentPosition  `json:"position"`
	Members       []ChannelMember        `json:"members"`
}

// ChannelMember represents a light in an entertainment channel
type ChannelMember struct {
	Service ResourceIdentifier `json:"service"`
	Index   int                `json:"index"`
}

// EntertainmentPosition represents a 3D position
type EntertainmentPosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

// EntertainmentLocations contains location bounds
type EntertainmentLocations struct {
	ServiceLocations []ServiceLocation `json:"service_locations"`
}

// ServiceLocation represents a service's physical location
type ServiceLocation struct {
	Service   ResourceIdentifier    `json:"service"`
	Position  EntertainmentPosition `json:"position"`
	Positions []EntertainmentPosition `json:"positions,omitempty"`
}

// GetEntertainmentConfigurations returns all entertainment configurations
func (c *Client) GetEntertainmentConfigurations(ctx context.Context) ([]Entertainment, error) {
	var response struct {
		Errors []Error         `json:"errors"`
		Data   []Entertainment `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/entertainment_configuration", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	return response.Data, nil
}

// GetEntertainmentConfiguration returns a specific entertainment configuration
func (c *Client) GetEntertainmentConfiguration(ctx context.Context, id string) (*Entertainment, error) {
	var response struct {
		Errors []Error         `json:"errors"`
		Data   []Entertainment `json:"data"`
	}
	
	err := c.getJSON(ctx, fmt.Sprintf("/resource/entertainment_configuration/%s", id), &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("entertainment configuration not found")
	}
	
	return &response.Data[0], nil
}

// StartEntertainment starts entertainment mode
func (c *Client) StartEntertainment(ctx context.Context, id string) error {
	update := map[string]interface{}{
		"action": "start",
	}
	return c.putOK(ctx, fmt.Sprintf("/resource/entertainment_configuration/%s", id), update)
}

// StopEntertainment stops entertainment mode
func (c *Client) StopEntertainment(ctx context.Context, id string) error {
	update := map[string]interface{}{
		"action": "stop",
	}
	return c.putOK(ctx, fmt.Sprintf("/resource/entertainment_configuration/%s", id), update)
}

// EntertainmentStreamer handles real-time color streaming via UDP
type EntertainmentStreamer struct {
	client        *Client
	conn          *net.UDPConn
	configID      string
	config        *Entertainment
	running       bool
	mu            sync.RWMutex
	updateRate    time.Duration
	stopChan      chan struct{}
	sequence      uint8
}

// EntertainmentUpdate represents a color update for streaming
type EntertainmentUpdate struct {
	LightID string
	Red     uint16
	Green   uint16
	Blue    uint16
}

// NewEntertainmentStreamer creates a new entertainment streamer
func NewEntertainmentStreamer(client *Client, configID string) (*EntertainmentStreamer, error) {
	return &EntertainmentStreamer{
		client:     client,
		configID:   configID,
		updateRate: 50 * time.Millisecond, // 20fps default
		stopChan:   make(chan struct{}),
		sequence:   0,
	}, nil
}

// Start begins the entertainment streaming session
func (e *EntertainmentStreamer) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("streamer already running")
	}

	// Start entertainment mode on the bridge
	err := e.client.StartEntertainment(ctx, e.configID)
	if err != nil {
		return fmt.Errorf("failed to start entertainment mode: %w", err)
	}

	// Get entertainment configuration
	config, err := e.client.GetEntertainmentConfiguration(ctx, e.configID)
	if err != nil {
		return fmt.Errorf("failed to get entertainment config: %w", err)
	}
	e.config = config

	// Connect UDP socket
	bridgeAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:2100", e.client.bridgeIP))
	if err != nil {
		return fmt.Errorf("failed to resolve bridge address: %w", err)
	}

	e.conn, err = net.DialUDP("udp", nil, bridgeAddr)
	if err != nil {
		return fmt.Errorf("failed to connect UDP socket: %w", err)
	}

	e.running = true
	
	// Start the streaming loop
	go e.streamingLoop()

	return nil
}

// Stop ends the entertainment streaming session
func (e *EntertainmentStreamer) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return nil
	}

	// Signal stop
	close(e.stopChan)
	e.running = false

	// Close UDP connection
	if e.conn != nil {
		e.conn.Close()
	}

	// Stop entertainment mode on bridge
	return e.client.StopEntertainment(ctx, e.configID)
}

// SetUpdateRate sets the streaming update rate
func (e *EntertainmentStreamer) SetUpdateRate(rate time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.updateRate = rate
}

// SendColors sends color updates to the entertainment lights
func (e *EntertainmentStreamer) SendColors(updates []EntertainmentUpdate) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.running {
		return fmt.Errorf("streamer not running")
	}

	return e.sendUDPPacket(updates)
}

// GetLights returns the lights in the entertainment configuration
func (e *EntertainmentStreamer) GetLights() []ResourceIdentifier {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	if e.config == nil {
		return nil
	}
	
	return e.config.LightServices
}

// streamingLoop handles the main streaming loop
func (e *EntertainmentStreamer) streamingLoop() {
	ticker := time.NewTicker(e.updateRate)
	defer ticker.Stop()

	for {
		select {
		case <-e.stopChan:
			return
		case <-ticker.C:
			// Send keep-alive packet
			e.sendUDPPacket([]EntertainmentUpdate{})
		}
	}
}

// sendUDPPacket sends a UDP packet with color data
func (e *EntertainmentStreamer) sendUDPPacket(updates []EntertainmentUpdate) error {
	if e.config == nil {
		return fmt.Errorf("no entertainment configuration loaded")
	}

	// Build entertainment protocol packet
	packet := make([]byte, 0, 1024)
	
	// Header: "HueStream" (9 bytes)
	packet = append(packet, []byte("HueStream")...)
	
	// API version (2 bytes) - version 2.0
	packet = append(packet, 0x02, 0x00)
	
	// Sequence number (1 byte)
	e.sequence++
	packet = append(packet, e.sequence)
	
	// Reserved (2 bytes)
	packet = append(packet, 0x00, 0x00)
	
	// Color mode (1 byte) - RGB
	packet = append(packet, 0x01)
	
	// Reserved (1 byte)
	packet = append(packet, 0x00)
	
	// Create color data map
	colorData := make(map[string]EntertainmentUpdate)
	for _, update := range updates {
		colorData[update.LightID] = update
	}
	
	// Add color data for each channel
	for _, channel := range e.config.Channels {
		for _, member := range channel.Members {
			lightID := member.Service.RID
			
			update, exists := colorData[lightID]
			if !exists {
				// Default to off
				update = EntertainmentUpdate{
					LightID: lightID,
					Red:     0,
					Green:   0,
					Blue:    0,
				}
			}
			
			// Channel ID (2 bytes)
			channelBytes := make([]byte, 2)
			binary.LittleEndian.PutUint16(channelBytes, uint16(channel.ChannelID))
			packet = append(packet, channelBytes...)
			
			// RGB values (6 bytes total - 2 bytes each)
			redBytes := make([]byte, 2)
			greenBytes := make([]byte, 2)
			blueBytes := make([]byte, 2)
			
			binary.LittleEndian.PutUint16(redBytes, update.Red)
			binary.LittleEndian.PutUint16(greenBytes, update.Green)
			binary.LittleEndian.PutUint16(blueBytes, update.Blue)
			
			packet = append(packet, redBytes...)
			packet = append(packet, greenBytes...)
			packet = append(packet, blueBytes...)
		}
	}
	
	// Send packet
	_, err := e.conn.Write(packet)
	return err
}

// Helper functions for color conversion

// RGBToUint16 converts 0-255 RGB values to 0-65535 range
func RGBToUint16(r, g, b uint8) (uint16, uint16, uint16) {
	return uint16(r) * 257, uint16(g) * 257, uint16(b) * 257
}

// FloatRGBToUint16 converts 0.0-1.0 RGB values to 0-65535 range
func FloatRGBToUint16(r, g, b float64) (uint16, uint16, uint16) {
	return uint16(r * 65535), uint16(g * 65535), uint16(b * 65535)
}