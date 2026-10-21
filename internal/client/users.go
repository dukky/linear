package client

import (
	"context"
	"errors"
)

type UsersResponse struct {
	Users struct {
		Nodes []User `json:"nodes"`
	} `json:"users"`
}

func (c *Client) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	query := `
		query($email: String!) {
			users(filter: { email: { eq: $email} }) {
				nodes {
					id
					email
					name
				}
			}
		}
	`

	vars := map[string]any{
		"email": email,
	}

	var userRsp UsersResponse

	err := c.Do(ctx, query, vars, &userRsp)
	if err != nil {
		return nil, err
	}

	if len(userRsp.Users.Nodes) == 0 {
		return nil, errors.New("no user found with the provided email")
	}

	return &userRsp.Users.Nodes[0], nil
}

// ViewerResponse is the response for the viewer query
type ViewerResponse struct {
	Viewer *User `json:"viewer"`
}

// GetViewer returns the user the API key authenticates as.
func (c *Client) GetViewer(ctx context.Context) (*User, error) {
	query := `
		query {
			viewer {
				id
				name
				email
			}
		}
	`

	var resp ViewerResponse
	if err := c.Do(ctx, query, nil, &resp); err != nil {
		return nil, err
	}

	if resp.Viewer == nil {
		return nil, errors.New("no viewer returned for the authenticated API key")
	}

	return resp.Viewer, nil
}
