package cmd

import (
	"fmt"
	"os"
	"strings"
	"syscall"

	"github.com/dukky/linear/internal/auth"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	getAuthStatus = auth.GetAuthStatus
	removeAPIKey  = auth.RemoveAPIKey
)

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage authentication",
	Long:  "Manage Linear API authentication using your API key",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with Linear API key",
	Long: `Store your Linear API key securely in the system keyring.

Get your API key from: https://linear.app/settings/api

The key will be stored securely in your system's keyring (macOS Keychain,
Windows Credential Manager, or Linux Secret Service).

Alternatively, you can set the LINEAR_API_KEY environment variable.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("Enter your Linear API key (starts with 'lin_api_'):")
		fmt.Print("> ")

		apiKeyBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError reading input: %v\n", err)
			os.Exit(1)
		}
		fmt.Println() // Print newline after password input

		apiKey := strings.TrimSpace(string(apiKeyBytes))
		if apiKey == "" {
			fmt.Fprintln(os.Stderr, "Error: API key cannot be empty")
			os.Exit(1)
		}

		if !strings.HasPrefix(apiKey, "lin_api_") {
			fmt.Fprintln(os.Stderr, "Warning: API key should start with 'lin_api_'")
		}

		err = auth.SaveAPIKey(apiKey)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error saving API key: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\nAuthentication successful!")
		fmt.Println("Your API key has been stored securely in the system keyring.")
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Display current authentication status and source (keyring or environment variable)",
	Run: func(cmd *cobra.Command, args []string) {
		source, authenticated := getAuthStatus()

		if authenticated {
			fmt.Printf("Status: Authenticated\n")
			fmt.Printf("Source: %s\n", source)
		} else {
			fmt.Printf("Status: Not authenticated\n")
			if source != "Not authenticated" {
				fmt.Printf("Details: %s\n", source)
			}
			fmt.Println("\nTo authenticate, run: linear auth login")
			fmt.Println("Or set the LINEAR_API_KEY environment variable")
		}
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Clear stored authentication",
	Long:  "Remove stored API key from the system keyring. Environment variable LINEAR_API_KEY is unaffected",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Removing your saved API key from keyring...")
		removed, err := removeAPIKey()
		if err != nil {
			return fmt.Errorf("error removing API key: %w", err)
		}
		if removed {
			fmt.Println("Stored key removed from keyring.")
		} else {
			fmt.Println("No API key set, exiting...")
		}
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	authCmd.AddCommand(authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}
