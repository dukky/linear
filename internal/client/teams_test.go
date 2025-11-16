package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_ListTeams(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"teams": {
					"nodes": [
						{
							"id": "team-1",
							"key": "ENG",
							"name": "Engineering",
							"description": "Engineering team"
						},
						{
							"id": "team-2",
							"key": "PROD",
							"name": "Product",
							"description": null
						}
					]
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	resp, err := client.ListTeams(context.Background())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Teams.Nodes) != 2 {
		t.Errorf("Expected 2 teams, got %d", len(resp.Teams.Nodes))
	}

	if resp.Teams.Nodes[0].Key != "ENG" {
		t.Errorf("Expected first team key to be 'ENG', got '%s'", resp.Teams.Nodes[0].Key)
	}

	if resp.Teams.Nodes[0].Description == nil {
		t.Error("Expected first team to have a description")
	}

	if resp.Teams.Nodes[1].Description != nil {
		t.Error("Expected second team to have no description")
	}
}

func TestClient_GetTeamByKey(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the key variable
		if req.Variables["key"] != "ENG" {
			t.Errorf("Expected key 'ENG', got '%v'", req.Variables["key"])
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"teams": {
					"nodes": [
						{
							"id": "team-1",
							"key": "ENG",
							"name": "Engineering"
						}
					]
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	resp, err := client.GetTeamByKey(context.Background(), "ENG")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Teams.Nodes) != 1 {
		t.Errorf("Expected 1 team, got %d", len(resp.Teams.Nodes))
	}

	if resp.Teams.Nodes[0].Key != "ENG" {
		t.Errorf("Expected team key to be 'ENG', got '%s'", resp.Teams.Nodes[0].Key)
	}
}

func TestClient_GetTeamByKey_NotFound(t *testing.T) {
	// Create a test server that returns empty results
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"teams": {
					"nodes": []
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	resp, err := client.GetTeamByKey(context.Background(), "NONEXISTENT")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Teams.Nodes) != 0 {
		t.Errorf("Expected 0 teams, got %d", len(resp.Teams.Nodes))
	}
}
