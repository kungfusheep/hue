package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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
	// xy from the sRGB matrix in hexToXY, linearized with math.Pow.
	// Black stays the zero-sum white point.
	const tolerance = 1e-4
	tests := []struct {
		hex       string
		expectedX float64
		expectedY float64
	}{
		{"#FF0000", 0.6399999255, 0.3300000683},
		{"#00FF00", 0.3000000084, 0.6000000168},
		{"#0000FF", 0.1500000083, 0.0600000033},
		{"#FFFFFF", 0.3127266044, 0.3290231524},
		{"000000", 0.3127, 0.3290},
		// Mid-tone: exponent 2 and 2.4 diverge (primaries/grays do not).
		{"#FF8000", 0.5429908801, 0.4070366672},
	}

	for _, test := range tests {
		x, y := hexToXY(test.hex)

		if abs(x-test.expectedX) > tolerance {
			t.Errorf("hexToXY(%s) X: expected %.10f, got %.10f", test.hex, test.expectedX, x)
		}

		if abs(y-test.expectedY) > tolerance {
			t.Errorf("hexToXY(%s) Y: expected %.10f, got %.10f", test.hex, test.expectedY, y)
		}
	}

	// Integer-power linearization (exponent truncated to 2) must not match.
	const wrongX, wrongY = 0.5243545494, 0.4218361071
	x, y := hexToXY("#FF8000")
	if abs(x-wrongX) <= tolerance && abs(y-wrongY) <= tolerance {
		t.Errorf("hexToXY(#FF8000) matched integer-power xy (%.10f, %.10f); gamma 2.4 is required", wrongX, wrongY)
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