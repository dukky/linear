package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/dukky/linear/internal/auth"
)

const linearAPIURL = "https://api.linear.app/graphql"

// Client is a simple GraphQL client for Linear
type Client struct {
	httpClient *http.Client
	apiKey     string
	endpoint   string
}

// NewClient creates a new Linear GraphQL client
func NewClient() (*Client, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, err
	}

	return &Client{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		endpoint:   linearAPIURL,
	}, nil
}

// graphQLRequest represents a GraphQL request
type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

// graphQLResponse represents a GraphQL response
type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Path    []any  `json:"path,omitempty"`
	} `json:"errors,omitempty"`
}

// Do executes a GraphQL query and unmarshals the response into result
func (c *Client) Do(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", c.endpoint, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
	}

	var gqlResp graphQLResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("%s", gqlResp.Errors[0].Message)
	}

	if result != nil && len(gqlResp.Data) > 0 {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}
