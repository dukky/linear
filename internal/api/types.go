package api

import "time"

// User represents a Linear user
type User struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Email       string    `json:"email"`
	Admin       bool      `json:"admin"`
	CreatedAt   time.Time `json:"createdAt"`
	DisplayName string    `json:"displayName"`
}

// Issue represents a Linear issue
type Issue struct {
	ID            string    `json:"id"`
	Identifier    string    `json:"identifier"`
	Title         string    `json:"title"`
	Description   string    `json:"description,omitempty"`
	Priority      int       `json:"priority"`
	PriorityLabel string    `json:"priorityLabel"`
	State         *State    `json:"state,omitempty"`
	Assignee      *User     `json:"assignee,omitempty"`
	Team          *Team     `json:"team,omitempty"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	URL           string    `json:"url"`
}

// IssueConnection represents a paginated list of issues
type IssueConnection struct {
	Nodes    []Issue  `json:"nodes"`
	PageInfo PageInfo `json:"pageInfo"`
}

// PageInfo contains pagination information
type PageInfo struct {
	HasNextPage bool   `json:"hasNextPage"`
	EndCursor   string `json:"endCursor"`
}

// State represents an issue state
type State struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Type  string `json:"type"`
	Color string `json:"color"`
}

// Team represents a Linear team
type Team struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Key         string `json:"key"`
	Description string `json:"description,omitempty"`
}

// IssueCreateInput represents the input for creating an issue
type IssueCreateInput struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	TeamID      string `json:"teamId"`
	AssigneeID  string `json:"assigneeId,omitempty"`
	Priority    int    `json:"priority,omitempty"`
	StateID     string `json:"stateId,omitempty"`
}
