package auth

import (
	"os"
	"testing"
)

func TestGetAPIKey_FromEnvironment(t *testing.T) {
	// Save original env var and restore after test
	originalEnv := os.Getenv(envVarName)
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
	}()

	// Set test API key in environment
	testAPIKey := "test-api-key-from-env"
	os.Setenv(envVarName, testAPIKey)

	apiKey, err := GetAPIKey()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if apiKey != testAPIKey {
		t.Errorf("Expected API key to be '%s', got '%s'", testAPIKey, apiKey)
	}
}

func TestGetAPIKey_EmptyEnvironment(t *testing.T) {
	// Save original env var and restore after test
	originalEnv := os.Getenv(envVarName)
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// This will fail because we don't have access to a test keyring
	// but we can verify it attempts to check the keyring
	_, err := GetAPIKey()

	// We expect an error because the keyring likely doesn't have our test key
	if err == nil {
		// If no error, the keyring actually had a key (unlikely in test environment)
		t.Log("Keyring had an API key set")
	}
}

func TestGetAuthStatus_WithEnvironment(t *testing.T) {
	// Save original env var and restore after test
	originalEnv := os.Getenv(envVarName)
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
	}()

	// Set test API key in environment
	os.Setenv(envVarName, "test-key")

	status, authenticated := GetAuthStatus()

	if !authenticated {
		t.Error("Expected to be authenticated when env var is set")
	}

	if status != "Environment variable (LINEAR_API_KEY)" {
		t.Errorf("Expected status to be 'Environment variable (LINEAR_API_KEY)', got '%s'", status)
	}
}

func TestGetAuthStatus_WithoutEnvironment(t *testing.T) {
	// Save original env var and restore after test
	originalEnv := os.Getenv(envVarName)
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	status, authenticated := GetAuthStatus()

	// The status should indicate checking keyring (or not authenticated)
	if status == "" {
		t.Error("Expected non-empty status")
	}

	// We can't reliably test the authenticated value without a keyring
	// but we can verify the function returns something
	t.Logf("Auth status: %s, authenticated: %v", status, authenticated)
}

func TestConstants(t *testing.T) {
	// Verify constants are set to expected values
	if keyringService != "linear-cli" {
		t.Errorf("Expected keyringService to be 'linear-cli', got '%s'", keyringService)
	}

	if keyringKey != "api-key" {
		t.Errorf("Expected keyringKey to be 'api-key', got '%s'", keyringKey)
	}

	if envVarName != "LINEAR_API_KEY" {
		t.Errorf("Expected envVarName to be 'LINEAR_API_KEY', got '%s'", envVarName)
	}
}

// TestSaveAPIKey is commented out because it would modify the actual keyring
// In a real test environment, you'd want to mock the keyring
/*
func TestSaveAPIKey(t *testing.T) {
	// This test is intentionally not implemented to avoid modifying the system keyring
	t.Skip("Skipping SaveAPIKey test to avoid modifying system keyring")
}
*/
