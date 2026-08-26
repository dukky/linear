package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGetTeamStates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		if req.Variables["id"] != "team-1" {
			t.Errorf("Expected id 'team-1', got '%v'", req.Variables["id"])
		}

		response := graphQLResponse{
			Data: json.RawMessage(`{
				"team": {
					"id": "team-1",
					"states": {
						"nodes": [
							{"id": "state-1", "name": "Backlog", "type": "backlog"},
							{"id": "state-2", "name": "In Progress", "type": "started"},
							{"id": "state-3", "name": "Done", "type": "completed"}
						]
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	resp, err := client.GetTeamStates(context.Background(), "team-1")
	if err != nil {
		t.Fatalf("GetTeamStates failed: %v", err)
	}

	if resp.Team == nil {
		t.Fatal("Expected team to be non-nil")
	}

	if len(resp.Team.States.Nodes) != 3 {
		t.Errorf("Expected 3 states, got %d", len(resp.Team.States.Nodes))
	}

	if resp.Team.States.Nodes[1].Name != "In Progress" {
		t.Errorf("Expected second state name 'In Progress', got '%s'", resp.Team.States.Nodes[1].Name)
	}
}

func TestGetTeamStates_TeamNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{"team": null}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	resp, err := client.GetTeamStates(context.Background(), "nonexistent")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Team != nil {
		t.Error("Expected team to be nil")
	}
}

func TestGetStateByName_ExactMatch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"team": {
					"id": "team-1",
					"states": {
						"nodes": [
							{"id": "state-1", "name": "Backlog", "type": "backlog"},
							{"id": "state-2", "name": "In Progress", "type": "started"},
							{"id": "state-3", "name": "Done", "type": "completed"}
						]
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	state, err := client.GetStateByName(context.Background(), "team-1", "Done")
	if err != nil {
		t.Fatalf("GetStateByName failed: %v", err)
	}

	if state.ID != "state-3" {
		t.Errorf("Expected state ID 'state-3', got '%s'", state.ID)
	}
}

func TestGetStateByName_CaseInsensitive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"team": {
					"id": "team-1",
					"states": {
						"nodes": [
							{"id": "state-2", "name": "In Progress", "type": "started"}
						]
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	state, err := client.GetStateByName(context.Background(), "team-1", "in progress")
	if err != nil {
		t.Fatalf("GetStateByName failed: %v", err)
	}

	if state.ID != "state-2" {
		t.Errorf("Expected state ID 'state-2', got '%s'", state.ID)
	}

	if state.Name != "In Progress" {
		t.Errorf("Expected state name 'In Progress', got '%s'", state.Name)
	}
}

func TestGetStateByName_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"team": {
					"id": "team-1",
					"states": {
						"nodes": [
							{"id": "state-1", "name": "Backlog", "type": "backlog"},
							{"id": "state-2", "name": "In Progress", "type": "started"}
						]
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	_, err := client.GetStateByName(context.Background(), "team-1", "Nonexistent")
	if err == nil {
		t.Fatal("Expected error for nonexistent state, got nil")
	}

	expectedMsg := `state not found: "Nonexistent"; valid states for this team: Backlog, In Progress`
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGetStateByName_TeamNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{"team": null}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	_, err := client.GetStateByName(context.Background(), "nonexistent-team", "Done")
	if err == nil {
		t.Fatal("Expected error for nonexistent team, got nil")
	}

	expectedMsg := "team not found: nonexistent-team"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestSelectStateByName_CaseInsensitive(t *testing.T) {
	states := []WorkflowState{
		{ID: "state-1", Name: "Done", Type: "completed"},
	}

	state, err := selectStateByName("done", states)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if state.ID != "state-1" {
		t.Errorf("Expected state ID 'state-1', got '%s'", state.ID)
	}
}

func TestGetStateByName_TrimsWhitespace(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"team": {
					"id": "team-1",
					"states": {
						"nodes": [
							{"id": "state-3", "name": "Done", "type": "completed"}
						]
					}
				}
			}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	state, err := client.GetStateByName(context.Background(), "team-1", "  Done  ")
	if err != nil {
		t.Fatalf("GetStateByName failed: %v", err)
	}

	if state.ID != "state-3" {
		t.Errorf("Expected state ID 'state-3', got '%s'", state.ID)
	}
}
