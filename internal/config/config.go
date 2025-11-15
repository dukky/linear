package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// OAuth endpoints for Linear
	LinearAuthURL  = "https://linear.app/oauth/authorize"
	LinearTokenURL = "https://api.linear.app/oauth/token"
	LinearAPIURL   = "https://api.linear.app/graphql"

	// Local redirect for OAuth
	RedirectURL  = "http://127.0.0.1:8793/callback"
	RedirectPort = "8793"
)

type Config struct {
	ConfigDir string
	TokenFile string
}

type TokenData struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token,omitempty"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in,omitempty"`
}

// New creates a new config instance
func New() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	configDir := filepath.Join(home, ".linear")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	return &Config{
		ConfigDir: configDir,
		TokenFile: filepath.Join(configDir, "tokens.enc"),
	}, nil
}

// SaveToken securely saves the token data with encryption
func (c *Config) SaveToken(token *TokenData) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Generate encryption key from machine-specific data
	key := c.deriveKey()

	// Encrypt the token data
	encrypted, err := encrypt(data, key)
	if err != nil {
		return fmt.Errorf("failed to encrypt token: %w", err)
	}

	// Write to file with restricted permissions
	if err := os.WriteFile(c.TokenFile, encrypted, 0600); err != nil {
		return fmt.Errorf("failed to write token file: %w", err)
	}

	return nil
}

// LoadToken loads and decrypts the stored token
func (c *Config) LoadToken() (*TokenData, error) {
	encrypted, err := os.ReadFile(c.TokenFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("no token found, please authenticate first")
		}
		return nil, fmt.Errorf("failed to read token file: %w", err)
	}

	// Generate decryption key
	key := c.deriveKey()

	// Decrypt the token data
	data, err := decrypt(encrypted, key)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt token: %w", err)
	}

	var token TokenData
	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("failed to unmarshal token: %w", err)
	}

	return &token, nil
}

// ClearToken removes the stored token
func (c *Config) ClearToken() error {
	if err := os.Remove(c.TokenFile); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove token file: %w", err)
	}
	return nil
}

// deriveKey creates an encryption key from machine-specific data
func (c *Config) deriveKey() []byte {
	// Use hostname as salt (machine-specific)
	hostname, _ := os.Hostname()
	salt := []byte(hostname + "-linear-cli")

	// Use user's home directory path as additional entropy
	home, _ := os.UserHomeDir()
	password := []byte(home + "-linear-token-key")

	// Derive a 32-byte key using PBKDF2
	return pbkdf2.Key(password, salt, 100000, 32, sha256.New)
}

// encrypt encrypts data using AES-GCM
func encrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return []byte(base64.StdEncoding.EncodeToString(ciphertext)), nil
}

// decrypt decrypts data using AES-GCM
func decrypt(encoded, key []byte) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(string(encoded))
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
