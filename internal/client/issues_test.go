package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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

		// Verify the default limit is set
		if first, ok := req.Variables["first"].(float64); !ok || first != 50 {
			t.Errorf("Expected first to be 50, got %v", req.Variables["first"])
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
					],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": ""
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

	opts := ListIssuesOptions{}
	resp, err := client.ListIssues(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Issues.Nodes) != 1 {
		t.Errorf("Expected 1 issue, got %d", len(resp.Issues.Nodes))
	}

	if resp.Issues.Nodes[0].Identifier != "TEST-1" {
		t.Errorf("Expected identifier 'TEST-1', got '%s'", resp.Issues.Nodes[0].Identifier)
	}

	if resp.Issues.PageInfo.HasNextPage {
		t.Error("Expected hasNextPage to be false")
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
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": ""
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

	opts := ListIssuesOptions{TeamKey: "ENG"}
	_, err := client.ListIssues(context.Background(), opts)
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

		reqInput, ok := req.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input object in variables, got %T", req.Variables["input"])
		}
		// Assert that requried fields were sent
		require.Equal(t, "New Test Issue", reqInput["title"])
		require.Equal(t, "Test description", reqInput["description"])
		require.Equal(t, "team-123", reqInput["teamId"])
		require.Equal(t, "proj-123", reqInput["projectId"])
		require.Equal(t, "user-123", reqInput["assigneeId"])

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
		ProjectID:   "proj-123",
		AssigneeID:  "user-123",
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

func TestClient_UpdateIssue(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if !strings.Contains(req.Query, "issueUpdate") {
			t.Errorf("Expected issueUpdate mutation, got query: %s", req.Query)
		}

		input, ok := req.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input object in variables, got %T", req.Variables["input"])
		}

		if req.Variables["id"] != "TEST-123" {
			t.Errorf("Expected id to be TEST-123, got %v", req.Variables["id"])
		}
		if input["title"] != "Updated title" {
			t.Errorf("Expected title to be Updated title, got %v", input["title"])
		}
		if input["description"] != "Updated description" {
			t.Errorf("Expected description to be Updated description, got %v", input["description"])
		}
		if input["projectId"] != "proj-123" {
			t.Errorf("Expected projectId to be proj-123, got %v", input["projectId"])
		}
		if priority, ok := input["priority"].(float64); !ok || priority != 2 {
			t.Errorf("Expected priority to be 2, got %v", input["priority"])
		}
		if input["assigneeId"] != "user-123" {
			t.Errorf("Expected assigneeId to be user-123, got %v", input["assigneeId"])
		}

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "issue-123",
						"identifier": "TEST-123",
						"title": "Updated title",
						"url": "https://linear.app/test/issue/TEST-123"
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

	title := "Updated title"
	description := "Updated description"
	priority := 2
	projectID := "proj-123"
	assigneeID := "user-123"
	input := UpdateIssueInput{
		Title:       &title,
		Description: &description,
		Priority:    &priority,
		ProjectID:   &projectID,
		AssigneeID:  &assigneeID,
	}

	resp, err := client.UpdateIssue(context.Background(), "TEST-123", input)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if !resp.IssueUpdate.Success {
		t.Error("Expected success to be true")
	}
	if resp.IssueUpdate.Issue == nil {
		t.Fatal("Expected issue to be non-nil")
	}
	if resp.IssueUpdate.Issue.Identifier != "TEST-123" {
		t.Errorf("Expected identifier TEST-123, got %s", resp.IssueUpdate.Issue.Identifier)
	}
}

func TestClient_UpdateIssue_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Errors: []struct {
				Message string `json:"message"`
				Path    []any  `json:"path,omitempty"`
			}{
				{Message: "Issue not found"},
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

	title := "Updated title"
	_, err := client.UpdateIssue(context.Background(), "NONEXISTENT", UpdateIssueInput{
		Title: &title,
	})
	if err == nil {
		t.Error("Expected an error, got nil")
	}
}

func TestClient_ListIssues_WithCustomLimit(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the custom limit is set
		if first, ok := req.Variables["first"].(float64); !ok || first != 100 {
			t.Errorf("Expected first to be 100, got %v", req.Variables["first"])
		}

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [],
					"pageInfo": {
						"hasNextPage": false,
						"endCursor": ""
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

	opts := ListIssuesOptions{Limit: 100}
	_, err := client.ListIssues(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
}

func TestClient_ListIssues_WithPagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Check if this is a paginated request
		cursor, hasCursor := req.Variables["after"].(string)

		var response graphQLResponse
		if !hasCursor || cursor == "" {
			// First page
			response = graphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [
							{
								"id": "issue-1",
								"identifier": "TEST-1",
								"title": "Test Issue 1",
								"priority": 1,
								"priorityLabel": "High",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-01T00:00:00Z",
								"url": "https://linear.app/test/issue/TEST-1"
							}
						],
						"pageInfo": {
							"hasNextPage": true,
							"endCursor": "cursor-1"
						}
					}
				}`),
			}
		} else {
			// Second page
			response = graphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [
							{
								"id": "issue-2",
								"identifier": "TEST-2",
								"title": "Test Issue 2",
								"priority": 1,
								"priorityLabel": "High",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-01T00:00:00Z",
								"url": "https://linear.app/test/issue/TEST-2"
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": "cursor-2"
						}
					}
				}`),
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	// First page
	opts := ListIssuesOptions{}
	resp, err := client.ListIssues(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(resp.Issues.Nodes) != 1 {
		t.Errorf("Expected 1 issue on first page, got %d", len(resp.Issues.Nodes))
	}

	if !resp.Issues.PageInfo.HasNextPage {
		t.Error("Expected hasNextPage to be true on first page")
	}

	// Second page
	opts.After = resp.Issues.PageInfo.EndCursor
	resp2, err := client.ListIssues(context.Background(), opts)
	if err != nil {
		t.Fatalf("Expected no error on second page, got %v", err)
	}

	if len(resp2.Issues.Nodes) != 1 {
		t.Errorf("Expected 1 issue on second page, got %d", len(resp2.Issues.Nodes))
	}

	if resp2.Issues.PageInfo.HasNextPage {
		t.Error("Expected hasNextPage to be false on second page")
	}
}

func TestClient_ListAllIssues(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		callCount++
		var response graphQLResponse

		if callCount == 1 {
			// First page
			response = graphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [
							{
								"id": "issue-1",
								"identifier": "TEST-1",
								"title": "Test Issue 1",
								"priority": 1,
								"priorityLabel": "High",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-01T00:00:00Z",
								"url": "https://linear.app/test/issue/TEST-1"
							}
						],
						"pageInfo": {
							"hasNextPage": true,
							"endCursor": "cursor-1"
						}
					}
				}`),
			}
		} else {
			// Second page
			response = graphQLResponse{
				Data: json.RawMessage(`{
					"issues": {
						"nodes": [
							{
								"id": "issue-2",
								"identifier": "TEST-2",
								"title": "Test Issue 2",
								"priority": 1,
								"priorityLabel": "High",
								"createdAt": "2024-01-01T00:00:00Z",
								"updatedAt": "2024-01-01T00:00:00Z",
								"url": "https://linear.app/test/issue/TEST-2"
							}
						],
						"pageInfo": {
							"hasNextPage": false,
							"endCursor": "cursor-2"
						}
					}
				}`),
			}
		}

		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	issues, err := client.ListAllIssues(context.Background(), ListIssuesOptions{})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(issues) != 2 {
		t.Errorf("Expected 2 issues total, got %d", len(issues))
	}

	if callCount != 2 {
		t.Errorf("Expected 2 API calls, got %d", callCount)
	}

	if issues[0].Identifier != "TEST-1" {
		t.Errorf("Expected first issue identifier 'TEST-1', got '%s'", issues[0].Identifier)
	}

	if issues[1].Identifier != "TEST-2" {
		t.Errorf("Expected second issue identifier 'TEST-2', got '%s'", issues[1].Identifier)
	}
}

func TestNextPageCursor_EmptyEndCursor(t *testing.T) {
	_, _, err := nextPageCursor("", PageInfo{
		HasNextPage: true,
		EndCursor:   "",
	})
	if err == nil {
		t.Fatal("Expected error when hasNextPage is true and endCursor is empty")
	}

	if !strings.Contains(err.Error(), "endCursor is empty") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNextPageCursor_UnchangedCursor(t *testing.T) {
	_, _, err := nextPageCursor("cursor-1", PageInfo{
		HasNextPage: true,
		EndCursor:   "cursor-1",
	})
	if err == nil {
		t.Fatal("Expected error when endCursor does not advance")
	}

	if !strings.Contains(err.Error(), "did not advance") {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestNextPageCursor_NoNextPage(t *testing.T) {
	cursor, hasNext, err := nextPageCursor("cursor-1", PageInfo{
		HasNextPage: false,
		EndCursor:   "cursor-2",
	})
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if hasNext {
		t.Fatal("Expected hasNext to be false")
	}
	if cursor != "" {
		t.Errorf("Expected empty cursor, got %q", cursor)
	}
}
