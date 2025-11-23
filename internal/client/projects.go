package client

import (
	"context"
	"fmt"
)

// ProjectsResponse is the response for listing projects
type ProjectsResponse struct {
	Projects struct {
		Nodes []Project `json:"nodes"`
	} `json:"projects"`
}

// ListProjects retrieves all projects
func (c *Client) ListProjects(ctx context.Context) (*ProjectsResponse, error) {
	query := `
		query {
			projects {
				nodes {
					id
					name
				}
			}
		}
	`

	var resp ProjectsResponse
	if err := c.Do(ctx, query, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetProjectsByTeam retrieves all projects for a given team
func (c *Client) GetProjectsByTeam(ctx context.Context, teamID string) (*ProjectsResponse, error) {
	query := `
		query($filter: ProjectFilter!) {
			projects(filter: $filter) {
				nodes {
					id
					name
				}
			}
		}
	`

	vars := map[string]interface{}{
		"filter": map[string]interface{}{
			"accessibleTeams": map[string]interface{}{
				"some": map[string]interface{}{
					"id": map[string]interface{}{
						"eq": teamID,
					},
				},
			},
		},
	}

	var resp ProjectsResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetProjectByIdentifier retrieves a project by name or UUID
// If teamID is provided, it filters projects by team to reduce ambiguity
// Returns the first matching project
func (c *Client) GetProjectByIdentifier(ctx context.Context, identifier string, teamID string) (*Project, error) {
	// Check if identifier looks like a UUID (basic check)
	if isUUID(identifier) {
		// If it's a UUID, query by ID
		query := `
			query($id: String!) {
				project(id: $id) {
					id
					name
				}
			}
		`

		vars := map[string]interface{}{
			"id": identifier,
		}

		var resp struct {
			Project *Project `json:"project"`
		}

		if err := c.Do(ctx, query, vars, &resp); err != nil {
			return nil, err
		}

		if resp.Project == nil {
			return nil, fmt.Errorf("project not found: %s", identifier)
		}

		return resp.Project, nil
	}

	// Otherwise, search by name
	query := `
		query($filter: ProjectFilter!) {
			projects(filter: $filter) {
				nodes {
					id
					name
				}
			}
		}
	`

	// Build filter with name matching
	filter := map[string]interface{}{
		"name": map[string]interface{}{
			"containsIgnoreCase": identifier,
		},
	}

	// If team ID is provided, filter by team as well
	if teamID != "" {
		filter["accessibleTeams"] = map[string]interface{}{
			"some": map[string]interface{}{
				"id": map[string]interface{}{
					"eq": teamID,
				},
			},
		}
	}

	vars := map[string]interface{}{
		"filter": filter,
	}

	var resp ProjectsResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	if len(resp.Projects.Nodes) == 0 {
		return nil, fmt.Errorf("project not found: %s", identifier)
	}

	// Return first match
	return &resp.Projects.Nodes[0], nil
}

// isUUID checks if a string looks like a UUID
func isUUID(s string) bool {
	// UUID format: 8-4-4-4-12 hex characters
	if len(s) != 36 {
		return false
	}
	for i, c := range s {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return false
			}
		} else {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
				return false
			}
		}
	}
	return true
}
