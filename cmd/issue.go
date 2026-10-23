package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dukky/linear/internal/client"
	"github.com/dukky/linear/internal/output"
	"github.com/spf13/cobra"
)

var (
	teamFilter             string
	projectFilter          string
	issueTitle             string
	issueDesc              string
	issueTeamID            string
	issueProjectIdentifier string
	issueAssignee          string
	issueState             string
	issuePriority          int
	issueParent            string
	issueUpdateTitle       string
	issueUpdateDesc        string
	issueUpdatePriority    int
	issueUpdateProject     string
	issueUpdateAssignee    string
	issueUpdateState       string
	issueUpdateParent      string
	issueUpdateBlocks      []string
	issueUpdateBlockedBy   []string
	issueUpdateRelated     []string
	issueLimit             int
	fetchAll               bool
	listAssigneeFilter     string
	listMineFilter         bool
	listStateFilter        string
	listStateTypeFilter    []string
)

var issueCmd = &cobra.Command{
	Use:   "issue",
	Short: "Manage issues",
	Long:  "List, view, and create Linear issues",
}

var issueListCmd = &cobra.Command{
	Use:   "list",
	Short: "List issues",
	Long: `List issues in your Linear workspace.

Use --team to filter by team key (e.g., --team ENG).
Use --project to filter by project name or ID.
Use --assignee to filter by assignee email, or --assignee me / --mine for
issues assigned to the authenticated user.
Use --state to filter by workflow state name (case-insensitive, e.g. "In Progress").
Use --state-type to filter by workflow state type; accepts comma-separated
values or repetition. Valid types: triage, backlog, unstarted, started,
completed, canceled, duplicate.
Use --limit to specify the maximum number of issues to fetch (default: 50).
Use --all to fetch all issues using pagination.

Both --limit and --all page through results at 100 issues per request. Each
request gets its own 30s timeout and there is no overall deadline, so large
teams and limits above Linear's 250-per-page cap work.

Examples:
  linear issue list --team ENG
  linear issue list --mine --state-type started,unstarted
  linear issue list --assignee dev@example.com --team ENG`,
	Run: func(cmd *cobra.Command, args []string) {
		// Validate filter flags before any network access.
		assigneeIsMe, assigneeEmail, err := resolveListAssignee(listAssigneeFilter, listMineFilter)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := client.ValidateStateTypes(listStateTypeFilter); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		// No overall deadline: Do() bounds every individual request with
		// client.DefaultRequestTimeout, so paginated fetches of large teams
		// are not cut off part-way through.
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		// Resolve project filter if specified
		var projectID string
		if projectFilter != "" {
			// Get team ID for scoping project lookup (if team filter provided)
			var teamID string
			if teamFilter != "" {
				teamResp, err := c.GetTeamByKey(ctx, teamFilter)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Error fetching team: %v\n", err)
					os.Exit(1)
				}
				if len(teamResp.Teams.Nodes) == 0 {
					fmt.Fprintf(os.Stderr, "Team not found: %s\n", teamFilter)
					os.Exit(1)
				}
				teamID = teamResp.Teams.Nodes[0].ID
			}

			project, err := c.GetProjectByIdentifier(ctx, projectFilter, teamID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching project: %v\n", err)
				fmt.Fprintln(os.Stderr, "Tip: Run 'linear project list' to see available projects")
				os.Exit(1)
			}
			projectID = project.ID
		}

		opts := client.ListIssuesOptions{
			TeamKey:       teamFilter,
			ProjectID:     projectID,
			AssigneeEmail: assigneeEmail,
			AssigneeIsMe:  assigneeIsMe,
			StateName:     listStateFilter,
			StateTypes:    listStateTypeFilter,
		}

		if !fetchAll {
			// --limit means "at most N issues"; ListAllIssues pages at 100
			// per request so large limits never become one oversized request.
			limit := issueLimit
			if limit <= 0 {
				limit = 50 // preserve the previous default for --limit <= 0
			}
			opts.MaxIssues = limit
		}

		issues, err := c.ListAllIssues(ctx, opts)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			if err := output.PrintJSON(issues); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Table output
		table := output.NewTable([]string{"ID", "TITLE", "STATUS", "ASSIGNEE", "PRIORITY"})
		for _, issue := range issues {
			assignee := "-"
			if issue.Assignee != nil {
				assignee = issue.Assignee.Name
			}

			priority := "-"
			if issue.PriorityLabel != "" {
				priority = issue.PriorityLabel
			}

			status := "-"
			if issue.State != nil {
				status = issue.State.Name
			}

			table.AddRow([]string{
				issue.Identifier,
				output.TruncateString(issue.Title, 50),
				status,
				assignee,
				priority,
			})
		}
		table.Print()
	},
}

// resolveListAssignee interprets the issue list --assignee and --mine flags.
// It returns isMe for the authenticated user, or a non-empty email.
func resolveListAssignee(assignee string, mine bool) (isMe bool, email string, err error) {
	assignee = strings.TrimSpace(assignee)
	if assignee == "" {
		return mine, "", nil
	}
	if strings.EqualFold(assignee, "me") {
		return true, "", nil
	}
	if mine {
		return false, "", fmt.Errorf("use either --mine or --assignee, not both")
	}
	if !strings.Contains(assignee, "@") {
		return false, "", fmt.Errorf("assignee must be an email address or 'me', got %q", assignee)
	}
	return false, assignee, nil
}

var issueViewCmd = &cobra.Command{
	Use:   "view <issue-id>",
	Short: "View issue details",
	Long: `View detailed information about a specific issue.

Examples:
  linear issue view ENG-123
  linear issue view <issue-uuid>`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueID := args[0]

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := c.GetIssue(ctx, issueID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching issue: %v\n", err)
			os.Exit(1)
		}

		if resp.Issue == nil {
			fmt.Fprintf(os.Stderr, "Issue not found: %s\n", issueID)
			os.Exit(1)
		}

		issue := resp.Issue

		if jsonOutput {
			if err := output.PrintJSON(issue); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Human-readable output
		fmt.Printf("ID:          %s\n", issue.Identifier)
		fmt.Printf("Title:       %s\n", issue.Title)

		if issue.State != nil {
			fmt.Printf("Status:      %s\n", issue.State.Name)
		}

		if issue.Assignee != nil {
			fmt.Printf("Assignee:    %s\n", issue.Assignee.Name)
		}

		if issue.PriorityLabel != "" {
			fmt.Printf("Priority:    %s\n", issue.PriorityLabel)
		}

		if issue.Team != nil {
			fmt.Printf("Team:        %s (%s)\n", issue.Team.Name, issue.Team.Key)
		}

		if issue.Project != nil {
			fmt.Printf("Project:     %s\n", issue.Project.Name)
		}

		if issue.Creator != nil {
			fmt.Printf("Creator:     %s\n", issue.Creator.Name)
		}

		fmt.Printf("Created:     %s\n", issue.CreatedAt)
		fmt.Printf("Updated:     %s\n", issue.UpdatedAt)

		if issue.CompletedAt != nil && *issue.CompletedAt != "" {
			fmt.Printf("Completed:   %s\n", *issue.CompletedAt)
		}

		fmt.Printf("URL:         %s\n", issue.URL)

		if issue.Description != nil && *issue.Description != "" {
			fmt.Printf("\nDescription:\n%s\n", *issue.Description)
		}

		if len(issue.Labels.Nodes) > 0 {
			fmt.Printf("\nLabels:\n")
			for _, label := range issue.Labels.Nodes {
				fmt.Printf("  - %s\n", label.Name)
			}
		}

		relations := formatRelations(issue.Relations, issue.InverseRelations)

		var children []string
		if issue.Children != nil {
			for _, child := range issue.Children.Nodes {
				children = append(children, child.Identifier+" "+child.Title)
			}
		}

		if issue.Parent != nil || len(children) > 0 || len(relations) > 0 {
			fmt.Println()
		}

		if issue.Parent != nil {
			fmt.Printf("Parent:      %s %s\n", issue.Parent.Identifier, issue.Parent.Title)
		}

		if len(children) > 0 {
			fmt.Printf("Sub-issues:  %s\n", strings.Join(children, ", "))
		}

		if len(relations) > 0 {
			fmt.Printf("Relations:   %s\n", strings.Join(relations, ", "))
		}
	},
}

// formatRelations renders an issue's relation edges for issue view text
// output, outgoing first. Each entry names the other endpoint: relatedIssue
// for an outgoing edge (from Issue.relations), issue for an incoming edge
// (from Issue.inverseRelations). A blocks edge reads "blocks X" outgoing and
// "blocked by X" incoming; related is symmetric and reads "related X" either
// way. Other relation types read "<type> X" outgoing and "<type> from X"
// incoming. An edge whose ID has already been rendered is skipped, so the
// same edge is never printed twice. A nil connection (not selected) renders
// as nothing.
func formatRelations(outgoing, incoming *client.IssueRelationConnection) []string {
	var entries []string
	seen := map[string]bool{}

	identifier := func(ref *client.IssueRef) string {
		if ref == nil || ref.Identifier == "" {
			return "?"
		}
		return ref.Identifier
	}

	add := func(rel client.IssueRelation, entry string) {
		if rel.ID != "" {
			if seen[rel.ID] {
				return
			}
			seen[rel.ID] = true
		}
		entries = append(entries, entry)
	}

	if outgoing != nil {
		for _, rel := range outgoing.Nodes {
			add(rel, rel.Type+" "+identifier(rel.RelatedIssue))
		}
	}

	if incoming == nil {
		return entries
	}

	for _, rel := range incoming.Nodes {
		other := identifier(rel.Issue)
		switch rel.Type {
		case client.RelationTypeBlocks:
			add(rel, "blocked by "+other)
		case client.RelationTypeRelated:
			add(rel, "related "+other)
		default:
			add(rel, rel.Type+" from "+other)
		}
	}

	return entries
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long: `Create a new issue in Linear.

Examples:
  linear issue create --team ENG --title "Fix bug" --description "Bug details"
  linear issue create --team ENG --title "New feature" --project "Mobile App"
  linear issue create --team ENG --title "Task" --project "4e26961e-967f-458f-8fa2-4240035aa178"
  linear issue create --team ENG --title "New task" --state "In Progress"
  linear issue create --team ENG --title "New task" --priority 2
  linear issue create --team ENG --title "Sub-task" --parent ENG-123`,
	Run: func(cmd *cobra.Command, args []string) {
		if issueTitle == "" {
			fmt.Fprintln(os.Stderr, "Error: --title is required")
			os.Exit(1)
		}

		if issueTeamID == "" {
			fmt.Fprintln(os.Stderr, "Error: --team is required")
			os.Exit(1)
		}

		priorityChanged := cmd.Flags().Changed("priority")

		if cmd.Flags().Changed("parent") && issueParent == "" {
			fmt.Fprintln(os.Stderr, "Error: --parent cannot be empty")
			os.Exit(1)
		}

		if priorityChanged && (issuePriority < 0 || issuePriority > 4) {
			fmt.Fprintln(os.Stderr, "Error: --priority must be between 0 and 4")
			os.Exit(1)
		}

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Get team by key to get the team ID
		teamResp, err := c.GetTeamByKey(ctx, issueTeamID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching team: %v\n", err)
			os.Exit(1)
		}
		if len(teamResp.Teams.Nodes) == 0 {
			fmt.Fprintf(os.Stderr, "Team not found: %s\n", issueTeamID)
			os.Exit(1)
		}

		teamID := teamResp.Teams.Nodes[0].ID

		// Resolve project if specified
		var projectID string
		if issueProjectIdentifier != "" {
			project, err := c.GetProjectByIdentifier(ctx, issueProjectIdentifier, teamID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching project: %v\n", err)
				fmt.Fprintf(os.Stderr, "Tip: Run 'linear project list --team %s' to see available projects\n", issueTeamID)
				os.Exit(1)
			}
			projectID = project.ID
		}

		// Resolve state if specified (state names are team-scoped)
		var stateID string
		if issueState != "" {
			state, err := c.GetStateByName(ctx, teamID, issueState)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching state: %v\n", err)
				os.Exit(1)
			}
			stateID = state.ID
		}

		// Create the issue
		input := client.CreateIssueInput{
			Title:  issueTitle,
			TeamID: teamID,
		}

		// Resolve parent if specified: parentId must be a UUID, so an
		// identifier such as ENG-123 is looked up first.
		var parent *client.Issue
		if issueParent != "" {
			parentResp, err := c.GetIssue(ctx, issueParent)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching parent issue: %v\n", err)
				os.Exit(1)
			}
			if parentResp.Issue == nil {
				fmt.Fprintf(os.Stderr, "Parent issue not found: %s\n", issueParent)
				os.Exit(1)
			}
			input.ParentID = parentResp.Issue.ID
			parent = parentResp.Issue
		}

		if issueDesc != "" {
			input.Description = issueDesc
		}

		if projectID != "" {
			input.ProjectID = projectID
		}

		if stateID != "" {
			input.StateID = stateID
		}

		if priorityChanged {
			input.Priority = &issuePriority
		}

		if issueAssignee != "" {
			user, err := c.GetUserByEmail(ctx, issueAssignee)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching user by email: %v\n", err)
				os.Exit(1)
			}
			input.AssigneeID = user.ID
		}

		resp, err := c.CreateIssue(ctx, input)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating issue: %v\n", err)
			os.Exit(1)
		}

		if !resp.IssueCreate.Success {
			fmt.Fprintln(os.Stderr, "Error: Failed to create issue")
			os.Exit(1)
		}

		if resp.IssueCreate.Issue == nil {
			fmt.Fprintln(os.Stderr, "Error: Issue was created but no details returned")
			os.Exit(1)
		}

		issue := resp.IssueCreate.Issue

		if jsonOutput {
			if err := output.PrintJSON(issue); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Human-readable output
		fmt.Printf("Issue created successfully!\n")
		fmt.Printf("ID:    %s\n", issue.Identifier)
		fmt.Printf("Title: %s\n", issue.Title)
		fmt.Printf("URL:   %s\n", issue.URL)

		if parent != nil {
			fmt.Printf("Parent: %s %s\n", parent.Identifier, parent.Title)
		}
	},
}

var issueUpdateCmd = &cobra.Command{
	Use:   "update <issue-id>",
	Short: "Update an existing issue",
	Long: `Update fields on an existing issue.

Examples:
  linear issue update ENG-123 --title "Updated title"
  linear issue update ENG-123 --description "New details"
  linear issue update ENG-123 --priority 1
  linear issue update ENG-123 --project "Mobile App"
  linear issue update ENG-123 --assignee "user@example.com"
  linear issue update ENG-123 --state "Done"
  linear issue update ENG-123 --parent ENG-100
  linear issue update ENG-123 --blocks ENG-124 --blocks ENG-125
  linear issue update ENG-123 --blocked-by ENG-120 --related ENG-130

--blocks, --blocked-by and --related are repeatable; each value is a single
issue (values are not split on commas).`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if err := runIssueUpdate(cmd, args); err != nil {
			fmt.Fprintf(os.Stderr, "%v\n", err)
			os.Exit(1)
		}
	},
}

// runIssueUpdate implements issue update. Every failure is returned as an
// error whose text is exactly what the command prints to stderr, so tests can
// assert on it without the process exiting.
func runIssueUpdate(cmd *cobra.Command, args []string) error {
	issueID := args[0]

	titleChanged := cmd.Flags().Changed("title")
	descriptionChanged := cmd.Flags().Changed("description")
	priorityChanged := cmd.Flags().Changed("priority")
	projectChanged := cmd.Flags().Changed("project")
	assigneeChanged := cmd.Flags().Changed("assignee")
	stateChanged := cmd.Flags().Changed("state")
	parentChanged := cmd.Flags().Changed("parent")
	blocksChanged := len(issueUpdateBlocks) > 0
	blockedByChanged := len(issueUpdateBlockedBy) > 0
	relatedChanged := len(issueUpdateRelated) > 0
	relationsChanged := blocksChanged || blockedByChanged || relatedChanged

	if !titleChanged && !descriptionChanged && !priorityChanged && !projectChanged && !assigneeChanged && !stateChanged && !parentChanged && !blocksChanged && !blockedByChanged && !relatedChanged {
		return errors.New("Error: specify at least one field to update (--title, --description, --priority, --project, --assignee, --state, --parent, --blocks, --blocked-by, --related)")
	}

	if titleChanged && issueUpdateTitle == "" {
		return errors.New("Error: --title cannot be empty")
	}

	if parentChanged && issueUpdateParent == "" {
		return errors.New("Error: --parent cannot be empty")
	}

	for _, relationFlag := range []struct {
		name   string
		values []string
	}{
		{"--blocks", issueUpdateBlocks},
		{"--blocked-by", issueUpdateBlockedBy},
		{"--related", issueUpdateRelated},
	} {
		for _, value := range relationFlag.values {
			if value == "" {
				return fmt.Errorf("Error: %s values cannot be empty", relationFlag.name)
			}
		}
	}

	if projectChanged && issueUpdateProject == "" {
		return errors.New("Error: --project cannot be empty")
	}

	if stateChanged && issueUpdateState == "" {
		return errors.New("Error: --state cannot be empty")
	}

	if priorityChanged && (issueUpdatePriority < 0 || issueUpdatePriority > 4) {
		return errors.New("Error: --priority must be between 0 and 4")
	}

	c, err := client.NewClient()
	if err != nil {
		return fmt.Errorf("Error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	input := client.UpdateIssueInput{}

	if titleChanged {
		input.Title = &issueUpdateTitle
	}

	if descriptionChanged {
		input.Description = &issueUpdateDesc
	}

	if priorityChanged {
		input.Priority = &issueUpdatePriority
	}

	if assigneeChanged {
		if issueUpdateAssignee == "" {
			return errors.New("Error: --assignee must not be empty")
		}
		user, err := c.GetUserByEmail(ctx, issueUpdateAssignee)
		if err != nil {
			return fmt.Errorf("Error fetching user by email: %v", err)
		}
		input.AssigneeID = &user.ID
	}

	// Project and state resolution are both scoped to the issue's own team
	// (which may differ from any --team used elsewhere), and relations need
	// the issue's UUID as an endpoint, so fetch the issue once up front and
	// reuse it for all three.
	var fetchedIssue *client.Issue
	if projectChanged || stateChanged || relationsChanged {
		issueResp, err := c.GetIssue(ctx, issueID)
		if err != nil {
			return fmt.Errorf("Error fetching issue: %v", err)
		}
		if issueResp.Issue == nil {
			return fmt.Errorf("Issue not found: %s", issueID)
		}
		fetchedIssue = issueResp.Issue

		teamID := ""
		if issueResp.Issue.Team != nil {
			teamID = issueResp.Issue.Team.ID
		}

		if projectChanged {
			project, err := c.GetProjectByIdentifier(ctx, issueUpdateProject, teamID)
			if err != nil {
				return fmt.Errorf("Error fetching project: %v\nTip: Run 'linear project list' to see available projects", err)
			}

			input.ProjectID = &project.ID
		}

		if stateChanged {
			if teamID == "" {
				return errors.New("Error: could not determine issue's team to resolve state")
			}

			state, err := c.GetStateByName(ctx, teamID, issueUpdateState)
			if err != nil {
				return fmt.Errorf("Error fetching state: %v", err)
			}

			input.StateID = &state.ID
		}
	}

	// parentId must be a UUID, so resolve identifiers such as ENG-123.
	var parent *client.Issue
	if parentChanged {
		parentResp, err := c.GetIssue(ctx, issueUpdateParent)
		if err != nil {
			return fmt.Errorf("Error fetching parent issue: %v", err)
		}
		if parentResp.Issue == nil {
			return fmt.Errorf("Parent issue not found: %s", issueUpdateParent)
		}
		parent = parentResp.Issue
		input.ParentID = &parent.ID
	}

	// Resolve every relation target before any write, so a mistyped
	// identifier fails the command without leaving a partial update.
	var relationRequests []relationRequest
	if relationsChanged {
		resolve := func(id string) (string, error) {
			resp, err := c.GetIssue(ctx, id)
			if err != nil {
				return "", err
			}
			if resp.Issue == nil {
				return "", fmt.Errorf("issue not found: %s", id)
			}
			return resp.Issue.ID, nil
		}
		relationRequests, err = buildRelationInputs(fetchedIssue.ID, issueUpdateBlocks, issueUpdateBlockedBy, issueUpdateRelated, resolve)
		if err != nil {
			return fmt.Errorf("Error: %v", err)
		}
	}

	// issue is what the text output describes. A relations-only update
	// sends no issueUpdate mutation, so it falls back to the issue fetched
	// above (identifier, title and URL are unaffected by relations).
	issue := fetchedIssue

	fieldsChanged := titleChanged || descriptionChanged || priorityChanged || projectChanged || assigneeChanged || stateChanged || parentChanged
	if fieldsChanged {
		resp, err := c.UpdateIssue(ctx, issueID, input)
		if err != nil {
			return fmt.Errorf("Error updating issue: %v", err)
		}

		if !resp.IssueUpdate.Success {
			return errors.New("Error: Failed to update issue")
		}

		if resp.IssueUpdate.Issue == nil {
			return errors.New("Error: Issue was updated but no details returned")
		}

		issue = resp.IssueUpdate.Issue
	}

	var created []*client.IssueRelation
	for _, req := range relationRequests {
		// Errors name the issue as the user typed it (req.Target), not the
		// resolved UUID.
		relResp, err := c.CreateIssueRelation(ctx, req.Input)
		if err != nil {
			return fmt.Errorf("Error creating %s relation with %s: %v", req.Input.Type, req.Target, err)
		}
		if !relResp.IssueRelationCreate.Success || relResp.IssueRelationCreate.IssueRelation == nil {
			return fmt.Errorf("Error: Failed to create %s relation with %s", req.Input.Type, req.Target)
		}
		created = append(created, relResp.IssueRelationCreate.IssueRelation)
	}

	if jsonOutput {
		// The update result predates any relations created above, so
		// re-fetch to print an issue that includes them.
		if len(created) > 0 {
			issueResp, err := c.GetIssue(ctx, issueID)
			if err != nil {
				return fmt.Errorf("Error fetching updated issue: %v", err)
			}
			if issueResp.Issue == nil {
				return fmt.Errorf("Issue not found: %s", issueID)
			}
			issue = issueResp.Issue
		}

		if err := output.PrintJSON(issue); err != nil {
			return fmt.Errorf("Error formatting output: %v", err)
		}
		return nil
	}

	fmt.Printf("Issue updated successfully!\n")
	fmt.Printf("ID:    %s\n", issue.Identifier)
	fmt.Printf("Title: %s\n", issue.Title)
	fmt.Printf("URL:   %s\n", issue.URL)

	for _, rel := range created {
		from, to := "?", "?"
		if rel.Issue != nil {
			from = rel.Issue.Identifier
		}
		if rel.RelatedIssue != nil {
			to = rel.RelatedIssue.Identifier
		}
		fmt.Printf("Relation created: %s %s %s\n", from, rel.Type, to)
	}

	if parent != nil {
		fmt.Printf("Parent: %s %s\n", parent.Identifier, parent.Title)
	}

	return nil
}

// relationRequest pairs an issueRelationCreate input with the target issue as
// the user typed it on the command line (e.g. ENG-123), so errors can name
// the issue the user recognises rather than its resolved UUID.
type relationRequest struct {
	Input  client.CreateIssueRelationInput
	Target string
}

// buildRelationInputs translates the repeatable relation flags into
// issueRelationCreate inputs. Linear stores a blocks edge on the blocking
// issue, so --blocked-by is the same relation with the endpoints swapped.
func buildRelationInputs(currentID string, blocks, blockedBy, related []string, resolve func(string) (string, error)) ([]relationRequest, error) {
	requests := make([]relationRequest, 0, len(blocks)+len(blockedBy)+len(related))

	for _, target := range blocks {
		targetID, err := resolve(target)
		if err != nil {
			return nil, fmt.Errorf("resolving --blocks issue %s: %w", target, err)
		}
		requests = append(requests, relationRequest{
			Input: client.CreateIssueRelationInput{
				IssueID:        currentID,
				RelatedIssueID: targetID,
				Type:           client.RelationTypeBlocks,
			},
			Target: target,
		})
	}

	for _, target := range blockedBy {
		targetID, err := resolve(target)
		if err != nil {
			return nil, fmt.Errorf("resolving --blocked-by issue %s: %w", target, err)
		}
		requests = append(requests, relationRequest{
			Input: client.CreateIssueRelationInput{
				IssueID:        targetID,
				RelatedIssueID: currentID,
				Type:           client.RelationTypeBlocks,
			},
			Target: target,
		})
	}

	for _, target := range related {
		targetID, err := resolve(target)
		if err != nil {
			return nil, fmt.Errorf("resolving --related issue %s: %w", target, err)
		}
		requests = append(requests, relationRequest{
			Input: client.CreateIssueRelationInput{
				IssueID:        currentID,
				RelatedIssueID: targetID,
				Type:           client.RelationTypeRelated,
			},
			Target: target,
		})
	}

	return requests, nil
}

var issueCommentCmd = &cobra.Command{
	Use:   "comment <issue-id> <message>",
	Short: "Add a comment to an issue",
	Long: `Add a comment to a specific issue.

Examples:
  linear issue comment ENG-123 "Looks good to me"
  linear issue comment <uuid> "Deploying now"`,
	Args: cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		issueID := args[0]
		body := args[1]

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := c.CreateComment(ctx, issueID, body)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error creating comment: %v\n", err)
			os.Exit(1)
		}

		if !resp.CommentCreate.Success {
			fmt.Fprintln(os.Stderr, "Error: Failed to create comment")
			os.Exit(1)
		}

		if jsonOutput {
			if err := output.PrintJSON(resp.CommentCreate.Comment); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		fmt.Printf("Comment added to %s\n", issueID)
	},
}

var commentsCmd = &cobra.Command{
	Use:   "comments <issue-id>",
	Short: "List comments on an issue",
	Long: `List comments on a specific issue.

Examples:
  linear issue comments ENG-123
  linear issue comments <issue-uuid>`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		issueID := args[0]

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		resp, err := c.GetIssueComments(ctx, issueID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching comments: %v\n", err)
			os.Exit(1)
		}

		if resp.Issue == nil {
			fmt.Fprintf(os.Stderr, "Error: issue %s not found\n", issueID)
			os.Exit(1)
		}

		comments := resp.Issue.Comments.Nodes

		if resp.Issue.Comments.PageInfo.HasNextPage {
			fmt.Fprintf(os.Stderr, "Warning: more than %d comments exist (pagination not yet supported).\n", len(comments))
		}

		if jsonOutput {
			if err := output.PrintJSON(comments); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		if len(comments) == 0 {
			fmt.Println("No comments found.")
			return
		}

		for i, comment := range comments {
			user := "Unknown"
			if comment.User != nil {
				user = comment.User.Name
			}
			fmt.Printf("[%d] %s — %s\n", i+1, user, comment.CreatedAt)
			fmt.Printf("    %s\n\n", comment.Body)
		}
	},
}

func init() {
	issueListCmd.Flags().StringVar(&teamFilter, "team", "", "Filter by team key (e.g., ENG)")
	issueListCmd.Flags().StringVar(&projectFilter, "project", "", "Filter by project name or ID")
	issueListCmd.Flags().IntVar(&issueLimit, "limit", 50, "Maximum number of issues to fetch (default: 50)")
	issueListCmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all issues using pagination")
	issueListCmd.Flags().StringVar(&listAssigneeFilter, "assignee", "", "Filter by assignee email, or 'me' for the authenticated user")
	issueListCmd.Flags().BoolVar(&listMineFilter, "mine", false, "Only issues assigned to the authenticated user (shorthand for --assignee me)")
	issueListCmd.Flags().StringVar(&listStateFilter, "state", "", "Filter by workflow state name, case-insensitive (e.g., \"In Progress\")")
	issueListCmd.Flags().StringSliceVar(&listStateTypeFilter, "state-type", nil, "Filter by state type(s), comma-separated or repeated: triage, backlog, unstarted, started, completed, canceled, duplicate")

	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueDesc, "description", "", "Issue description")
	issueCreateCmd.Flags().StringVar(&issueTeamID, "team", "", "Team key (required)")
	issueCreateCmd.Flags().StringVar(&issueProjectIdentifier, "project", "", "Project name or ID (optional)")
	issueCreateCmd.Flags().StringVar(&issueAssignee, "assignee", "", "Issue Assignee (email, optional)")
	issueCreateCmd.Flags().StringVar(&issueState, "state", "", "Issue state/workflow status name, matched case-insensitively within the team (optional, e.g., \"In Progress\")")
	issueCreateCmd.Flags().IntVar(&issuePriority, "priority", 0, "Issue priority (0-4, optional)")
	issueCreateCmd.Flags().StringVar(&issueParent, "parent", "", "Parent issue identifier or ID, making this a sub-issue (optional)")

	issueUpdateCmd.Flags().StringVar(&issueUpdateTitle, "title", "", "Updated issue title")
	issueUpdateCmd.Flags().StringVar(&issueUpdateDesc, "description", "", "Updated issue description (use empty string to clear)")
	issueUpdateCmd.Flags().IntVar(&issueUpdatePriority, "priority", 0, "Updated issue priority (0-4)")
	issueUpdateCmd.Flags().StringVar(&issueUpdateProject, "project", "", "Updated project name or ID")
	issueUpdateCmd.Flags().StringVar(&issueUpdateAssignee, "assignee", "", "Updated issue assignee (email)")
	issueUpdateCmd.Flags().StringVar(&issueUpdateState, "state", "", "Updated issue state/workflow status name, matched case-insensitively within the issue's team")
	issueUpdateCmd.Flags().StringVar(&issueUpdateParent, "parent", "", "Parent issue identifier or ID, making this a sub-issue")
	issueUpdateCmd.Flags().StringArrayVar(&issueUpdateBlocks, "blocks", nil, "Issue this issue blocks, by identifier or ID (repeatable)")
	issueUpdateCmd.Flags().StringArrayVar(&issueUpdateBlockedBy, "blocked-by", nil, "Issue that blocks this issue, by identifier or ID (repeatable)")
	issueUpdateCmd.Flags().StringArrayVar(&issueUpdateRelated, "related", nil, "Related issue, by identifier or ID (repeatable)")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	issueCmd.AddCommand(issueUpdateCmd)
	issueCmd.AddCommand(commentsCmd)
	issueCmd.AddCommand(issueCommentCmd)
	rootCmd.AddCommand(issueCmd)
}
