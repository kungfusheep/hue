package client

import (
	"context"
	"encoding/json"
	"fmt"
)

// Scene CRUD operations

// CreateSceneFromCurrentState creates a new scene capturing current light states
func (c *Client) CreateSceneFromCurrentState(ctx context.Context, name string, roomID string) (*Scene, error) {
	// Get the room to find all lights
	rooms, err := c.GetRooms(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get rooms: %w", err)
	}
	
	var targetRoom *Room
	for _, r := range rooms {
		if r.ID == roomID {
			targetRoom = &r
			break
		}
	}
	
	if targetRoom == nil {
		return nil, fmt.Errorf("room %s not found", roomID)
	}
	
	// Get all lights in the room
	var lightIDs []string
	devices, err := c.GetDevices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get devices: %w", err)
	}
	
	// Find all light IDs from devices in this room
	for _, child := range targetRoom.Children {
		if child.RType == "device" {
			// Find this device and its lights
			for _, device := range devices {
				if device.ID == child.RID {
					for _, svc := range device.Services {
						if svc.RType == "light" {
							lightIDs = append(lightIDs, svc.RID)
						}
					}
				}
			}
		}
	}
	
	// Get current state of each light
	var actions []SceneAction
	for _, lightID := range lightIDs {
		light, err := c.GetLight(ctx, lightID)
		if err != nil {
			continue // Skip if we can't get the light
		}
		
		action := SceneAction{
			Target: ResourceIdentifier{
				RID:   lightID,
				RType: "light",
			},
			Action: LightUpdate{
				On: &OnState{
					On: light.On.On,
				},
			},
		}
		
		// Add dimming if light is on
		if light.On.On && light.Dimming.Brightness > 0 {
			action.Action.Dimming = &Dimming{
				Brightness: light.Dimming.Brightness,
			}
		}
		
		// Add color if available
		if light.Color != nil {
			action.Action.Color = &Color{
				XY: light.Color.XY,
			}
		}
		
		// Add color temperature if available
		if light.ColorTemperature != nil && light.ColorTemperature.MirekValid {
			action.Action.ColorTemperature = &ColorTemperature{
				Mirek: light.ColorTemperature.Mirek,
			}
		}
		
		actions = append(actions, action)
	}
	
	// Create the scene
	sceneCreate := SceneCreate{
		Type: "scene",
		Metadata: Metadata{
			Name: name,
		},
		Group: ResourceIdentifier{
			RID:   roomID,
			RType: "room",
		},
		Actions: actions,
		Speed:   0.5, // Default transition speed
	}
	
	return c.CreateScene(ctx, sceneCreate)
}

// DeleteScene deletes a scene
func (c *Client) DeleteScene(ctx context.Context, id string) error {
	_, err := c.delete(ctx, fmt.Sprintf("/resource/scene/%s", id))
	return err
}

// UpdateScene updates a scene's metadata
func (c *Client) UpdateScene(ctx context.Context, id string, update SceneUpdate) error {
	return c.putOK(ctx, fmt.Sprintf("/resource/scene/%s", id), update)
}

// Group CRUD operations

// AddLightToGroup adds a light to a group
func (c *Client) AddLightToGroup(ctx context.Context, groupID, lightID string) error {
	// First get the current group
	groups, err := c.GetGroups(ctx)
	if err != nil {
		return err
	}
	
	var group *Group
	for _, g := range groups {
		if g.ID == groupID {
			group = &g
			break
		}
	}
	
	if group == nil {
		return fmt.Errorf("group %s not found", groupID)
	}
	
	// Groups in v2 are managed through rooms/zones
	// We need to find the room/zone and update its children
	rooms, err := c.GetRooms(ctx)
	if err != nil {
		return err
	}
	
	for _, room := range rooms {
		for _, service := range room.Services {
			if service.RType == "grouped_light" && service.RID == groupID {
				// This is the room for our group
				// Find the device that contains the light
				devices, err := c.GetDevices(ctx)
				if err != nil {
					return err
				}
				
				var deviceID string
				for _, device := range devices {
					for _, svc := range device.Services {
						if svc.RType == "light" && svc.RID == lightID {
							deviceID = device.ID
							break
						}
					}
					if deviceID != "" {
						break
					}
				}
				
				if deviceID == "" {
					return fmt.Errorf("device containing light %s not found", lightID)
				}
				
				// Check if device is already in room
				for _, child := range room.Children {
					if child.RType == "device" && child.RID == deviceID {
						return nil // Already in the room
					}
				}
				
				// Add device to room
				room.Children = append(room.Children, ResourceIdentifier{
					RID:   deviceID,
					RType: "device",
				})
				
				// Update room
				update := RoomUpdate{
					Children: room.Children,
				}
				
				return c.UpdateRoom(ctx, room.ID, update)
			}
		}
	}
	
	// Check zones too
	zones, err := c.GetZones(ctx)
	if err != nil {
		return err
	}
	
	for _, zone := range zones {
		for _, service := range zone.Services {
			if service.RType == "grouped_light" && service.RID == groupID {
				// Add light directly to zone
				zone.Children = append(zone.Children, ResourceIdentifier{
					RID:   lightID,
					RType: "light",
				})
				
				// Update zone
				update := ZoneUpdate{
					Children: zone.Children,
				}
				
				return c.UpdateZone(ctx, zone.ID, update)
			}
		}
	}
	
	return fmt.Errorf("room or zone for group %s not found", groupID)
}

// RemoveLightFromGroup removes a light from a group
func (c *Client) RemoveLightFromGroup(ctx context.Context, groupID, lightID string) error {
	// Similar to AddLightToGroup but removes the device/light
	rooms, err := c.GetRooms(ctx)
	if err != nil {
		return err
	}
	
	for _, room := range rooms {
		for _, service := range room.Services {
			if service.RType == "grouped_light" && service.RID == groupID {
				// Find device containing the light
				devices, err := c.GetDevices(ctx)
				if err != nil {
					return err
				}
				
				var deviceID string
				for _, device := range devices {
					for _, svc := range device.Services {
						if svc.RType == "light" && svc.RID == lightID {
							deviceID = device.ID
							break
						}
					}
					if deviceID != "" {
						break
					}
				}
				
				// Remove device from room children
				var newChildren []ResourceIdentifier
				for _, child := range room.Children {
					if !(child.RType == "device" && child.RID == deviceID) {
						newChildren = append(newChildren, child)
					}
				}
				
				if len(newChildren) == len(room.Children) {
					return nil // Device wasn't in the room
				}
				
				// Update room
				update := RoomUpdate{
					Children: newChildren,
				}
				
				return c.UpdateRoom(ctx, room.ID, update)
			}
		}
	}
	
	// Check zones
	zones, err := c.GetZones(ctx)
	if err != nil {
		return err
	}
	
	for _, zone := range zones {
		for _, service := range zone.Services {
			if service.RType == "grouped_light" && service.RID == groupID {
				// Remove light from zone children
				var newChildren []ResourceIdentifier
				for _, child := range zone.Children {
					if !(child.RType == "light" && child.RID == lightID) {
						newChildren = append(newChildren, child)
					}
				}
				
				if len(newChildren) == len(zone.Children) {
					return nil // Light wasn't in the zone
				}
				
				// Update zone
				update := ZoneUpdate{
					Children: newChildren,
				}
				
				return c.UpdateZone(ctx, zone.ID, update)
			}
		}
	}
	
	return fmt.Errorf("room or zone for group %s not found", groupID)
}

// Room/Zone update operations

// UpdateRoom updates a room
func (c *Client) UpdateRoom(ctx context.Context, id string, update RoomUpdate) error {
	return c.putOK(ctx, fmt.Sprintf("/resource/room/%s", id), update)
}

// UpdateZone updates a zone
func (c *Client) UpdateZone(ctx context.Context, id string, update ZoneUpdate) error {
	return c.putOK(ctx, fmt.Sprintf("/resource/zone/%s", id), update)
}

// CreateZone creates a new zone
func (c *Client) CreateZone(ctx context.Context, zone ZoneCreate) (*Zone, error) {
	var response struct {
		Data   []Zone  `json:"data"`
		Errors []Error `json:"errors"`
	}
	
	respBody, err := c.post(ctx, "/resource/zone", zone)
	if err != nil {
		return nil, err
	}
	
	if err := json.Unmarshal(respBody, &response); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	
	if len(response.Errors) > 0 {
		return nil, fmt.Errorf("API error: %s", response.Errors[0].Description)
	}
	
	if len(response.Data) == 0 {
		return nil, fmt.Errorf("no zone returned")
	}
	
	return &response.Data[0], nil
}

// DeleteZone deletes a zone
func (c *Client) DeleteZone(ctx context.Context, id string) error {
	_, err := c.delete(ctx, fmt.Sprintf("/resource/zone/%s", id))
	return err
}

