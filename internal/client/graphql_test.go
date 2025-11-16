package client

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClient_Do_Success(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request headers
		if r.Header.Get("Authorization") != "test-api-key" {
			t.Errorf("Expected Authorization header to be 'test-api-key', got '%s'", r.Header.Get("Authorization"))
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("Expected Content-Type header to be 'application/json', got '%s'", r.Header.Get("Content-Type"))
		}

		// Return a successful GraphQL response
		response := graphQLResponse{
			Data: json.RawMessage(`{"test": "value"}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	var result map[string]string
	err := client.Do(context.Background(), "query { test }", nil, &result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if result["test"] != "value" {
		t.Errorf("Expected result['test'] to be 'value', got '%s'", result["test"])
	}
}

func TestClient_Do_GraphQLError(t *testing.T) {
	// Create a test server that returns a GraphQL error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Errors: []struct {
				Message string `json:"message"`
				Path    []any  `json:"path,omitempty"`
			}{
				{Message: "Field 'test' not found"},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	var result map[string]string
	err := client.Do(context.Background(), "query { test }", nil, &result)

	if err == nil {
		t.Error("Expected an error, got nil")
	}

	if err.Error() != "Field 'test' not found" {
		t.Errorf("Expected error message 'Field 'test' not found', got '%s'", err.Error())
	}
}

func TestClient_Do_HTTPError(t *testing.T) {
	// Create a test server that returns an HTTP error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Unauthorized"))
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	var result map[string]string
	err := client.Do(context.Background(), "query { test }", nil, &result)

	if err == nil {
		t.Error("Expected an error, got nil")
	}

	// The error message should contain the status code
	if err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

func TestClient_Do_InvalidJSON(t *testing.T) {
	// Create a test server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	var result map[string]string
	err := client.Do(context.Background(), "query { test }", nil, &result)

	if err == nil {
		t.Error("Expected an error, got nil")
	}
}

func TestClient_Do_WithVariables(t *testing.T) {
	// Create a test server that verifies variables are sent
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Fatalf("Failed to decode request body: %v", err)
		}

		// Verify variables are present
		if req.Variables == nil {
			t.Error("Expected variables to be present")
		}

		if req.Variables["key"] != "value" {
			t.Errorf("Expected variable 'key' to be 'value', got '%v'", req.Variables["key"])
		}

		// Return a successful response
		response := graphQLResponse{
			Data: json.RawMessage(`{"result": "ok"}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	variables := map[string]interface{}{
		"key": "value",
	}

	var result map[string]string
	err := client.Do(context.Background(), "query($key: String!) { test(key: $key) }", variables, &result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestClient_Do_ContextCancellation(t *testing.T) {
	// Create a test server that delays the response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		response := graphQLResponse{
			Data: json.RawMessage(`{"test": "value"}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	// Create a context that will be cancelled immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	var result map[string]string
	err := client.Do(ctx, "query { test }", nil, &result)

	if err == nil {
		t.Error("Expected an error due to context cancellation, got nil")
	}
}

func TestClient_Do_EmptyResponse(t *testing.T) {
	// Create a test server that returns an empty data response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	var result map[string]string
	err := client.Do(context.Background(), "query { test }", nil, &result)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
}

func TestClient_Do_NilResult(t *testing.T) {
	// Create a test server that returns a successful response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := graphQLResponse{
			Data: json.RawMessage(`{"test": "value"}`),
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		apiKey:     "test-api-key",
		endpoint:   server.URL,
	}

	// Pass nil as result - should not error
	err := client.Do(context.Background(), "query { test }", nil, nil)

	if err != nil {
		t.Errorf("Expected no error with nil result, got %v", err)
	}
}
