package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// Client represents a Philips Hue v2 API client
type Client struct {
	bridgeIP   string
	username   string
	httpClient *http.Client
	baseURL    string
}

// NewClient creates a new Hue v2 API client
func NewClient(bridgeIP, username string, httpClient *http.Client) *Client {
	return &Client{
		bridgeIP:   bridgeIP,
		username:   username,
		httpClient: httpClient,
		baseURL:    fmt.Sprintf("https://%s/clip/v2", bridgeIP),
	}
}

// TestConnection verifies the connection to the Hue bridge
func (c *Client) TestConnection(ctx context.Context) error {
	// Try to get the bridge configuration
	_, err := c.get(ctx, "/resource/bridge")
	return err
}

// GetLights returns all lights
func (c *Client) GetLights(ctx context.Context) ([]Light, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Light `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/light", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	return response.Data, nil
}

// GetLight returns a specific light
func (c *Client) GetLight(ctx context.Context, id string) (*Light, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Light `json:"data"`
	}
	
	err := c.getJSON(ctx, fmt.Sprintf("/resource/light/%s", id), &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("light not found")
	}
	
	return &response.Data[0], nil
}

// UpdateLight updates a light's state
func (c *Client) UpdateLight(ctx context.Context, id string, update LightUpdate) error {
	_, err := c.put(ctx, fmt.Sprintf("/resource/light/%s", id), update)
	return err
}

// GetGroups returns all groups/rooms
func (c *Client) GetGroups(ctx context.Context) ([]Group, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Group `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/grouped_light", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	return response.Data, nil
}

// GetGroup returns a specific group
func (c *Client) GetGroup(ctx context.Context, id string) (*Group, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Group `json:"data"`
	}
	
	err := c.getJSON(ctx, fmt.Sprintf("/resource/grouped_light/%s", id), &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("group not found")
	}
	
	return &response.Data[0], nil
}

// UpdateGroup updates a group's state
func (c *Client) UpdateGroup(ctx context.Context, id string, update GroupUpdate) error {
	_, err := c.put(ctx, fmt.Sprintf("/resource/grouped_light/%s", id), update)
	return err
}

// GetScenes returns all scenes
func (c *Client) GetScenes(ctx context.Context) ([]Scene, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Scene `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/scene", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	return response.Data, nil
}

// ActivateScene activates a scene
func (c *Client) ActivateScene(ctx context.Context, id string) error {
	update := map[string]interface{}{
		"recall": map[string]interface{}{
			"action": "active",
		},
	}
	_, err := c.put(ctx, fmt.Sprintf("/resource/scene/%s", id), update)
	return err
}

// ActivateSceneDynamic activates a scene with dynamic/animated palette
func (c *Client) ActivateSceneDynamic(ctx context.Context, id string) error {
	update := map[string]interface{}{
		"recall": map[string]interface{}{
			"action": "dynamic_palette",
		},
	}
	_, err := c.put(ctx, fmt.Sprintf("/resource/scene/%s", id), update)
	return err
}

// UpdateSceneSpeed updates the speed of a scene (must be done before activation)
func (c *Client) UpdateSceneSpeed(ctx context.Context, id string, speed float64) error {
	update := map[string]interface{}{
		"speed": speed,
	}
	_, err := c.put(ctx, fmt.Sprintf("/resource/scene/%s", id), update)
	return err
}

// CreateScene creates a new scene
func (c *Client) CreateScene(ctx context.Context, scene SceneCreate) (*Scene, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []struct {
			ID string `json:"rid"`
		} `json:"data"`
	}
	
	body, err := c.post(ctx, "/resource/scene", scene)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no scene ID returned")
	}
	
	// Get the created scene
	return c.GetScene(ctx, response.Data[0].ID)
}

// GetScene returns a specific scene
func (c *Client) GetScene(ctx context.Context, id string) (*Scene, error) {
	var response struct {
		Errors []Error `json:"errors"`
		Data   []Scene `json:"data"`
	}
	
	err := c.getJSON(ctx, fmt.Sprintf("/resource/scene/%s", id), &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("scene not found")
	}
	
	return &response.Data[0], nil
}

// GetBridge returns bridge information
func (c *Client) GetBridge(ctx context.Context) (*Bridge, error) {
	var response struct {
		Errors []Error  `json:"errors"`
		Data   []Bridge `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/bridge", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("bridge not found")
	}
	
	return &response.Data[0], nil
}

// HTTP helper methods

func (c *Client) get(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "GET", path, nil)
}

func (c *Client) getJSON(ctx context.Context, path string, result interface{}) error {
	body, err := c.get(ctx, path)
	if err != nil {
		return err
	}
	return json.Unmarshal(body, result)
}

func (c *Client) put(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.request(ctx, "PUT", path, data)
}

func (c *Client) post(ctx context.Context, path string, data interface{}) ([]byte, error) {
	return c.request(ctx, "POST", path, data)
}

func (c *Client) delete(ctx context.Context, path string) ([]byte, error) {
	return c.request(ctx, "DELETE", path, nil)
}

func (c *Client) request(ctx context.Context, method, path string, data interface{}) ([]byte, error) {
	url := c.baseURL + path
	
	var body io.Reader
	if data != nil {
		jsonData, err := json.Marshal(data)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewReader(jsonData)
	}
	
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}
	
	req.Header.Set("hue-application-key", c.username)
	if data != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	
	return respBody, nil
}

// Helper methods for common operations

// TurnOnLight turns a light on
func (c *Client) TurnOnLight(ctx context.Context, id string) error {
	return c.UpdateLight(ctx, id, LightUpdate{
		On: &OnState{On: true},
	})
}

// TurnOffLight turns a light off
func (c *Client) TurnOffLight(ctx context.Context, id string) error {
	return c.UpdateLight(ctx, id, LightUpdate{
		On: &OnState{On: false},
	})
}

// SetLightBrightness sets a light's brightness (0-100)
func (c *Client) SetLightBrightness(ctx context.Context, id string, brightness float64) error {
	return c.UpdateLight(ctx, id, LightUpdate{
		Dimming: &Dimming{Brightness: brightness},
	})
}

// SetLightColor sets a light's color from hex string
func (c *Client) SetLightColor(ctx context.Context, id string, hexColor string) error {
	x, y := hexToXY(hexColor)
	return c.UpdateLight(ctx, id, LightUpdate{
		Color: &Color{XY: XY{X: x, Y: y}},
	})
}

// SetLightColorAndTurnOn sets a light's color and turns it on in a single atomic operation
func (c *Client) SetLightColorAndTurnOn(ctx context.Context, id string, hexColor string) error {
	x, y := hexToXY(hexColor)
	return c.UpdateLight(ctx, id, LightUpdate{
		On:    &OnState{On: true},
		Color: &Color{XY: XY{X: x, Y: y}},
	})
}

// SetLightEffect sets a light's effect
func (c *Client) SetLightEffect(ctx context.Context, id string, effect string, duration int) error {
	update := LightUpdate{
		Effects: &Effects{Effect: effect},
	}
	
	if duration > 0 {
		update.Dynamics = &Dynamics{Duration: duration * 1000} // Convert to milliseconds
	}
	
	return c.UpdateLight(ctx, id, update)
}

// TurnOnGroup turns a group on
func (c *Client) TurnOnGroup(ctx context.Context, id string) error {
	return c.UpdateGroup(ctx, id, GroupUpdate{
		On: &OnState{On: true},
	})
}

// TurnOffGroup turns a group off
func (c *Client) TurnOffGroup(ctx context.Context, id string) error {
	return c.UpdateGroup(ctx, id, GroupUpdate{
		On: &OnState{On: false},
	})
}

// SetGroupBrightness sets a group's brightness (0-100)
func (c *Client) SetGroupBrightness(ctx context.Context, id string, brightness float64) error {
	return c.UpdateGroup(ctx, id, GroupUpdate{
		Dimming: &Dimming{Brightness: brightness},
	})
}

// SetGroupColor sets a group's color from hex string
func (c *Client) SetGroupColor(ctx context.Context, id string, hexColor string) error {
	x, y := hexToXY(hexColor)
	return c.UpdateGroup(ctx, id, GroupUpdate{
		Color: &Color{XY: XY{X: x, Y: y}},
	})
}

// SetGroupColorAndTurnOn sets a group's color and turns it on in a single atomic operation
func (c *Client) SetGroupColorAndTurnOn(ctx context.Context, id string, hexColor string) error {
	x, y := hexToXY(hexColor)
	return c.UpdateGroup(ctx, id, GroupUpdate{
		On:    &OnState{On: true},
		Color: &Color{XY: XY{X: x, Y: y}},
	})
}

// SetGroupEffect sets a group's effect
func (c *Client) SetGroupEffect(ctx context.Context, id string, effect string, duration int) error {
	update := GroupUpdate{
		Effects: &Effects{Effect: effect},
	}
	
	if duration > 0 {
		update.Dynamics = &Dynamics{Duration: duration * 1000} // Convert to milliseconds
	}
	
	return c.UpdateGroup(ctx, id, update)
}

// IdentifyLight makes a light blink for identification
func (c *Client) IdentifyLight(ctx context.Context, id string) error {
	return c.UpdateLight(ctx, id, LightUpdate{
		Alert: &Alert{Action: "breathe"},
	})
}

// GetAllSupportedEffects returns all effects supported by any light in the system
func (c *Client) GetAllSupportedEffects(ctx context.Context) ([]string, error) {
	lights, err := c.GetLights(ctx)
	if err != nil {
		return nil, err
	}
	
	effectsMap := make(map[string]bool)
	
	for _, light := range lights {
		if light.Effects != nil {
			for _, effect := range light.Effects.EffectValues {
				effectsMap[effect] = true
			}
		}
	}
	
	var effects []string
	for effect := range effectsMap {
		effects = append(effects, effect)
	}
	
	return effects, nil
}

// Color conversion helpers

func hexToXY(hex string) (float64, float64) {
	// Remove # if present
	hex = strings.TrimPrefix(hex, "#")
	
	// Parse hex values
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	
	// Convert to XY using simplified algorithm
	// This is a basic conversion - a full implementation would use the light's color gamut
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0
	
	// Apply gamma correction
	if rf > 0.04045 {
		rf = pow((rf+0.055)/1.055, 2.4)
	} else {
		rf = rf / 12.92
	}
	
	if gf > 0.04045 {
		gf = pow((gf+0.055)/1.055, 2.4)
	} else {
		gf = gf / 12.92
	}
	
	if bf > 0.04045 {
		bf = pow((bf+0.055)/1.055, 2.4)
	} else {
		bf = bf / 12.92
	}
	
	// Convert to XYZ using sRGB color space matrix
	X := rf*0.4124564 + gf*0.3575761 + bf*0.1804375
	Y := rf*0.2126729 + gf*0.7151522 + bf*0.0721750
	Z := rf*0.0193339 + gf*0.1191920 + bf*0.9503041
	
	// Convert to xy
	sum := X + Y + Z
	if sum == 0 {
		return 0.3127, 0.3290 // Default white
	}
	
	x := X / sum
	y := Y / sum
	
	return x, y
}

func pow(base, exp float64) float64 {
	// Simple power function
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	return result
}