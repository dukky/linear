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

func TestGetUserByEmail(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the email variable
		if req.Variables["email"] != "test@example.com" {
			t.Errorf("expected email test@example.com, got %v", req.Variables["email"])
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"users": {
					"nodes": [
						{
							"id": "user-123",
							"name": "Test User",
							"email": "test@example.com"
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

	resp, err := client.GetUserByEmail(context.Background(), "test@example.com")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp.Email == "" {
		t.Fatal("Expected email to be non-empty")
	}

	if resp.ID != "user-123" {
		t.Errorf("Expected identifier 'user-123', got '%s'", resp.ID)
	}
}

func TestGetUserByEmail_NotFound(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Decode the request
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request: %v", err)
		}

		// Verify the email variable
		if req.Variables["email"] != "test@example.com" {
			t.Errorf("expected email test@example.com, got %v", req.Variables["email"])
		}

		// Return a mock response
		response := graphQLResponse{
			Data: json.RawMessage(`{
				"users": {
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

	_, err := client.GetUserByEmail(context.Background(), "test@example.com")
	require.Error(t, err)

	require.EqualError(t, err, "no user found with the provided email")
}

func TestClient_GetViewer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("Failed to decode request: %v", err)
			return
		}

		compact := strings.Join(strings.Fields(req.Query), " ")
		if !strings.Contains(compact, "viewer { id name email }") {
			t.Errorf("Expected query to select viewer { id name email }, got: %s", req.Query)
		}

		json.NewEncoder(w).Encode(graphQLResponse{
			Data: json.RawMessage(`{"viewer": {"id": "user-me", "name": "Me Myself", "email": "me@example.com"}}`),
		})
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	user, err := client.GetViewer(context.Background())
	require.NoError(t, err)
	require.Equal(t, &User{ID: "user-me", Name: "Me Myself", Email: "me@example.com"}, user)
}

func TestClient_GetViewer_NilViewer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(graphQLResponse{Data: json.RawMessage(`{"viewer": null}`)})
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-key",
		endpoint:   server.URL,
	}

	_, err := client.GetViewer(context.Background())
	require.Error(t, err)
}
