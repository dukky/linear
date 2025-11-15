package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"github.com/linear-cli/linear/internal/api"
	"github.com/linear-cli/linear/internal/config"
	"github.com/spf13/cobra"
)

var (
	issueLimit  int
	issueTeam   string
	issueTitle  string
	issueDesc   string
	issueAssign string
)

// issueCmd represents the issue command
var issueCmd = &cobra.Command{
	Use:     "issue",
	Aliases: []string{"issues"},
	Short:   "Manage Linear issues",
	Long:    `Create, list, and view Linear issues.`,
}

// issueListCmd lists issues
var issueListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List issues",
	RunE:    runIssueList,
}

// issueViewCmd views a single issue
var issueViewCmd = &cobra.Command{
	Use:   "view <issue-id>",
	Short: "View issue details",
	Args:  cobra.ExactArgs(1),
	RunE:  runIssueView,
}

// issueCreateCmd creates a new issue
var issueCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create a new issue",
	RunE:  runIssueCreate,
}

func init() {
	rootCmd.AddCommand(issueCmd)
	issueCmd.AddCommand(issueListCmd)
	issueCmd.AddCommand(issueViewCmd)
	issueCmd.AddCommand(issueCreateCmd)

	issueListCmd.Flags().IntVarP(&issueLimit, "limit", "n", 10, "Number of issues to retrieve")
	issueListCmd.Flags().StringVarP(&issueTeam, "team", "t", "", "Filter by team key")

	issueCreateCmd.Flags().StringVar(&issueTitle, "title", "", "Issue title (required)")
	issueCreateCmd.Flags().StringVar(&issueDesc, "description", "", "Issue description")
	issueCreateCmd.Flags().StringVarP(&issueTeam, "team", "t", "", "Team ID (required)")
	issueCreateCmd.Flags().StringVar(&issueAssign, "assignee", "", "Assignee ID")
	issueCreateCmd.MarkFlagRequired("title")
	issueCreateCmd.MarkFlagRequired("team")
}

func runIssueList(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("authentication required. Run 'linear auth' first: %w", err)
	}

	var filter map[string]interface{}
	if issueTeam != "" {
		filter = map[string]interface{}{
			"team": map[string]interface{}{
				"key": map[string]interface{}{
					"eq": issueTeam,
				},
			},
		}
	}

	issues, err := client.ListIssues(issueLimit, filter)
	if err != nil {
		return fmt.Errorf("failed to list issues: %w", err)
	}

	if len(issues.Nodes) == 0 {
		fmt.Println("No issues found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTITLE\tSTATE\tPRIORITY\tASSIGNEE\tUPDATED")
	fmt.Fprintln(w, "──\t─────\t─────\t────────\t────────\t───────")

	for _, issue := range issues.Nodes {
		state := "N/A"
		if issue.State != nil {
			state = issue.State.Name
		}

		assignee := "Unassigned"
		if issue.Assignee != nil {
			assignee = issue.Assignee.DisplayName
			if assignee == "" {
				assignee = issue.Assignee.Name
			}
		}

		updated := formatTime(issue.UpdatedAt)

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\n",
			issue.Identifier,
			truncate(issue.Title, 40),
			state,
			issue.PriorityLabel,
			truncate(assignee, 15),
			updated,
		)
	}

	w.Flush()
	fmt.Printf("\nShowing %d issue(s)\n", len(issues.Nodes))

	return nil
}

func runIssueView(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("authentication required. Run 'linear auth' first: %w", err)
	}

	issue, err := client.GetIssue(args[0])
	if err != nil {
		return fmt.Errorf("failed to get issue: %w", err)
	}

	fmt.Printf("\n%s: %s\n", issue.Identifier, issue.Title)
	fmt.Println(formatSeparator(len(issue.Identifier) + len(issue.Title) + 2))

	if issue.State != nil {
		fmt.Printf("State:    %s\n", issue.State.Name)
	}

	fmt.Printf("Priority: %s\n", issue.PriorityLabel)

	if issue.Assignee != nil {
		name := issue.Assignee.DisplayName
		if name == "" {
			name = issue.Assignee.Name
		}
		fmt.Printf("Assignee: %s\n", name)
	}

	if issue.Team != nil {
		fmt.Printf("Team:     %s (%s)\n", issue.Team.Name, issue.Team.Key)
	}

	fmt.Printf("Created:  %s\n", formatTime(issue.CreatedAt))
	fmt.Printf("Updated:  %s\n", formatTime(issue.UpdatedAt))
	fmt.Printf("URL:      %s\n", issue.URL)

	if issue.Description != "" {
		fmt.Printf("\nDescription:\n%s\n", issue.Description)
	}

	fmt.Println()

	return nil
}

func runIssueCreate(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("authentication required. Run 'linear auth' first: %w", err)
	}

	input := api.IssueCreateInput{
		Title:       issueTitle,
		Description: issueDesc,
		TeamID:      issueTeam,
		AssigneeID:  issueAssign,
	}

	issue, err := client.CreateIssue(input)
	if err != nil {
		return fmt.Errorf("failed to create issue: %w", err)
	}

	fmt.Printf("✓ Created issue %s: %s\n", issue.Identifier, issue.Title)
	fmt.Printf("  URL: %s\n", issue.URL)

	return nil
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func formatTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	switch {
	case diff < time.Minute:
		return "just now"
	case diff < time.Hour:
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	case diff < 24*time.Hour:
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	case diff < 7*24*time.Hour:
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	default:
		return t.Format("Jan 2")
	}
}

func formatSeparator(length int) string {
	sep := ""
	for i := 0; i < length; i++ {
		sep += "─"
	}
	return sep
}
