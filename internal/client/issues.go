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

// IssuesResponse is the response for listing issues
type IssuesResponse struct {
	Issues struct {
		Nodes    []Issue  `json:"nodes"`
		PageInfo PageInfo `json:"pageInfo"`
	} `json:"issues"`
}

// PageInfo contains pagination information
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// IssueResponse is the response for getting a single issue
type IssueResponse struct {
	Issue *Issue `json:"issue"`
}

// ListIssuesOptions contains options for listing issues
type ListIssuesOptions struct {
	TeamKey string
	Limit   int
	After   string
}

// ListIssues retrieves issues with optional team filter and pagination
func (c *Client) ListIssues(ctx context.Context, opts ListIssuesOptions) (*IssuesResponse, error) {
	query := `
		query($filter: IssueFilter, $first: Int!, $after: String) {
			issues(filter: $filter, first: $first, after: $after) {
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

	// Default limit to 50 if not specified
	limit := opts.Limit
	if limit <= 0 {
		limit = 50
	}

	vars := map[string]interface{}{
		"first": limit,
	}

	if opts.After != "" {
		vars["after"] = opts.After
	}

	if opts.TeamKey != "" {
		vars["filter"] = map[string]interface{}{
			"team": map[string]interface{}{
				"key": map[string]interface{}{
					"eq": opts.TeamKey,
				},
			},
		}
	}

	var resp IssuesResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// ListAllIssues retrieves all issues using cursor-based pagination
func (c *Client) ListAllIssues(ctx context.Context, teamKey string) ([]Issue, error) {
	var allIssues []Issue
	opts := ListIssuesOptions{
		TeamKey: teamKey,
		Limit:   100, // Use larger page size for efficiency
	}

	for {
		resp, err := c.ListIssues(ctx, opts)
		if err != nil {
			return nil, err
		}

		allIssues = append(allIssues, resp.Issues.Nodes...)

		if !resp.Issues.PageInfo.HasNextPage {
			break
		}

		opts.After = resp.Issues.PageInfo.EndCursor
	}

	return allIssues, nil
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
	ProjectID     string   `json:"projectId,omitempty"`
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
