package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/linear-cli/linear/internal/auth"
	"github.com/linear-cli/linear/internal/config"
	"github.com/spf13/cobra"
)

var (
	clientID     string
	clientSecret string
)

// authCmd represents the auth command
var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Authenticate with Linear",
	Long: `Authenticate with Linear using OAuth 2.0.

This command will open your browser to complete the authentication flow.
Your credentials will be securely stored on your local machine.`,
	RunE: runAuth,
}

// logoutCmd represents the logout command
var logoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Log out and remove stored credentials",
	RunE:  runLogout,
}

// whoamiCmd shows the currently authenticated user
var whoamiCmd = &cobra.Command{
	Use:   "whoami",
	Short: "Show the currently authenticated user",
	RunE:  runWhoami,
}

func init() {
	rootCmd.AddCommand(authCmd)
	authCmd.AddCommand(logoutCmd)
	authCmd.AddCommand(whoamiCmd)

	authCmd.Flags().StringVar(&clientID, "client-id", "", "Linear OAuth client ID")
	authCmd.Flags().StringVar(&clientSecret, "client-secret", "", "Linear OAuth client secret")
}

func runAuth(cmd *cobra.Command, args []string) error {
	// Get client credentials from environment or flags
	if clientID == "" {
		clientID = os.Getenv("LINEAR_CLIENT_ID")
	}
	if clientSecret == "" {
		clientSecret = os.Getenv("LINEAR_CLIENT_SECRET")
	}

	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("client ID and secret are required. Set LINEAR_CLIENT_ID and LINEAR_CLIENT_SECRET environment variables or use --client-id and --client-secret flags")
	}

	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	oauthClient := auth.NewOAuthClient(clientID, clientSecret, cfg)

	// Build auth URL and open browser
	fmt.Println("Starting OAuth authentication flow...")

	// Try to open browser automatically
	if err := openBrowser(buildBrowserURL(oauthClient)); err != nil {
		fmt.Printf("Could not open browser automatically: %v\n", err)
	}

	if err := oauthClient.Authenticate(); err != nil {
		return err
	}

	return nil
}

func runLogout(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	if err := cfg.ClearToken(); err != nil {
		return fmt.Errorf("failed to clear credentials: %w", err)
	}

	fmt.Println("✓ Successfully logged out")
	return nil
}

func runWhoami(cmd *cobra.Command, args []string) error {
	cfg, err := config.New()
	if err != nil {
		return fmt.Errorf("failed to initialize config: %w", err)
	}

	token, err := cfg.LoadToken()
	if err != nil {
		return fmt.Errorf("not authenticated. Run 'linear auth' to authenticate")
	}

	fmt.Printf("Authenticated with token: %s...\n", token.AccessToken[:20])
	return nil
}

func buildBrowserURL(client *auth.OAuthClient) string {
	// This is a simplified version - the actual auth flow handles this
	return config.LinearAuthURL
}

// openBrowser tries to open the URL in the default browser
func openBrowser(url string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform")
	}

	return cmd.Start()
}
