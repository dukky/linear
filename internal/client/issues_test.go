package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// projectSelectionBlock returns the text between the first "project {" and
// the next "}" in a GraphQL query, i.e. the fields selected on the issue's
// nested project. A plain strings.Contains(query, "url") is not enough to
// prove the project carries url, because issues select their own url too.
func projectSelectionBlock(t *testing.T, query string) string {
	t.Helper()
	const open = "project {"
	start := strings.Index(query, open)
	if start == -1 {
		t.Fatalf("Expected query to contain a %q block, got query: %s", open, query)
	}
	start += len(open)
	end := strings.Index(query[start:], "}")
	if end == -1 {
		t.Fatalf("Unterminated %q block in query: %s", open, query)
	}
	return query[start : start+end]
}

// assertIssueProjectSelectsURL checks that the nested project block of an
// issue query/mutation selects url.
func assertIssueProjectSelectsURL(t *testing.T, query string) {
	t.Helper()
	block := projectSelectionBlock(t, query)
	if !strings.Contains(block, "url") {
		t.Errorf("Expected nested project block to select url, got project block: {%s}", block)
	}
}

// assertQuerySelectsIssueLinks checks that a single-issue query or mutation
// selects the sub-issue and both relation connections. "relations(" is
// lowercase, so it does not match inside "inverseRelations(".
func assertQuerySelectsIssueLinks(t *testing.T, query string) {
	t.Helper()
	for _, want := range []string{"children(first: 50)", "relations(first: 50)", "inverseRelations(first: 50)"} {
		if !strings.Contains(query, want) {
			t.Errorf("Expected query to contain %q, got query: %s", want, query)
		}
	}
}

const mockIssueProjectURL = "https://linear.app/test/project/mobile-app-abc123"

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
		assertIssueProjectSelectsURL(t, req.Query)

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
							"url": "https://linear.app/test/issue/TEST-1",
							"project": {
								"id": "proj-123",
								"name": "Mobile App",
								"url": "https://linear.app/test/project/mobile-app-abc123"
							}
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

	if resp.Issues.Nodes[0].Project == nil {
		t.Fatal("Expected issue project to be hydrated, got nil")
	}
	if resp.Issues.Nodes[0].Project.URL != mockIssueProjectURL {
		t.Errorf("Expected project URL %q, got %q", mockIssueProjectURL, resp.Issues.Nodes[0].Project.URL)
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
		assertIssueProjectSelectsURL(t, req.Query)

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
					"url": "https://linear.app/test/issue/TEST-123",
					"project": {
						"id": "proj-123",
						"name": "Mobile App",
						"url": "https://linear.app/test/project/mobile-app-abc123"
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

	if resp.Issue.Project == nil {
		t.Fatal("Expected issue project to be hydrated, got nil")
	}
	if resp.Issue.Project.URL != mockIssueProjectURL {
		t.Errorf("Expected project URL %q, got %q", mockIssueProjectURL, resp.Issue.Project.URL)
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

		// The mutation must request the full issue payload (state, priority, ...)
		// so that --json output is hydrated (regression for GitHub issue #12).
		if !strings.Contains(req.Query, "state {") || !strings.Contains(req.Query, "priorityLabel") {
			t.Errorf("Expected issueCreate to select the full issue payload (state, priorityLabel), got query: %s", req.Query)
		}
		assertIssueProjectSelectsURL(t, req.Query)

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
						"priority": 2,
						"priorityLabel": "High",
						"url": "https://linear.app/test/issue/TEST-124",
						"state": {
							"name": "In Progress",
							"color": "#f2c94c",
							"type": "started"
						},
						"project": {
							"id": "proj-123",
							"name": "Mobile App",
							"url": "https://linear.app/test/project/mobile-app-abc123"
						}
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

	// The created issue's relations/enums must be hydrated (issue #12): the
	// response should carry the state the issue was created in, not null.
	if resp.IssueCreate.Issue.State == nil {
		t.Error("Expected issue state to be hydrated, got nil")
	} else if resp.IssueCreate.Issue.State.Name != "In Progress" {
		t.Errorf("Expected state name 'In Progress', got '%s'", resp.IssueCreate.Issue.State.Name)
	}
	if resp.IssueCreate.Issue.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", resp.IssueCreate.Issue.Priority)
	}

	if resp.IssueCreate.Issue.Project == nil {
		t.Fatal("Expected issue project to be hydrated, got nil")
	}
	if resp.IssueCreate.Issue.Project.URL != mockIssueProjectURL {
		t.Errorf("Expected project URL %q, got %q", mockIssueProjectURL, resp.IssueCreate.Issue.Project.URL)
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

		// The mutation must request the full issue payload (state, priority, ...)
		// so that --json output is hydrated (regression for GitHub issue #12).
		if !strings.Contains(req.Query, "state {") || !strings.Contains(req.Query, "priorityLabel") {
			t.Errorf("Expected issueUpdate to select the full issue payload (state, priorityLabel), got query: %s", req.Query)
		}
		assertIssueProjectSelectsURL(t, req.Query)

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
						"priority": 2,
						"priorityLabel": "High",
						"url": "https://linear.app/test/issue/TEST-123",
						"state": {
							"name": "In Progress",
							"color": "#f2c94c",
							"type": "started"
						},
						"project": {
							"id": "proj-123",
							"name": "Mobile App",
							"url": "https://linear.app/test/project/mobile-app-abc123"
						}
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

	// The updated issue's relations/enums must be hydrated (issue #12): the
	// response should carry the new state, not null.
	if resp.IssueUpdate.Issue.State == nil {
		t.Error("Expected issue state to be hydrated, got nil")
	} else if resp.IssueUpdate.Issue.State.Name != "In Progress" {
		t.Errorf("Expected state name 'In Progress', got '%s'", resp.IssueUpdate.Issue.State.Name)
	}
	if resp.IssueUpdate.Issue.Priority != 2 {
		t.Errorf("Expected priority 2, got %d", resp.IssueUpdate.Issue.Priority)
	}

	if resp.IssueUpdate.Issue.Project == nil {
		t.Fatal("Expected issue project to be hydrated, got nil")
	}
	if resp.IssueUpdate.Issue.Project.URL != mockIssueProjectURL {
		t.Errorf("Expected project URL %q, got %q", mockIssueProjectURL, resp.IssueUpdate.Issue.Project.URL)
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

// issuePage builds a mock issues page with identifiers TEST-start..TEST-(start+n-1).
func issuePage(start, n int, hasNext bool, cursor string) graphQLResponse {
	nodes := make([]map[string]interface{}, 0, n)
	for i := start; i < start+n; i++ {
		nodes = append(nodes, map[string]interface{}{
			"id":         fmt.Sprintf("issue-%d", i),
			"identifier": fmt.Sprintf("TEST-%d", i),
			"title":      fmt.Sprintf("Test Issue %d", i),
		})
	}
	data, _ := json.Marshal(map[string]interface{}{
		"issues": map[string]interface{}{
			"nodes":    nodes,
			"pageInfo": map[string]interface{}{"hasNextPage": hasNext, "endCursor": cursor},
		},
	})
	return graphQLResponse{Data: data}
}

// requestFirst returns the integer "first" variable from a GraphQL request.
func requestFirst(t *testing.T, req graphQLRequest) int {
	t.Helper()
	f, ok := req.Variables["first"].(float64)
	if !ok {
		t.Fatalf("Expected numeric first variable, got %#v", req.Variables["first"])
	}
	return int(f)
}

func TestListAllIssues_MaxIssues(t *testing.T) {
	var requests int
	var firsts []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}
		requests++
		firsts = append(firsts, requestFirst(t, req))

		// Server ignores first and always serves pages of 2.
		switch requests {
		case 1:
			json.NewEncoder(w).Encode(issuePage(1, 2, true, "cursor-1"))
		case 2:
			json.NewEncoder(w).Encode(issuePage(3, 2, true, "cursor-2"))
		default:
			json.NewEncoder(w).Encode(issuePage(5, 2, false, "cursor-3"))
		}
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	issues, err := client.ListAllIssues(context.Background(), ListIssuesOptions{MaxIssues: 3})
	require.NoError(t, err)
	require.Len(t, issues, 3)
	require.Equal(t, "TEST-1", issues[0].Identifier)
	require.Equal(t, "TEST-3", issues[2].Identifier)

	require.Equal(t, 2, requests, "pagination should stop once MaxIssues is reached; third page must not be fetched")
	for i, f := range firsts {
		require.LessOrEqualf(t, f, issuePageSize, "request %d asked for first=%d", i+1, f)
	}
	// First request asks for min(100, 3) = 3; second for the remaining 1.
	require.Equal(t, []int{3, 1}, firsts)
}

func TestListAllIssues_DefaultPageSize(t *testing.T) {
	var requests int
	var firsts []int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}
		requests++
		firsts = append(firsts, requestFirst(t, req))
		json.NewEncoder(w).Encode(issuePage(1, 120, false, "cursor-1"))
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	issues, err := client.ListAllIssues(context.Background(), ListIssuesOptions{MaxIssues: 0, Limit: 250})
	require.NoError(t, err)
	require.Len(t, issues, 120)
	require.Equal(t, 1, requests)
	require.Equal(t, []int{issuePageSize}, firsts)
	require.Equal(t, 100, firsts[0])
}

// captureIssueFilter starts a server that records the decoded filter variable
// of each ListIssues request and returns an empty page. The returned counter
// reports how many requests reached the server.
func captureIssueFilter(t *testing.T) (*Client, *map[string]interface{}, *int) {
	t.Helper()
	var filter map[string]interface{}
	var requests int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}
		filter = nil
		if raw, ok := req.Variables["filter"]; ok {
			m, ok := raw.(map[string]interface{})
			if !ok {
				t.Errorf("Expected filter to be an object, got %T", raw)
			}
			filter = m
		}
		json.NewEncoder(w).Encode(issuePage(1, 1, false, "cursor-1"))
	}))
	t.Cleanup(server.Close)

	c := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}
	return c, &filter, &requests
}

// filterPath walks nested filter maps by key, failing the test if any level
// is missing or not an object.
func filterPath(t *testing.T, filter map[string]interface{}, keys ...string) interface{} {
	t.Helper()
	var cur interface{} = filter
	for i, k := range keys {
		m, ok := cur.(map[string]interface{})
		if !ok {
			t.Fatalf("filter path %v: %v is %T, not an object (filter: %#v)", keys, keys[:i], cur, filter)
		}
		cur, ok = m[k]
		if !ok {
			t.Fatalf("filter path %v: key %q missing (filter: %#v)", keys, k, filter)
		}
	}
	return cur
}

func TestClient_ListIssues_AssigneeIsMeFilter(t *testing.T) {
	c, filter, _ := captureIssueFilter(t)

	_, err := c.ListIssues(context.Background(), ListIssuesOptions{AssigneeIsMe: true, AssigneeEmail: "ignored@example.com"})
	require.NoError(t, err)

	require.Equal(t, true, filterPath(t, *filter, "assignee", "isMe", "eq"))
	assignee := filterPath(t, *filter, "assignee").(map[string]interface{})
	require.NotContains(t, assignee, "email", "isMe must take precedence over email")
}

func TestClient_ListIssues_AssigneeEmailFilter(t *testing.T) {
	c, filter, _ := captureIssueFilter(t)

	_, err := c.ListIssues(context.Background(), ListIssuesOptions{AssigneeEmail: "dev@example.com"})
	require.NoError(t, err)

	require.Equal(t, "dev@example.com", filterPath(t, *filter, "assignee", "email", "eq"))
}

func TestClient_ListIssues_StateNameFilter(t *testing.T) {
	c, filter, _ := captureIssueFilter(t)

	_, err := c.ListIssues(context.Background(), ListIssuesOptions{StateName: "In Progress"})
	require.NoError(t, err)

	require.Equal(t, "In Progress", filterPath(t, *filter, "state", "name", "eqIgnoreCase"))
}

func TestClient_ListIssues_StateTypesFilter(t *testing.T) {
	c, filter, _ := captureIssueFilter(t)

	_, err := c.ListIssues(context.Background(), ListIssuesOptions{StateTypes: []string{"started", "unstarted"}})
	require.NoError(t, err)

	require.Equal(t, []interface{}{"started", "unstarted"}, filterPath(t, *filter, "state", "type", "in"))
}

func TestClient_ListIssues_StateNameAndTypesCombine(t *testing.T) {
	c, filter, _ := captureIssueFilter(t)

	_, err := c.ListIssues(context.Background(), ListIssuesOptions{StateName: "Review", StateTypes: []string{"started"}})
	require.NoError(t, err)

	require.Equal(t, "Review", filterPath(t, *filter, "state", "name", "eqIgnoreCase"))
	require.Equal(t, []interface{}{"started"}, filterPath(t, *filter, "state", "type", "in"))
}

// TestListAllIssues_CombinedFilters checks that every filter lands in one
// IssueFilter object and survives the ListAllIssues paging path.
func TestListAllIssues_CombinedFilters(t *testing.T) {
	c, filter, requests := captureIssueFilter(t)

	issues, err := c.ListAllIssues(context.Background(), ListIssuesOptions{
		TeamKey:       "ENG",
		ProjectID:     "project-1",
		AssigneeEmail: "dev@example.com",
		StateTypes:    []string{"started"},
		MaxIssues:     5,
	})
	require.NoError(t, err)
	require.Len(t, issues, 1)
	require.Equal(t, 1, *requests)

	require.Equal(t, "ENG", filterPath(t, *filter, "team", "key", "eq"))
	require.Equal(t, "project-1", filterPath(t, *filter, "project", "id", "eq"))
	require.Equal(t, "dev@example.com", filterPath(t, *filter, "assignee", "email", "eq"))
	require.Equal(t, []interface{}{"started"}, filterPath(t, *filter, "state", "type", "in"))
	require.Len(t, *filter, 4)
}

func TestListAllIssues_UnknownStateTypeRejectedBeforeRequest(t *testing.T) {
	c, _, requests := captureIssueFilter(t)

	_, err := c.ListAllIssues(context.Background(), ListIssuesOptions{StateTypes: []string{"started", "in-review"}})
	require.Error(t, err)
	require.Contains(t, err.Error(), `"in-review"`)
	for _, v := range ValidStateTypes {
		require.Contains(t, err.Error(), v)
	}
	require.Equal(t, 0, *requests, "invalid state type must be rejected before any request")
}

func TestClient_CreateIssue_WithParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		input, ok := req.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input object in variables, got %T", req.Variables["input"])
		}
		require.Equal(t, "parent-uuid-1", input["parentId"])
		assertQuerySelectsIssueLinks(t, req.Query)

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issueCreate": {
					"success": true,
					"issue": {
						"id": "child-uuid",
						"identifier": "TEST-2",
						"title": "Child",
						"parent": {
							"id": "parent-uuid-1",
							"identifier": "TEST-1",
							"title": "Parent",
							"url": "https://linear.app/test/issue/TEST-1"
						}
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

	resp, err := client.CreateIssue(context.Background(), CreateIssueInput{
		Title:    "Child",
		TeamID:   "team-123",
		ParentID: "parent-uuid-1",
	})
	require.NoError(t, err)
	require.NotNil(t, resp.IssueCreate.Issue)
	require.NotNil(t, resp.IssueCreate.Issue.Parent)
	require.Equal(t, "TEST-1", resp.IssueCreate.Issue.Parent.Identifier)
}

func TestClient_UpdateIssue_WithParent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		input, ok := req.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input object in variables, got %T", req.Variables["input"])
		}
		require.Equal(t, "parent-uuid-2", input["parentId"])
		assertQuerySelectsIssueLinks(t, req.Query)
		// Only the parent was set, so no other field may be sent.
		require.Len(t, input, 1, "expected only parentId in input, got %v", input)

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issueUpdate": {
					"success": true,
					"issue": {
						"id": "issue-uuid",
						"identifier": "TEST-3",
						"title": "Child",
						"parent": {
							"id": "parent-uuid-2",
							"identifier": "TEST-1",
							"title": "Parent"
						}
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

	parentID := "parent-uuid-2"
	resp, err := client.UpdateIssue(context.Background(), "TEST-3", UpdateIssueInput{ParentID: &parentID})
	require.NoError(t, err)
	require.NotNil(t, resp.IssueUpdate.Issue)
	require.NotNil(t, resp.IssueUpdate.Issue.Parent)
	require.Equal(t, "parent-uuid-2", resp.IssueUpdate.Issue.Parent.ID)
}

func TestClient_CreateIssueRelation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if !strings.Contains(req.Query, "issueRelationCreate") {
			t.Errorf("Expected issueRelationCreate mutation, got query: %s", req.Query)
		}
		if !strings.Contains(req.Query, "issueRelation {") {
			t.Errorf("Expected mutation to select issueRelation, got query: %s", req.Query)
		}
		if !strings.Contains(req.Query, "$input: IssueRelationCreateInput!") {
			t.Errorf("Expected $input: IssueRelationCreateInput! variable, got query: %s", req.Query)
		}

		input, ok := req.Variables["input"].(map[string]interface{})
		if !ok {
			t.Fatalf("Expected input object in variables, got %T", req.Variables["input"])
		}
		require.Equal(t, "blocker-uuid", input["issueId"])
		require.Equal(t, "blocked-uuid", input["relatedIssueId"])
		require.Equal(t, "blocks", input["type"])

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issueRelationCreate": {
					"success": true,
					"issueRelation": {
						"id": "rel-1",
						"type": "blocks",
						"issue": {"id": "blocker-uuid", "identifier": "TEST-1", "title": "Blocker"},
						"relatedIssue": {"id": "blocked-uuid", "identifier": "TEST-2", "title": "Blocked"}
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

	resp, err := client.CreateIssueRelation(context.Background(), CreateIssueRelationInput{
		IssueID:        "blocker-uuid",
		RelatedIssueID: "blocked-uuid",
		Type:           RelationTypeBlocks,
	})
	require.NoError(t, err)
	require.True(t, resp.IssueRelationCreate.Success)

	rel := resp.IssueRelationCreate.IssueRelation
	require.NotNil(t, rel)
	require.Equal(t, "rel-1", rel.ID)
	require.Equal(t, "blocks", rel.Type)
	require.NotNil(t, rel.Issue)
	require.Equal(t, "TEST-1", rel.Issue.Identifier)
	require.Equal(t, "blocker-uuid", rel.Issue.ID)
	require.NotNil(t, rel.RelatedIssue)
	require.Equal(t, "TEST-2", rel.RelatedIssue.Identifier)
	require.Equal(t, "blocked-uuid", rel.RelatedIssue.ID)
}

func TestClient_CreateIssueRelation_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Errors: []struct {
				Message string `json:"message"`
				Path    []any  `json:"path,omitempty"`
			}{
				{Message: "Entity not found: Issue"},
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

	resp, err := client.CreateIssueRelation(context.Background(), CreateIssueRelationInput{
		IssueID:        "a",
		RelatedIssueID: "b",
		Type:           RelationTypeRelated,
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "Entity not found: Issue")
	require.Nil(t, resp)
}

func TestClient_GetIssue_WithLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if !strings.Contains(req.Query, "parent {") {
			t.Errorf("Expected query to select parent, got query: %s", req.Query)
		}
		assertQuerySelectsIssueLinks(t, req.Query)
		assertIssueProjectSelectsURL(t, req.Query)

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issue": {
					"id": "issue-uuid",
					"identifier": "TEST-5",
					"title": "Middle",
					"project": {"id": "proj-123", "name": "Mobile App", "url": "https://linear.app/test/project/mobile-app-abc123"},
					"parent": {"id": "parent-uuid", "identifier": "TEST-1", "title": "Epic", "url": "https://linear.app/test/issue/TEST-1"},
					"children": {"nodes": [
						{"id": "c1", "identifier": "TEST-6", "title": "Child one", "url": "https://linear.app/test/issue/TEST-6"},
						{"id": "c2", "identifier": "TEST-7", "title": "Child two", "url": "https://linear.app/test/issue/TEST-7"}
					]},
					"relations": {"nodes": [
						{
							"id": "rel-1",
							"type": "blocks",
							"issue": {"id": "issue-uuid", "identifier": "TEST-5", "title": "Middle"},
							"relatedIssue": {"id": "other-uuid", "identifier": "TEST-9", "title": "Downstream"}
						},
						{
							"id": "rel-2",
							"type": "related",
							"issue": {"id": "issue-uuid", "identifier": "TEST-5", "title": "Middle"},
							"relatedIssue": {"id": "rel-uuid", "identifier": "TEST-10", "title": "Sibling"}
						}
					]},
					"inverseRelations": {"nodes": [
						{
							"id": "rel-3",
							"type": "blocks",
							"issue": {"id": "upstream-uuid", "identifier": "TEST-4", "title": "Upstream", "url": "https://linear.app/test/issue/TEST-4"},
							"relatedIssue": {"id": "issue-uuid", "identifier": "TEST-5", "title": "Middle", "url": "https://linear.app/test/issue/TEST-5"}
						}
					]}
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

	resp, err := client.GetIssue(context.Background(), "TEST-5")
	require.NoError(t, err)
	issue := resp.Issue
	require.NotNil(t, issue)

	require.NotNil(t, issue.Parent)
	require.Equal(t, IssueRef{ID: "parent-uuid", Identifier: "TEST-1", Title: "Epic", URL: "https://linear.app/test/issue/TEST-1"}, *issue.Parent)

	require.NotNil(t, issue.Children)
	require.Len(t, issue.Children.Nodes, 2)
	require.Equal(t, "TEST-6", issue.Children.Nodes[0].Identifier)
	require.Equal(t, "TEST-7", issue.Children.Nodes[1].Identifier)
	require.Equal(t, "Child two", issue.Children.Nodes[1].Title)

	require.NotNil(t, issue.Relations)
	require.Len(t, issue.Relations.Nodes, 2)
	blocks := issue.Relations.Nodes[0]
	require.Equal(t, "rel-1", blocks.ID)
	require.Equal(t, RelationTypeBlocks, blocks.Type)
	require.NotNil(t, blocks.Issue)
	require.Equal(t, "TEST-5", blocks.Issue.Identifier)
	require.NotNil(t, blocks.RelatedIssue)
	require.Equal(t, "TEST-9", blocks.RelatedIssue.Identifier)
	require.Equal(t, RelationTypeRelated, issue.Relations.Nodes[1].Type)
	require.Equal(t, "TEST-10", issue.Relations.Nodes[1].RelatedIssue.Identifier)

	// Incoming edges hydrate separately: here TEST-4 blocks this issue.
	require.NotNil(t, issue.InverseRelations)
	require.Len(t, issue.InverseRelations.Nodes, 1)
	inverse := issue.InverseRelations.Nodes[0]
	require.Equal(t, "rel-3", inverse.ID)
	require.Equal(t, RelationTypeBlocks, inverse.Type)
	require.NotNil(t, inverse.Issue)
	require.Equal(t, IssueRef{ID: "upstream-uuid", Identifier: "TEST-4", Title: "Upstream", URL: "https://linear.app/test/issue/TEST-4"}, *inverse.Issue)
	require.NotNil(t, inverse.RelatedIssue)
	require.Equal(t, "issue-uuid", inverse.RelatedIssue.ID)
}

func TestClient_ListIssues_WithoutLinks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// issue list stays light: per-issue connections would multiply the
		// cost of every page (see #18).
		for _, banned := range []string{"relations(", "inverseRelations(", "children("} {
			if strings.Contains(req.Query, banned) {
				t.Errorf("Expected list query not to contain %q, got query: %s", banned, req.Query)
			}
		}
		assertIssueProjectSelectsURL(t, req.Query)

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"issues": {
					"nodes": [{"id": "issue-1", "identifier": "TEST-1", "title": "One"}],
					"pageInfo": {"hasNextPage": false, "endCursor": ""}
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

	resp, err := client.ListIssues(context.Background(), ListIssuesOptions{})
	require.NoError(t, err)
	require.Len(t, resp.Issues.Nodes, 1)

	// Unselected connections stay nil so they are omitted from JSON rather
	// than reported as empty.
	issue := resp.Issues.Nodes[0]
	require.Nil(t, issue.Children)
	require.Nil(t, issue.Relations)
	require.Nil(t, issue.InverseRelations)
	out, err := json.Marshal(issue)
	require.NoError(t, err)
	for _, key := range []string{`"children"`, `"relations"`, `"inverseRelations"`} {
		require.NotContains(t, string(out), key)
	}
	require.Contains(t, string(out), `"parent":null`)
}
