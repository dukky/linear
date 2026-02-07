package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	originalStdout := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	os.Stdout = originalStdout

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("failed to close reader: %v", err)
	}

	return buf.String()
}

func TestAuthStatus_Authenticated(t *testing.T) {
	originalGetAuthStatus := getAuthStatus
	t.Cleanup(func() {
		getAuthStatus = originalGetAuthStatus
	})

	getAuthStatus = func() (string, bool) {
		return "Environment variable (LINEAR_API_KEY)", true
	}

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	if !strings.Contains(output, "Status: Authenticated") {
		t.Fatalf("expected authenticated status, got %q", output)
	}
	if !strings.Contains(output, "Source: Environment variable (LINEAR_API_KEY)") {
		t.Fatalf("expected source line, got %q", output)
	}
}

func TestAuthStatus_UnauthenticatedShowsDetails(t *testing.T) {
	originalGetAuthStatus := getAuthStatus
	t.Cleanup(func() {
		getAuthStatus = originalGetAuthStatus
	})

	getAuthStatus = func() (string, bool) {
		return "Error accessing keyring: backend unavailable", false
	}

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	if !strings.Contains(output, "Status: Not authenticated") {
		t.Fatalf("expected unauthenticated status, got %q", output)
	}
	if !strings.Contains(output, "Details: Error accessing keyring: backend unavailable") {
		t.Fatalf("expected details line, got %q", output)
	}
}
