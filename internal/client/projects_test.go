package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListProjects(t *testing.T) {
	mockResp := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "proj-1",
						"name": "Project One",
					},
					{
						"id":   "proj-2",
						"name": "Project Two",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	ctx := context.Background()
	resp, err := client.ListProjects(ctx)
	if err != nil {
		t.Fatalf("ListProjects failed: %v", err)
	}

	if len(resp.Projects.Nodes) != 2 {
		t.Errorf("Expected 2 projects, got %d", len(resp.Projects.Nodes))
	}

	if resp.Projects.Nodes[0].ID != "proj-1" {
		t.Errorf("Expected first project ID to be 'proj-1', got '%s'", resp.Projects.Nodes[0].ID)
	}
}

func TestGetProjectsByTeam(t *testing.T) {
	mockResp := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "proj-1",
						"name": "Team Project",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	ctx := context.Background()
	resp, err := client.GetProjectsByTeam(ctx, "team-123")
	if err != nil {
		t.Fatalf("GetProjectsByTeam failed: %v", err)
	}

	if len(resp.Projects.Nodes) != 1 {
		t.Errorf("Expected 1 project, got %d", len(resp.Projects.Nodes))
	}

	if resp.Projects.Nodes[0].Name != "Team Project" {
		t.Errorf("Expected project name to be 'Team Project', got '%s'", resp.Projects.Nodes[0].Name)
	}
}

func TestGetProjectByIdentifier_UUID(t *testing.T) {
	mockResp := map[string]interface{}{
		"data": map[string]interface{}{
			"project": map[string]interface{}{
				"id":   "4e26961e-967f-458f-8fa2-4240035aa178",
				"name": "Test Project",
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	ctx := context.Background()
	project, err := client.GetProjectByIdentifier(ctx, "4e26961e-967f-458f-8fa2-4240035aa178", "")
	if err != nil {
		t.Fatalf("GetProjectByIdentifier failed: %v", err)
	}

	if project.ID != "4e26961e-967f-458f-8fa2-4240035aa178" {
		t.Errorf("Expected project ID to be '4e26961e-967f-458f-8fa2-4240035aa178', got '%s'", project.ID)
	}

	if project.Name != "Test Project" {
		t.Errorf("Expected project name to be 'Test Project', got '%s'", project.Name)
	}
}

func TestGetProjectByIdentifier_Name(t *testing.T) {
	mockResp := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{
					{
						"id":   "proj-123",
						"name": "Mobile App",
					},
				},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	ctx := context.Background()
	project, err := client.GetProjectByIdentifier(ctx, "Mobile App", "team-123")
	if err != nil {
		t.Fatalf("GetProjectByIdentifier failed: %v", err)
	}

	if project.ID != "proj-123" {
		t.Errorf("Expected project ID to be 'proj-123', got '%s'", project.ID)
	}

	if project.Name != "Mobile App" {
		t.Errorf("Expected project name to be 'Mobile App', got '%s'", project.Name)
	}
}

func TestGetProjectByIdentifier_NotFound(t *testing.T) {
	mockResp := map[string]interface{}{
		"data": map[string]interface{}{
			"projects": map[string]interface{}{
				"nodes": []map[string]interface{}{},
			},
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(mockResp)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	ctx := context.Background()
	_, err := client.GetProjectByIdentifier(ctx, "Nonexistent", "team-123")
	if err == nil {
		t.Error("Expected error for nonexistent project, got nil")
	}

	expectedMsg := "project not found: Nonexistent"
	if err.Error() != expectedMsg {
		t.Errorf("Expected error message '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestIsUUID(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"4e26961e-967f-458f-8fa2-4240035aa178", true},
		{"896bdb2d-89f6-43e0-a7e3-b95920249228", true},
		{"Mobile App", false},
		{"not-a-uuid", false},
		{"4e26961e967f458f8fa24240035aa178", false},    // no dashes
		{"4e26961e-967f-458f-8fa2-4240035aa17", false}, // too short
		{"", false},
	}

	for _, test := range tests {
		result := isUUID(test.input)
		if result != test.expected {
			t.Errorf("isUUID(%s) = %v, expected %v", test.input, result, test.expected)
		}
	}
}

func TestSelectProjectByIdentifier_ExactMatchPreferred(t *testing.T) {
	projects := []Project{
		{ID: "proj-1", Name: "Mobile Platform"},
		{ID: "proj-2", Name: "Mobile"},
	}

	project, err := selectProjectByIdentifier("mobile", projects)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if project.ID != "proj-2" {
		t.Errorf("Expected exact match project ID to be 'proj-2', got '%s'", project.ID)
	}
}

func TestSelectProjectByIdentifier_AmbiguousPartialMatch(t *testing.T) {
	projects := []Project{
		{ID: "proj-1", Name: "Mobile App"},
		{ID: "proj-2", Name: "Mobile Platform"},
	}

	_, err := selectProjectByIdentifier("Mobile", projects)
	if err == nil {
		t.Fatal("Expected ambiguity error, got nil")
	}
}
