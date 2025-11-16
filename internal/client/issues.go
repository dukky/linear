package client

import "context"

// Issue represents a Linear issue
type Issue struct {
	ID            string   `json:"id"`
	Identifier    string   `json:"identifier"`
	Title         string   `json:"title"`
	Description   *string  `json:"description"`
	Priority      int      `json:"priority"`
	PriorityLabel string   `json:"priorityLabel"`
	CreatedAt     string   `json:"createdAt"`
	UpdatedAt     string   `json:"updatedAt"`
	CompletedAt   *string  `json:"completedAt"`
	URL           string   `json:"url"`
	State         *State   `json:"state"`
	Assignee      *User    `json:"assignee"`
	Creator       *User    `json:"creator"`
	Team          *Team    `json:"team"`
	Project       *Project `json:"project"`
	Labels        struct {
		Nodes []Label `json:"nodes"`
	} `json:"labels"`
}

// State represents an issue state
type State struct {
	Name  string `json:"name"`
	Color string `json:"color"`
	Type  string `json:"type"`
}

// User represents a Linear user
type User struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Project represents a Linear project
type Project struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Label represents an issue label
type Label struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color"`
}

// PageInfo contains pagination information
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// IssuesResponse is the response for listing issues
type IssuesResponse struct {
	Issues struct {
		Nodes    []Issue  `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"issues"`
}

// IssueResponse is the response for getting a single issue
type IssueResponse struct {
	Issue *Issue `json:"issue"`
}

// ListIssues retrieves issues with optional team filter
// Automatically handles pagination to fetch all issues
func (c *Client) ListIssues(ctx context.Context, teamKey string) (*IssuesResponse, error) {
	query := `
		query($filter: IssueFilter, $after: String) {
			issues(filter: $filter, first: 100, after: $after) {
				nodes {
					id
					identifier
					title
					description
					priority
					priorityLabel
					createdAt
					updatedAt
					url
					state {
						name
						color
						type
					}
					assignee {
						id
						name
						email
					}
					team {
						id
						key
						name
					}
					project {
						id
						name
					}
					labels {
						nodes {
							id
							name
							color
						}
					}
				}
				pageInfo {
					hasNextPage
					endCursor
				}
			}
		}
	`

	var allIssues []Issue
	var cursor *string

	// Fetch all pages
	for {
		vars := make(map[string]interface{})

		if teamKey != "" {
			vars["filter"] = map[string]interface{}{
				"team": map[string]interface{}{
					"key": map[string]interface{}{
						"eq": teamKey,
					},
				},
			}
		}

		if cursor != nil {
			vars["after"] = *cursor
		}

		var resp IssuesResponse
		if err := c.Do(ctx, query, vars, &resp); err != nil {
			return nil, err
		}

		allIssues = append(allIssues, resp.Issues.Nodes...)

		if !resp.Issues.PageInfo.HasNextPage {
			break
		}
		cursor = &resp.Issues.PageInfo.EndCursor
	}

	// Return a consolidated response
	return &IssuesResponse{
		Issues: struct {
			Nodes    []Issue  `json:"nodes"`
			PageInfo PageInfo `json:"pageInfo"`
		}{
			Nodes: allIssues,
			PageInfo: PageInfo{
				HasNextPage: false,
				EndCursor:   "",
			},
		},
	}, nil
}

// GetIssue retrieves a single issue by ID or identifier
func (c *Client) GetIssue(ctx context.Context, id string) (*IssueResponse, error) {
	query := `
		query($id: String!) {
			issue(id: $id) {
				id
				identifier
				title
				description
				priority
				priorityLabel
				createdAt
				updatedAt
				completedAt
				url
				state {
					name
					color
					type
				}
				assignee {
					id
					name
					email
				}
				team {
					id
					key
					name
				}
				project {
					id
					name
				}
				labels {
					nodes {
						id
						name
						color
					}
				}
				creator {
					id
					name
					email
				}
			}
		}
	`

	vars := map[string]interface{}{
		"id": id,
	}

	var resp IssueResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// CreateIssueInput represents the input for creating an issue
type CreateIssueInput struct {
	Title         string   `json:"title"`
	Description   string   `json:"description,omitempty"`
	TeamID        string   `json:"teamId"`
	LabelIds      []string `json:"labelIds,omitempty"`
	SubscriberIds []string `json:"subscriberIds,omitempty"`
}

// CreateIssueResponse is the response for creating an issue
type CreateIssueResponse struct {
	IssueCreate struct {
		Success bool `json:"success"`
		Issue   *struct {
			ID         string `json:"id"`
			Identifier string `json:"identifier"`
			Title      string `json:"title"`
			URL        string `json:"url"`
		} `json:"issue"`
	} `json:"issueCreate"`
}

// CreateIssue creates a new issue
func (c *Client) CreateIssue(ctx context.Context, input CreateIssueInput) (*CreateIssueResponse, error) {
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

	vars := map[string]interface{}{
		"input": input,
	}

	var resp CreateIssueResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
