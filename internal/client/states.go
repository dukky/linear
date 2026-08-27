package client

import (
	"context"
	"fmt"
	"strings"
)

// WorkflowState represents a Linear workflow state (issue status), scoped to a team.
type WorkflowState struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// TeamStatesResponse is the response for fetching a team's workflow states
type TeamStatesResponse struct {
	Team *struct {
		States struct {
			Nodes []WorkflowState `json:"nodes"`
		} `json:"states"`
	} `json:"team"`
}

// GetTeamStates retrieves the workflow states available to a team, identified by team UUID.
func (c *Client) GetTeamStates(ctx context.Context, teamID string) (*TeamStatesResponse, error) {
	query := `
		query($id: String!) {
			team(id: $id) {
				id
				states {
					nodes {
						id
						name
						type
					}
				}
			}
		}
	`

	vars := map[string]interface{}{
		"id": teamID,
	}

	var resp TeamStatesResponse
	if err := c.Do(ctx, query, vars, &resp); err != nil {
		return nil, err
	}

	return &resp, nil
}

// GetStateByName resolves a workflow state name to its ID, scoped to the given team.
// State names are team-scoped in Linear, so the lookup is always constrained to teamID.
// Matching is case-insensitive. If no state matches, the error lists the team's valid
// state names so the caller can correct themselves.
func (c *Client) GetStateByName(ctx context.Context, teamID string, name string) (*WorkflowState, error) {
	name = strings.TrimSpace(name)

	resp, err := c.GetTeamStates(ctx, teamID)
	if err != nil {
		return nil, err
	}

	if resp.Team == nil {
		return nil, fmt.Errorf("team not found: %s", teamID)
	}

	return selectStateByName(name, resp.Team.States.Nodes)
}

func selectStateByName(name string, states []WorkflowState) (*WorkflowState, error) {
	for i := range states {
		if strings.EqualFold(states[i].Name, name) {
			return &states[i], nil
		}
	}

	return nil, fmt.Errorf("state not found: %q; valid states for this team: %s", name, stateNames(states))
}

func stateNames(states []WorkflowState) string {
	names := make([]string, 0, len(states))
	for _, s := range states {
		names = append(names, s.Name)
	}
	return strings.Join(names, ", ")
}
