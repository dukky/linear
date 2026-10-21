package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dukky/linear/internal/auth"
)

const linearAPIURL = "https://api.linear.app/graphql"

// DefaultRequestTimeout bounds each individual GraphQL request made by Do.
// Paginated fetches issue many requests, each with its own budget, so a
// large result set is never cut off by a single overall deadline.
const DefaultRequestTimeout = 30 * time.Second

// requestTimeout is the per-request timeout applied by Do. It is a variable
// so tests can shorten it.
var requestTimeout = DefaultRequestTimeout

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
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:   apiKey,
		endpoint: linearAPIURL,
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

	// Each request gets its own deadline. A caller-supplied deadline still
	// applies because the child context inherits it.
	reqCtx, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, "POST", c.endpoint, bytes.NewReader(jsonBody))
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
