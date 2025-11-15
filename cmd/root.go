package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	jsonOutput bool
	rootCmd    = &cobra.Command{
		Use:   "linear",
		Short: "Linear CLI - Manage Linear issues, projects, and teams from the command line",
		Long: `A command-line interface for Linear issue tracking.

Authenticate with your Linear API key using 'linear auth login' or set the
LINEAR_API_KEY environment variable.

Perfect for use with Claude Code and human workflows.`,
	}
)

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "Output in JSON format")
}
