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
	projectTeamFilter string
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

func init() {
	projectListCmd.Flags().StringVar(&projectTeamFilter, "team", "", "Filter projects by team key (e.g., ENG)")

	projectCmd.AddCommand(projectListCmd)
	rootCmd.AddCommand(projectCmd)
}
