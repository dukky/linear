package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/linear-cli/linear/internal/config"
)

// OAuthClient handles the OAuth flow
type OAuthClient struct {
	ClientID     string
	ClientSecret string
	Config       *config.Config
}

type authorizationResponse struct {
	Code  string
	State string
	Error string
}

// NewOAuthClient creates a new OAuth client
func NewOAuthClient(clientID, clientSecret string, cfg *config.Config) *OAuthClient {
	return &OAuthClient{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Config:       cfg,
	}
}

// Authenticate performs the OAuth flow with PKCE
func (o *OAuthClient) Authenticate() error {
	// Generate PKCE code verifier and challenge
	codeVerifier, err := generateCodeVerifier()
	if err != nil {
		return fmt.Errorf("failed to generate code verifier: %w", err)
	}

	codeChallenge := generateCodeChallenge(codeVerifier)

	// Generate random state for CSRF protection
	state, err := generateRandomString(32)
	if err != nil {
		return fmt.Errorf("failed to generate state: %w", err)
	}

	// Build authorization URL
	authURL := o.buildAuthURL(codeChallenge, state)

	fmt.Println("Opening browser for authentication...")
	fmt.Println("If the browser doesn't open automatically, please visit:")
	fmt.Println(authURL)
	fmt.Println()

	// Start local server to receive callback
	authCode, err := o.startCallbackServer(state)
	if err != nil {
		return fmt.Errorf("authentication failed: %w", err)
	}

	// Exchange authorization code for access token
	token, err := o.exchangeToken(authCode, codeVerifier)
	if err != nil {
		return fmt.Errorf("failed to exchange token: %w", err)
	}

	// Save token securely
	if err := o.Config.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}

	fmt.Println("\n✓ Authentication successful!")
	return nil
}

// buildAuthURL constructs the OAuth authorization URL
func (o *OAuthClient) buildAuthURL(codeChallenge, state string) string {
	params := url.Values{
		"client_id":             {o.ClientID},
		"redirect_uri":          {config.RedirectURL},
		"response_type":         {"code"},
		"state":                 {state},
		"scope":                 {"read write"},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
		"prompt":                {"consent"},
	}

	return fmt.Sprintf("%s?%s", config.LinearAuthURL, params.Encode())
}

// startCallbackServer starts a temporary HTTP server to receive the OAuth callback
func (o *OAuthClient) startCallbackServer(expectedState string) (string, error) {
	resultChan := make(chan authorizationResponse, 1)

	server := &http.Server{
		Addr:         "127.0.0.1:" + config.RedirectPort,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")
		errorMsg := r.URL.Query().Get("error")

		response := authorizationResponse{
			Code:  code,
			State: state,
			Error: errorMsg,
		}

		resultChan <- response

		// Send success page
		w.Header().Set("Content-Type", "text/html")
		if errorMsg != "" {
			fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Failed</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
	<h1>❌ Authentication Failed</h1>
	<p>Error: %s</p>
	<p>You can close this window.</p>
</body>
</html>`, errorMsg)
		} else {
			fmt.Fprint(w, `
<!DOCTYPE html>
<html>
<head><title>Authentication Successful</title></head>
<body style="font-family: sans-serif; text-align: center; padding: 50px;">
	<h1>✓ Authentication Successful</h1>
	<p>You can close this window and return to the terminal.</p>
</body>
</html>`)
		}
	})

	go func() {
		server.ListenAndServe()
	}()

	// Wait for callback with timeout
	var result authorizationResponse
	select {
	case result = <-resultChan:
		server.Close()
	case <-time.After(5 * time.Minute):
		server.Close()
		resultChan <- authorizationResponse{Error: "timeout waiting for authorization"}
	}

	if result.Error != "" {
		return "", fmt.Errorf("authorization error: %s", result.Error)
	}

	if result.State != expectedState {
		return "", fmt.Errorf("state mismatch: possible CSRF attack")
	}

	if result.Code == "" {
		return "", fmt.Errorf("no authorization code received")
	}

	return result.Code, nil
}

// exchangeToken exchanges the authorization code for an access token
func (o *OAuthClient) exchangeToken(code, codeVerifier string) (*config.TokenData, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {config.RedirectURL},
		"client_id":     {o.ClientID},
		"client_secret": {o.ClientSecret},
		"code_verifier": {codeVerifier},
	}

	req, err := http.NewRequest("POST", config.LinearTokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token exchange failed: %s - %s", resp.Status, string(body))
	}

	var token config.TokenData
	if err := json.Unmarshal(body, &token); err != nil {
		return nil, fmt.Errorf("failed to parse token response: %w", err)
	}

	return &token, nil
}

// generateCodeVerifier generates a random code verifier for PKCE
func generateCodeVerifier() (string, error) {
	return generateRandomString(64)
}

// generateCodeChallenge creates a code challenge from the verifier
func generateCodeChallenge(verifier string) string {
	hash := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(hash[:])
}

// generateRandomString generates a cryptographically secure random string
func generateRandomString(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes)[:length], nil
}
