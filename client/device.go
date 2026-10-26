package client

import (
	"context"
	"fmt"
)

// Device represents a device resource
type Device struct {
	ID           string               `json:"id"`
	IDV1         string               `json:"id_v1"`
	Type         string               `json:"type"`
	Services     []ResourceIdentifier `json:"services"`
	Metadata     Metadata             `json:"metadata"`
	ProductData  ProductData          `json:"product_data"`
	PowerState   *PowerState          `json:"device_power,omitempty"`
}

// ProductData contains product information
type ProductData struct {
	ModelID         string `json:"model_id"`
	ManufacturerName string `json:"manufacturer_name"`
	ProductName     string `json:"product_name"`
	ProductArchetype string `json:"product_archetype"`
	Certified       bool   `json:"certified"`
	SoftwareVersion string `json:"software_version"`
}

// PowerState represents device power information
type PowerState struct {
	PowerState       string  `json:"power_state"`
	BatteryState     string  `json:"battery_state,omitempty"`
	BatteryLevel     float64 `json:"battery_level,omitempty"`
}

// GetDevices returns all devices
func (c *Client) GetDevices(ctx context.Context) ([]Device, error) {
	var response struct {
		Errors []Error  `json:"errors"`
		Data   []Device `json:"data"`
	}
	
	err := c.getJSON(ctx, "/resource/device", &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	return response.Data, nil
}

// GetDevice returns a specific device
func (c *Client) GetDevice(ctx context.Context, id string) (*Device, error) {
	var response struct {
		Errors []Error  `json:"errors"`
		Data   []Device `json:"data"`
	}
	
	err := c.getJSON(ctx, fmt.Sprintf("/resource/device/%s", id), &response)
	if err != nil {
		return nil, err
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("device not found")
	}
	
	return &response.Data[0], nil
}

// IdentifyDevice makes a device identify itself (usually by blinking)
func (c *Client) IdentifyDevice(ctx context.Context, id string) error {
	update := map[string]interface{}{
		"identify": map[string]interface{}{
			"action": "identify",
		},
	}
	return c.putOK(ctx, fmt.Sprintf("/resource/device/%s", id), update)
}