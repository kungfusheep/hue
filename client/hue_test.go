package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client := NewClient("192.168.1.1", "test-username", http.DefaultClient)
	
	if client.bridgeIP != "192.168.1.1" {
		t.Errorf("Expected bridge IP 192.168.1.1, got %s", client.bridgeIP)
	}
	
	if client.username != "test-username" {
		t.Errorf("Expected username test-username, got %s", client.username)
	}
	
	if client.baseURL != "https://192.168.1.1/clip/v2" {
		t.Errorf("Expected base URL https://192.168.1.1/clip/v2, got %s", client.baseURL)
	}
}

func TestGetLights(t *testing.T) {
	// Create test server
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/clip/v2/resource/light" {
			t.Errorf("Expected path /clip/v2/resource/light, got %s", r.URL.Path)
		}
		
		if r.Header.Get("hue-application-key") != "test-key" {
			t.Errorf("Expected hue-application-key header test-key, got %s", r.Header.Get("hue-application-key"))
		}
		
		response := map[string]interface{}{
			"data": []Light{
				{
					ID:   "light-1",
					IDV1: "/lights/1",
					Type: "light",
					Metadata: Metadata{
						Name:      "Test Light",
						Archetype: "sultan_bulb",
					},
					On: OnState{On: true},
					Dimming: Dimming{Brightness: 50.0},
				},
			},
			"errors": []Error{},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	// Create client with test server
	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	// Test GetLights
	ctx := context.Background()
	lights, err := client.GetLights(ctx)
	if err != nil {
		t.Fatalf("GetLights failed: %v", err)
	}
	
	if len(lights) != 1 {
		t.Fatalf("Expected 1 light, got %d", len(lights))
	}
	
	light := lights[0]
	if light.ID != "light-1" {
		t.Errorf("Expected light ID light-1, got %s", light.ID)
	}
	
	if light.Metadata.Name != "Test Light" {
		t.Errorf("Expected light name Test Light, got %s", light.Metadata.Name)
	}
	
	if !light.On.On {
		t.Error("Expected light to be on")
	}
	
	if light.Dimming.Brightness != 50.0 {
		t.Errorf("Expected brightness 50.0, got %f", light.Dimming.Brightness)
	}
}

func TestUpdateLight(t *testing.T) {
	// Track request received
	var requestReceived map[string]interface{}
	
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PUT" {
			t.Errorf("Expected PUT method, got %s", r.Method)
		}
		
		if r.URL.Path != "/clip/v2/resource/light/light-1" {
			t.Errorf("Expected path /clip/v2/resource/light/light-1, got %s", r.URL.Path)
		}
		
		// Decode request body
		json.NewDecoder(r.Body).Decode(&requestReceived)
		
		// Return success response
		response := map[string]interface{}{
			"data": []map[string]string{
				{"rid": "light-1"},
			},
			"errors": []Error{},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	// Test turning light on
	ctx := context.Background()
	err := client.TurnOnLight(ctx, "light-1")
	if err != nil {
		t.Fatalf("TurnOnLight failed: %v", err)
	}
	
	// Check request
	if on, ok := requestReceived["on"].(map[string]interface{}); ok {
		if on["on"] != true {
			t.Error("Expected on.on to be true")
		}
	} else {
		t.Error("Expected on field in request")
	}
}

func TestUpdateLightBridgeErrors(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"data": []interface{}{},
			"errors": []Error{{
				Description: "parameter, /metadata/name, not available",
			}},
		})
	}))
	defer server.Close()

	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}

	err := client.UpdateLight(context.Background(), "light-1", LightUpdate{})
	if err == nil {
		t.Fatal("expected API error from bridge errors array")
	}
	if err.Error() != "API error: parameter, /metadata/name, not available" {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestUpdateLightEmptyBody(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}

	if err := client.UpdateLight(context.Background(), "light-1", LightUpdate{}); err != nil {
		t.Fatalf("empty body should succeed: %v", err)
	}
}

func TestUpdateLightNonJSONBody(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("not-json"))
	}))
	defer server.Close()

	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}

	err := client.UpdateLight(context.Background(), "light-1", LightUpdate{})
	if err == nil {
		t.Fatal("expected unmarshal error for non-JSON body")
	}
	if !strings.Contains(err.Error(), "failed to unmarshal response") {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestSetLightEffect(t *testing.T) {
	var requestReceived map[string]interface{}
	
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&requestReceived)
		
		response := map[string]interface{}{
			"data": []map[string]string{
				{"rid": "light-1"},
			},
			"errors": []Error{},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	// Test setting candle effect with duration
	ctx := context.Background()
	err := client.SetLightEffect(ctx, "light-1", "candle", 60)
	if err != nil {
		t.Fatalf("SetLightEffect failed: %v", err)
	}
	
	// Check effects field
	if effects, ok := requestReceived["effects"].(map[string]interface{}); ok {
		if effects["effect"] != "candle" {
			t.Errorf("Expected effect candle, got %v", effects["effect"])
		}
	} else {
		t.Error("Expected effects field in request")
	}
	
	// Check dynamics field
	if dynamics, ok := requestReceived["dynamics"].(map[string]interface{}); ok {
		if dynamics["duration"] != 60000.0 { // Should be converted to milliseconds
			t.Errorf("Expected duration 60000, got %v", dynamics["duration"])
		}
	} else {
		t.Error("Expected dynamics field in request")
	}
}

func TestHexToXY(t *testing.T) {
	tests := []struct {
		hex      string
		expectedX float64
		expectedY float64
		tolerance float64
	}{
		{"#FF0000", 0.64, 0.33, 0.1},  // Red
		{"#00FF00", 0.30, 0.60, 0.3},  // Green (wider tolerance)
		{"#0000FF", 0.15, 0.06, 0.1},  // Blue
		{"#FFFFFF", 0.31, 0.33, 0.1},  // White
		{"000000", 0.31, 0.33, 0.1},   // Black (defaults to white point)
	}
	
	for _, test := range tests {
		x, y := hexToXY(test.hex)
		
		if abs(x-test.expectedX) > test.tolerance {
			t.Errorf("hexToXY(%s) X: expected ~%f, got %f", test.hex, test.expectedX, x)
		}
		
		if abs(y-test.expectedY) > test.tolerance {
			t.Errorf("hexToXY(%s) Y: expected ~%f, got %f", test.hex, test.expectedY, y)
		}
	}
}

func TestAPIErrorHandling(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"data": []interface{}{},
			"errors": []Error{
				{
					Type:        "resource_not_found",
					Address:     "/lights/999",
					Description: "Light not found",
				},
			},
		}
		
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()
	
	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	ctx := context.Background()
	_, err := client.GetLight(ctx, "999")
	
	if err == nil {
		t.Fatal("Expected error for non-existent light")
	}
	
	if err.Error() != "API error: Light not found" {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestHTTPErrorHandling(t *testing.T) {
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()
	
	client := &Client{
		bridgeIP:   server.URL,
		username:   "invalid-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	ctx := context.Background()
	_, err := client.GetLights(ctx)
	
	if err == nil {
		t.Fatal("Expected error for unauthorized request")
	}
	
	if err.Error() != "HTTP 401: Unauthorized" {
		t.Errorf("Expected HTTP 401 error, got: %v", err)
	}
}

func TestContextCancellation(t *testing.T) {
	// Slow server that will be cancelled
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	
	client := &Client{
		bridgeIP:   server.URL,
		username:   "test-key",
		httpClient: server.Client(),
		baseURL:    server.URL + "/clip/v2",
	}
	
	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	
	_, err := client.GetLights(ctx)
	
	if err == nil {
		t.Fatal("Expected timeout error")
	}
}

// Helper function for float comparison
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}