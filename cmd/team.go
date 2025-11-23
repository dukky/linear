package cmd

import (
	"context"
	"fmt"
	"time"

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
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := client.NewClient()
		if err != nil {
			return err
		}

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		resp, err := c.ListTeams(ctx)
		if err != nil {
			return fmt.Errorf("error fetching teams: %w", err)
		}

		if jsonOutput {
			if err := output.PrintJSON(resp.Teams.Nodes); err != nil {
				return fmt.Errorf("error formatting output: %w", err)
			}
			return nil
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
		return nil
	},
}

func init() {
	teamCmd.AddCommand(teamListCmd)
	rootCmd.AddCommand(teamCmd)
}
