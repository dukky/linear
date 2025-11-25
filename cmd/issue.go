package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/dukky/linear/internal/client"
	"github.com/dukky/linear/internal/output"
	"github.com/spf13/cobra"
)

var (
	teamFilter  string
	issueTitle  string
	issueDesc   string
	issueTeamID string
	issueLimit  int
	fetchAll    bool
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
Use --limit to specify the number of issues to fetch (default: 50).
Use --all to fetch all issues using pagination.`,
	Run: func(cmd *cobra.Command, args []string) {
		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		var ctx context.Context
		var cancel context.CancelFunc

		if fetchAll {
			// For --all operations, use longer overall timeout to allow pagination
			ctx, cancel = context.WithTimeout(context.Background(), 10*time.Minute)
		} else {
			// For single-page operations, use standard timeout
			ctx, cancel = context.WithTimeout(context.Background(), 30*time.Second)
		}
		defer cancel()

		var issues []client.Issue

		if fetchAll {
			// Fetch all issues using pagination
			allIssues, err := c.ListAllIssues(ctx, teamFilter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
				os.Exit(1)
			}
			issues = allIssues
		} else {
			// Fetch with specified limit
			opts := client.ListIssuesOptions{
				TeamKey: teamFilter,
				Limit:   issueLimit,
			}
			resp, err := c.ListIssues(ctx, opts)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching issues: %v\n", err)
				os.Exit(1)
			}
			issues = resp.Issues.Nodes
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
	},
}

var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	Long: `Create a new issue in Linear.

Examples:
  linear issue create --team ENG --title "Fix bug" --description "Bug details"
  linear issue create --team ENG --title "New feature"`,
	Run: func(cmd *cobra.Command, args []string) {
		if issueTitle == "" {
			fmt.Fprintln(os.Stderr, "Error: --title is required")
			os.Exit(1)
		}

		if issueTeamID == "" {
			fmt.Fprintln(os.Stderr, "Error: --team is required")
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

		// Create the issue
		input := client.CreateIssueInput{
			Title:  issueTitle,
			TeamID: teamID,
		}

		if issueDesc != "" {
			input.Description = issueDesc
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
	},
}

func init() {
	issueListCmd.Flags().StringVar(&teamFilter, "team", "", "Filter by team key (e.g., ENG)")
	issueListCmd.Flags().IntVar(&issueLimit, "limit", 50, "Maximum number of issues to fetch (default: 50)")
	issueListCmd.Flags().BoolVar(&fetchAll, "all", false, "Fetch all issues using pagination")

	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueDesc, "description", "", "Issue description")
	issueCreateCmd.Flags().StringVar(&issueTeamID, "team", "", "Team key (required)")

	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)
	rootCmd.AddCommand(issueCmd)
}
