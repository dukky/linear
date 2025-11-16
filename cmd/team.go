package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/dukky/linear/internal/client"
	"github.com/dukky/linear/internal/output"
	"github.com/spf13/cobra"
)

var teamCmd = &cobra.Command{
	Use:   "team",
	Short: "Manage teams",
	Long:  "List and view Linear teams",
}

var teamListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all teams",
	Long:  "List all teams in your Linear workspace",
	Run: func(cmd *cobra.Command, args []string) {
		c, err := client.NewClient()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}

		ctx := context.Background()
		resp, err := c.ListTeams(ctx)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error fetching teams: %v\n", err)
			os.Exit(1)
		}

		if jsonOutput {
			if err := output.PrintJSON(resp.Teams.Nodes); err != nil {
				fmt.Fprintf(os.Stderr, "Error formatting output: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// Table output
		table := output.NewTable([]string{"KEY", "NAME", "DESCRIPTION"})
		for _, team := range resp.Teams.Nodes {
			desc := ""
			if team.Description != nil {
				desc = output.FormatMultilineString(*team.Description, 50)
			}
			table.AddRow([]string{team.Key, team.Name, desc})
		}
		table.Print()
	},
}

func init() {
	teamCmd.AddCommand(teamListCmd)
	rootCmd.AddCommand(teamCmd)
}
