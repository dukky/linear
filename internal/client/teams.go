package client

import "context"

// Team represents a Linear team
type Team struct {
	ID          string  `json:"id"`
	Key         string  `json:"key"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
}

// TeamsResponse is the response for listing teams
type TeamsResponse struct {
	Teams struct {
		Nodes []Team `json:"nodes"`
	} `json:"teams"`
}

// ListTeams retrieves all teams
func (c *Client) ListTeams(ctx context.Context) (*TeamsResponse, error) {
	query := `
		query {
			teams {
				nodes {
					id
					key
					name
					description
				}
			}
		}
	`

	var resp TeamsResponse
	if err := c.Do(ctx, query, nil, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetTeamByKey retrieves a team by its key
func (c *Client) GetTeamByKey(ctx context.Context, key string) (*TeamsResponse, error) {
	query := `
		query($key: String!) {
			teams(filter: { key: { eq: $key } }) {
				nodes {
					id
					key
					name
				}
			}
		}
	`

	vars := map[string]interface{}{
		"key": key,
	}

	var resp TeamsResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}
