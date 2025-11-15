package cmd

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/linear-cli/linear/internal/api"
	"github.com/linear-cli/linear/internal/config"
	"github.com/spf13/cobra"
)

// teamCmd represents the team command
var teamCmd = &cobra.Command{
	Use:     "team",
	Aliases: []string{"teams"},
	Short:   "Manage Linear teams",
	Long:    `List and view Linear teams.`,
}

// teamListCmd lists teams
var teamListCmd = &cobra.Command{
	Use:     "list",
	Aliases: []string{"ls"},
	Short:   "List all teams",
	RunE:    runTeamList,
}

func init() {
	rootCmd.AddCommand(teamCmd)
	teamCmd.AddCommand(teamListCmd)
}

func runTeamList(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return err
	}

	client, err := api.NewClient(cfg)
	if err != nil {
		return fmt.Errorf("authentication required. Run 'linear auth' first: %w", err)
	}

	teams, err := client.ListTeams()
	if err != nil {
		return fmt.Errorf("failed to list teams: %w", err)
	}

	if len(teams) == 0 {
		fmt.Println("No teams found")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "KEY\tNAME\tDESCRIPTION")
	fmt.Fprintln(w, "───\t────\t───────────")

	for _, team := range teams {
		desc := team.Description
		if desc == "" {
			desc = "-"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", team.Key, team.Name, truncate(desc, 50))
	}

	w.Flush()
	fmt.Printf("\nShowing %d team(s)\n", len(teams))

	return nil
}
