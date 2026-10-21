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
	projectTeamFilter     string
	projectViewTeamFilter string
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Manage projects",
	Long:  "List and view Linear projects",
}

var projectListCmd = &cobra.Command{
	Use:   "list",
	Short: "List projects",
	Long:  "List all projects in your Linear workspace, optionally filtered by team",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		var resp *client.ProjectsResponse

		if projectTeamFilter != "" {
			// Get team by key first
			teamResp, err := c.GetTeamByKey(ctx, projectTeamFilter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching team: %v\n", err)
				os.Exit(1)
			}
			if len(teamResp.Teams.Nodes) == 0 {
				fmt.Fprintf(os.Stderr, "Team not found: %s\n", projectTeamFilter)
				os.Exit(1)
			}

			teamID := teamResp.Teams.Nodes[0].ID

			// Get projects for the team
			resp, err = c.GetProjectsByTeam(ctx, teamID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching projects: %v\n", err)
				os.Exit(1)
			}
		} else {
			// Get all projects
			resp, err = c.ListProjects(ctx)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching projects: %v\n", err)
				os.Exit(1)
			}
		}

		if jsonOutput {
			if err := output.PrintJSON(resp.Projects.Nodes); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Table output
		table := output.NewTable([]string{"ID", "NAME"})
		for _, project := range resp.Projects.Nodes {
			table.AddRow([]string{project.ID, project.Name})
		}
		table.Print()
	},
}

var projectViewCmd = &cobra.Command{
	Use:   "view <name|id>",
	Short: "View project details",
	Long: `View details of a project by name or UUID.

Examples:
  linear project view "Mobile App"
  linear project view "Mobile App" --team ENG
  linear project view 4e26961e-967f-458f-8fa2-4240035aa178`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		identifier := args[0]

		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		teamID := ""
		if projectViewTeamFilter != "" {
			teamResp, err := c.GetTeamByKey(ctx, projectViewTeamFilter)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching team: %v\n", err)
				os.Exit(1)
			}
			if len(teamResp.Teams.Nodes) == 0 {
				fmt.Fprintf(os.Stderr, "Team not found: %s\n", projectViewTeamFilter)
				os.Exit(1)
			}
			teamID = teamResp.Teams.Nodes[0].ID
		}

		project, err := c.GetProjectByIdentifier(ctx, identifier, teamID)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching project: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			if err := output.PrintJSON(project); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Human-readable output
		fmt.Printf("Name:        %s\n", project.Name)
		fmt.Printf("ID:          %s\n", project.ID)

		if project.Status != nil {
			if project.Status.Type != "" {
				fmt.Printf("Status:      %s (%s)\n", project.Status.Name, project.Status.Type)
			} else {
				fmt.Printf("Status:      %s\n", project.Status.Name)
			}
		}

		fmt.Printf("URL:         %s\n", project.URL)

		if project.Description != "" {
			fmt.Printf("\nDescription:\n%s\n", project.Description)
		} else {
			fmt.Printf("\nDescription:\n(no description)\n")
		}
	},
}

func init() {
	projectListCmd.Flags().StringVar(&projectTeamFilter, "team", "", "Filter projects by team key (e.g., ENG)")
	projectViewCmd.Flags().StringVar(&projectViewTeamFilter, "team", "", "Team key to disambiguate project names (e.g., ENG)")

	projectCmd.AddCommand(projectListCmd)
	projectCmd.AddCommand(projectViewCmd)
	rootCmd.AddCommand(projectCmd)
}
