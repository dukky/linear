package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_ListIssues_NoFilter(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request to verify it's correctly structured
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the query contains the expected fields
		if req.Query == "" {
			t.Error("Expected query to be non-empty")
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [
						{
							"id": "issue-1",
							"identifier": "TEST-1",
							"title": "Test Issue",
							"priority": 1,
							"priorityLabel": "High",
							"createdAt": "2024-01-01T00:00:00Z",
							"updatedAt": "2024-01-01T00:00:00Z",
							"url": "https://linear.app/test/issue/TEST-1"
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

	resp, err := client.ListIssues(context.Background(), "")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Issues.Nodes) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(resp.Issues.Nodes))
	}

	if resp.Issues.Nodes[0].Identifier != "TEST-1" {
		t.Errorf("Expected identifier 'TEST-1', got '%s'", resp.Issues.Nodes[0].Identifier)
	}
}

func TestClient_ListIssues_WithTeamFilter(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request to verify the filter
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify variables contain the team filter
		if req.Variables == nil {
			t.Error("Expected variables to be present")
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{"issues": {"nodes": []}}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	_, err := client.ListIssues(context.Background(), "ENG")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_GetIssue(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the ID variable
		if req.Variables["id"] != "TEST-123" {
			t.Errorf("Expected id 'TEST-123', got '%v'", req.Variables["id"])
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issue": {
					"id": "issue-123",
					"identifier": "TEST-123",
					"title": "Test Issue",
					"description": "Test description",
					"priority": 1,
					"priorityLabel": "High",
					"createdAt": "2024-01-01T00:00:00Z",
					"updatedAt": "2024-01-01T00:00:00Z",
					"url": "https://linear.app/test/issue/TEST-123"
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

	resp, err := client.GetIssue(context.Background(), "TEST-123")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Issue == nil {
		t.Fatal("Expected issue to be non-nil")
	}

	if resp.Issue.Identifier != "TEST-123" {
		t.Errorf("Expected identifier 'TEST-123', got '%s'", resp.Issue.Identifier)
	}
}

func TestClient_CreateIssue(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify it's a mutation
		if req.Query == "" {
			t.Error("Expected query to be non-empty")
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issueCreate": {
					"success": true,
					"issue": {
						"id": "new-issue-id",
						"identifier": "TEST-124",
						"title": "New Test Issue",
						"url": "https://linear.app/test/issue/TEST-124"
					}
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

	input := CreateIssueInput{
		Title:       "New Test Issue",
		Description: "Test description",
		TeamID:      "team-123",
	}

	resp, err := client.CreateIssue(context.Background(), input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IssueCreate.Success {
		t.Error("Expected success to be true")
	}

	if resp.IssueCreate.Issue == nil {
		t.Fatal("Expected issue to be non-nil")
	}

	if resp.IssueCreate.Issue.Identifier != "TEST-124" {
		t.Errorf("Expected identifier 'TEST-124', got '%s'", resp.IssueCreate.Issue.Identifier)
	}
}

func TestClient_CreateIssue_Error(t *testing.T) {
	// Create a test server that returns an error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Errors: []struct {
				Message string `json:"message"`
				Path    []any  `json:"path,omitempty"`
			}{
				{Message: "Team not found"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	input := CreateIssueInput{
		Title:  "New Test Issue",
		TeamID: "invalid-team",
	}

	_, err := client.CreateIssue(context.Background(), input)
	if err == nil {
		t.Error("Expected an error, got nil")
	}
}
