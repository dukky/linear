package auth

import (
	"errors"
	"os"
	"testing"

	"github.com/99designs/keyring"
)

// mockKeyringProvider is a mock implementation of KeyringProvider for testing
type mockKeyringProvider struct {
	items map[string]keyring.Item
	err   error
}

func (m *mockKeyringProvider) Get(key string) (keyring.Item, error) {
	if m.err != nil {
		return keyring.Item{}, m.err
	}
	item, ok := m.items[key]
	if !ok {
		return keyring.Item{}, keyring.ErrKeyNotFound
	}
	return item, nil
}

func (m *mockKeyringProvider) Set(item keyring.Item) error {
	if m.err != nil {
		return m.err
	}
	if m.items == nil {
		m.items = make(map[string]keyring.Item)
	}
	m.items[item.Key] = item
	return nil
}

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

func TestGetAPIKey_FromKeyring(t *testing.T) {
	// Save original env var and keyring opener, restore after test
	originalEnv := os.Getenv(envVarName)
	originalOpener := keyringOpener
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
		keyringOpener = originalOpener
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// Set up mock keyring with API key
	testAPIKey := "test-api-key-from-keyring"
	mock := &mockKeyringProvider{
		items: map[string]keyring.Item{
			keyringKey: {
				Key:  keyringKey,
				Data: []byte(testAPIKey),
			},
		},
	}
	keyringOpener = func() (KeyringProvider, error) {
		return mock, nil
	}

	apiKey, err := GetAPIKey()

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if apiKey != testAPIKey {
		t.Errorf("Expected API key to be '%s', got '%s'", testAPIKey, apiKey)
	}
}

func TestGetAPIKey_KeyringNotFound(t *testing.T) {
	// Save original env var and keyring opener, restore after test
	originalEnv := os.Getenv(envVarName)
	originalOpener := keyringOpener
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
		keyringOpener = originalOpener
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// Set up mock keyring with no API key
	mock := &mockKeyringProvider{
		items: make(map[string]keyring.Item),
	}
	keyringOpener = func() (KeyringProvider, error) {
		return mock, nil
	}

	_, err := GetAPIKey()

	if err == nil {
		t.Error("Expected error when API key not found, got nil")
	}

	expectedMsg := "no API key found"
	if err != nil && !contains(err.Error(), expectedMsg) {
		t.Errorf("Expected error to contain '%s', got '%s'", expectedMsg, err.Error())
	}
}

func TestGetAPIKey_KeyringError(t *testing.T) {
	// Save original env var and keyring opener, restore after test
	originalEnv := os.Getenv(envVarName)
	originalOpener := keyringOpener
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
		keyringOpener = originalOpener
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// Set up mock keyring that returns an error
	mockErr := errors.New("keyring access denied")
	keyringOpener = func() (KeyringProvider, error) {
		return nil, mockErr
	}

	_, err := GetAPIKey()

	if err == nil {
		t.Error("Expected error when keyring fails to open, got nil")
	}

	if err != nil && !contains(err.Error(), "failed to access keyring") {
		t.Errorf("Expected error to contain 'failed to access keyring', got '%s'", err.Error())
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

func TestGetAuthStatus_WithKeyring(t *testing.T) {
	// Save original env var and keyring opener, restore after test
	originalEnv := os.Getenv(envVarName)
	originalOpener := keyringOpener
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
		keyringOpener = originalOpener
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// Set up mock keyring with API key
	mock := &mockKeyringProvider{
		items: map[string]keyring.Item{
			keyringKey: {
				Key:  keyringKey,
				Data: []byte("test-key"),
			},
		},
	}
	keyringOpener = func() (KeyringProvider, error) {
		return mock, nil
	}

	status, authenticated := GetAuthStatus()

	if !authenticated {
		t.Error("Expected to be authenticated when keyring has key")
	}

	if status != "System keyring" {
		t.Errorf("Expected status to be 'System keyring', got '%s'", status)
	}
}

func TestGetAuthStatus_NotAuthenticated(t *testing.T) {
	// Save original env var and keyring opener, restore after test
	originalEnv := os.Getenv(envVarName)
	originalOpener := keyringOpener
	defer func() {
		if originalEnv != "" {
			os.Setenv(envVarName, originalEnv)
		} else {
			os.Unsetenv(envVarName)
		}
		keyringOpener = originalOpener
	}()

	// Unset the environment variable
	os.Unsetenv(envVarName)

	// Set up mock keyring with no API key
	mock := &mockKeyringProvider{
		items: make(map[string]keyring.Item),
	}
	keyringOpener = func() (KeyringProvider, error) {
		return mock, nil
	}

	status, authenticated := GetAuthStatus()

	if authenticated {
		t.Error("Expected not to be authenticated when no key found")
	}

	if status != "Not authenticated" {
		t.Errorf("Expected status to be 'Not authenticated', got '%s'", status)
	}
}

func TestSaveAPIKey(t *testing.T) {
	// Save original keyring opener, restore after test
	originalOpener := keyringOpener
	defer func() {
		keyringOpener = originalOpener
	}()

	// Set up mock keyring
	mock := &mockKeyringProvider{
		items: make(map[string]keyring.Item),
	}
	keyringOpener = func() (KeyringProvider, error) {
		return mock, nil
	}

	testAPIKey := "test-api-key-to-save"
	err := SaveAPIKey(testAPIKey)

	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	// Verify the key was saved
	item, ok := mock.items[keyringKey]
	if !ok {
		t.Error("Expected API key to be saved in keyring")
	}

	if string(item.Data) != testAPIKey {
		t.Errorf("Expected saved API key to be '%s', got '%s'", testAPIKey, string(item.Data))
	}
}

func TestSaveAPIKey_KeyringError(t *testing.T) {
	// Save original keyring opener, restore after test
	originalOpener := keyringOpener
	defer func() {
		keyringOpener = originalOpener
	}()

	// Set up mock keyring that returns an error
	mockErr := errors.New("keyring access denied")
	keyringOpener = func() (KeyringProvider, error) {
		return nil, mockErr
	}

	err := SaveAPIKey("test-key")

	if err == nil {
		t.Error("Expected error when keyring fails to open, got nil")
	}

	if err != nil && !contains(err.Error(), "failed to access keyring") {
		t.Errorf("Expected error to contain 'failed to access keyring', got '%s'", err.Error())
	}
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

// contains is a helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 || containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
