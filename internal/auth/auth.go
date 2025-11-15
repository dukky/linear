package auth

import (
	"errors"
	"fmt"
	"os"

	"github.com/99designs/keyring"
)

const (
	keyringService = "linear-cli"
	keyringKey     = "api-key"
	envVarName     = "LINEAR_API_KEY"
)

// GetAPIKey retrieves the Linear API key from keyring or environment variable
func GetAPIKey() (string, error) {
	// First, check environment variable
	if apiKey := os.Getenv(envVarName); apiKey != "" {
		return apiKey, nil
	}

	// Then check keyring
	ring, err := openKeyring()
	if err != nil {
		return "", fmt.Errorf("failed to access keyring: %w", err)
	}

	item, err := ring.Get(keyringKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "", errors.New("no API key found. Run 'linear auth login' or set LINEAR_API_KEY environment variable")
		}
		return "", fmt.Errorf("failed to retrieve API key from keyring: %w", err)
	}

	return string(item.Data), nil
}

// SaveAPIKey stores the API key in the system keyring
func SaveAPIKey(apiKey string) error {
	ring, err := openKeyring()
	if err != nil {
		return fmt.Errorf("failed to access keyring: %w", err)
	}

	err = ring.Set(keyring.Item{
		Key:  keyringKey,
		Data: []byte(apiKey),
	})
	if err != nil {
		return fmt.Errorf("failed to save API key to keyring: %w", err)
	}

	return nil
}

// GetAuthStatus returns information about the current authentication status
func GetAuthStatus() (string, bool) {
	// Check environment variable first
	if os.Getenv(envVarName) != "" {
		return "Environment variable (LINEAR_API_KEY)", true
	}

	// Check keyring
	ring, err := openKeyring()
	if err != nil {
		return fmt.Sprintf("Error accessing keyring: %v", err), false
	}

	_, err = ring.Get(keyringKey)
	if err != nil {
		if errors.Is(err, keyring.ErrKeyNotFound) {
			return "Not authenticated", false
		}
		return fmt.Sprintf("Error reading keyring: %v", err), false
	}

	return "System keyring", true
}

// openKeyring opens the system keyring with appropriate configuration
func openKeyring() (keyring.Keyring, error) {
	return keyring.Open(keyring.Config{
		ServiceName: keyringService,
		// Use the most appropriate backend for each OS
		AllowedBackends: []keyring.BackendType{
			keyring.KeychainBackend,       // macOS
			keyring.WinCredBackend,        // Windows
			keyring.SecretServiceBackend,  // Linux with Secret Service
			keyring.KWalletBackend,        // KDE
			keyring.FileBackend,           // Fallback to encrypted file
		},
	})
}
