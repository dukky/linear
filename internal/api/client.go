package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/linear-cli/linear/internal/config"
)

// Client represents a Linear API client
type Client struct {
	Config     *config.Config
	HTTPClient *http.Client
	Token      string
}

type graphQLRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables,omitempty"`
}

type graphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string `json:"message"`
		Path    []any  `json:"path"`
	} `json:"errors"`
}

// NewClient creates a new Linear API client
func NewClient(cfg *config.Config) (*Client, error) {
	token, err := cfg.LoadToken()
	if err != nil {
		return nil, err
	}

	return &Client{
		Config: cfg,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		Token: token.AccessToken,
	}, nil
}

// Query executes a GraphQL query
func (c *Client) Query(query string, variables map[string]interface{}, result interface{}) error {
	reqBody := graphQLRequest{
		Query:     query,
		Variables: variables,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("POST", config.LinearAPIURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	var gqlResp graphQLResponse
	if err := json.Unmarshal(body, &gqlResp); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	if len(gqlResp.Errors) > 0 {
		return fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if result != nil {
		if err := json.Unmarshal(gqlResp.Data, result); err != nil {
			return fmt.Errorf("failed to unmarshal data: %w", err)
		}
	}

	return nil
}

// GetViewer returns information about the authenticated user
func (c *Client) GetViewer() (*User, error) {
	query := `
		query {
			viewer {
				id
				name
				email
				admin
				createdAt
				displayName
			}
		}
	`

	var result struct {
		Viewer User `json:"viewer"`
	}

	if err := c.Query(query, nil, &result); err != nil {
		return nil, err
	}

	return &result.Viewer, nil
}

// ListIssues retrieves a list of issues
func (c *Client) ListIssues(first int, filter map[string]interface{}) (*IssueConnection, error) {
	query := `
		query($first: Int!, $filter: IssueFilter) {
			issues(first: $first, filter: $filter) {
				nodes {
					id
					identifier
					title
					description
					priority
					priorityLabel
					state {
						id
						name
						type
						color
					}
					assignee {
						id
						name
						displayName
					}
					team {
						id
						name
						key
					}
					createdAt
					updatedAt
					url
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	variables := map[string]interface{}{
		"first": first,
	}
	if filter != nil {
		variables["filter"] = filter
	}

	var result struct {
		Issues IssueConnection `json:"issues"`
	}

	if err := c.Query(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.Issues, nil
}

// GetIssue retrieves a single issue by ID
func (c *Client) GetIssue(id string) (*Issue, error) {
	query := `
		query($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				priority
				priorityLabel
				state {
					id
					name
					type
					color
				}
				assignee {
					id
					name
					displayName
				}
				team {
					id
					name
					key
				}
				createdAt
				updatedAt
				url
			}
		}
	`

	variables := map[string]interface{}{
		"id": id,
	}

	var result struct {
		Issue Issue `json:"issue"`
	}

	if err := c.Query(query, variables, &result); err != nil {
		return nil, err
	}

	return &result.Issue, nil
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(input IssueCreateInput) (*Issue, error) {
	query := `
		mutation($input: IssueCreateInput!) {
			issueCreate(input: $input) {
				success
				issue {
					id
					identifier
					title
					url
				}
			}
		}
	`

	variables := map[string]interface{}{
		"input": input,
	}

	var result struct {
		IssueCreate struct {
			Success bool  `json:"success"`
			Issue   Issue `json:"issue"`
		} `json:"issueCreate"`
	}

	if err := c.Query(query, variables, &result); err != nil {
		return nil, err
	}

	if !result.IssueCreate.Success {
		return nil, fmt.Errorf("failed to create issue")
	}

	return &result.IssueCreate.Issue, nil
}

// ListTeams retrieves all teams
func (c *Client) ListTeams() ([]Team, error) {
	query := `
		query {
			teams {
				nodes {
					id
					name
					key
					description
				}
			}
		}
	`

	var result struct {
		Teams struct {
			Nodes []Team `json:"nodes"`
		} `json:"teams"`
	}

	if err := c.Query(query, nil, &result); err != nil {
		return nil, err
	}

	return result.Teams.Nodes, nil
}
