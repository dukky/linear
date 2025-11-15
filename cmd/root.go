package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	// Version information
	Version = "0.1.0"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "linear",
	Short: "Linear CLI - Command line interface for Linear",
	Long: `A secure command line interface for Linear that supports OAuth authentication
and provides easy access to Linear's API for managing issues, projects, and teams.`,
	Version: Version,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SetVersionTemplate(`{{.Version}}`)
}
