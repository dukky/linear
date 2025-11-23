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
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Enter your Linear API key (starts with 'lin_api_'):")
		fmt.Print("> ")

		apiKeyBytes, err := term.ReadPassword(int(syscall.Stdin))
		if err != nil {
			return fmt.Errorf("error reading input: %w", err)
		}
		fmt.Println() // Print newline after password input

		apiKey := strings.TrimSpace(string(apiKeyBytes))
		if apiKey == "" {
			return fmt.Errorf("API key cannot be empty")
		}

		if !strings.HasPrefix(apiKey, "lin_api_") {
			fmt.Fprintln(os.Stderr, "Warning: API key should start with 'lin_api_'")
		}

		err = auth.SaveAPIKey(apiKey)
		if err != nil {
			return fmt.Errorf("error saving API key: %w", err)
		}

		fmt.Println("\nAuthentication successful!")
		fmt.Println("Your API key has been stored securely in the system keyring.")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show authentication status",
	Long:  "Display current authentication status and source (keyring or environment variable)",
	Run: func(cmd *cobra.Command, args []string) {
		source, authenticated := auth.GetAuthStatus()

		if authenticated {
			fmt.Printf("Status: Authenticated\n")
			fmt.Printf("Source: %s\n", source)
		} else {
			fmt.Printf("Status: Not authenticated\n")
			fmt.Println("\nTo authenticate, run: linear auth login")
			fmt.Println("Or set the LINEAR_API_KEY environment variable")
		}
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd)
	authCmd.AddCommand(authStatusCmd)
	rootCmd.AddCommand(authCmd)
}
