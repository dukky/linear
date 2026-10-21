package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/dukky/linear/internal/client"
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
	stubFetchViewer(t, &client.User{ID: "user-1", Name: "Ada Lovelace", Email: "ada@example.com"}, nil)

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

func TestAuthLogout_RemovesAuth(t *testing.T) {
	originalRemoveAPIKey := removeAPIKey
	t.Cleanup(func() {
		removeAPIKey = originalRemoveAPIKey
	})

	removeAPIKey = func() (bool, error) {
		return true, nil
	}

	output := captureStdout(t, func() {
		if err := authLogoutCmd.RunE(authLogoutCmd, nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	if !strings.Contains(output, "Stored key removed from keyring.") {
		t.Fatalf("expected success line, got %q", output)
	}
}

func TestAuthLogout_RemovesAuth_NotSetIdempotent(t *testing.T) {
	originalRemoveAPIKey := removeAPIKey
	t.Cleanup(func() {
		removeAPIKey = originalRemoveAPIKey
	})

	removeAPIKey = func() (bool, error) {
		return false, nil
	}

	output := captureStdout(t, func() {
		if err := authLogoutCmd.RunE(authLogoutCmd, nil); err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
	})

	if !strings.Contains(output, "No API key set, exiting...") {
		t.Fatalf("expected success line, got %q", output)
	}
}

func TestAuthLogout_RemovesAuth_Error(t *testing.T) {
	originalRemoveAPIKey := removeAPIKey
	t.Cleanup(func() {
		removeAPIKey = originalRemoveAPIKey
	})

	removeAPIKey = func() (bool, error) {
		return false, errors.New("Unable to remove API key")
	}

	err := authLogoutCmd.RunE(authLogoutCmd, nil)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !strings.Contains(err.Error(), "error removing API key: Unable to remove API key") {
		t.Fatalf("unexpected error, got %q", err.Error())
	}
}

// stubFetchViewer replaces fetchViewer for the duration of the test and
// resets the package-level jsonOutput flag before and after.
func stubFetchViewer(t *testing.T, user *client.User, err error) {
	t.Helper()
	originalFetchViewer := fetchViewer
	originalJSON := jsonOutput
	jsonOutput = false
	t.Cleanup(func() {
		fetchViewer = originalFetchViewer
		jsonOutput = originalJSON
	})
	fetchViewer = func(ctx context.Context) (*client.User, error) {
		if _, ok := ctx.Deadline(); !ok {
			t.Error("expected fetchViewer to be called with a deadline")
		}
		return user, err
	}
}

func stubAuthStatus(t *testing.T, source string, authenticated bool) {
	t.Helper()
	originalGetAuthStatus := getAuthStatus
	t.Cleanup(func() {
		getAuthStatus = originalGetAuthStatus
	})
	getAuthStatus = func() (string, bool) {
		return source, authenticated
	}
}

func TestAuthStatus_ShowsUser(t *testing.T) {
	stubAuthStatus(t, "System keyring", true)
	stubFetchViewer(t, &client.User{ID: "user-1", Name: "Ada Lovelace", Email: "ada@example.com"}, nil)

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	if !strings.Contains(output, "Status: Authenticated") {
		t.Fatalf("expected authenticated status, got %q", output)
	}
	if !strings.Contains(output, "User:   Ada Lovelace <ada@example.com>") {
		t.Fatalf("expected aligned user line with name and email, got %q", output)
	}
}

func TestAuthStatus_ViewerFetchFailureIsSoft(t *testing.T) {
	stubAuthStatus(t, "System keyring", true)
	stubFetchViewer(t, nil, errors.New("network down"))

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	if !strings.Contains(output, "Status: Authenticated") || !strings.Contains(output, "Source: System keyring") {
		t.Fatalf("expected status and source despite fetch failure, got %q", output)
	}
	if strings.Contains(output, "User:") {
		t.Fatalf("expected no user line on fetch failure, got %q", output)
	}
}

func TestAuthStatus_JSON(t *testing.T) {
	stubAuthStatus(t, "Environment variable (LINEAR_API_KEY)", true)
	stubFetchViewer(t, &client.User{ID: "user-1", Name: "Ada Lovelace", Email: "ada@example.com"}, nil)
	jsonOutput = true

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	var got struct {
		Status string       `json:"status"`
		Source string       `json:"source"`
		User   *client.User `json:"user"`
	}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", output, err)
	}
	if got.Status != "authenticated" {
		t.Errorf("expected status authenticated, got %q", got.Status)
	}
	if got.Source != "Environment variable (LINEAR_API_KEY)" {
		t.Errorf("unexpected source %q", got.Source)
	}
	want := client.User{ID: "user-1", Name: "Ada Lovelace", Email: "ada@example.com"}
	if got.User == nil || *got.User != want {
		t.Errorf("expected user %+v, got %+v", want, got.User)
	}
}

func TestAuthStatus_JSONNotAuthenticated(t *testing.T) {
	stubAuthStatus(t, "Not authenticated", false)
	stubFetchViewer(t, nil, errors.New("must not be called"))
	fetchViewer = func(ctx context.Context) (*client.User, error) {
		t.Error("fetchViewer must not be called when not authenticated")
		return nil, nil
	}
	jsonOutput = true

	output := captureStdout(t, func() {
		authStatusCmd.Run(authStatusCmd, nil)
	})

	var got map[string]interface{}
	if err := json.Unmarshal([]byte(output), &got); err != nil {
		t.Fatalf("expected JSON output, got %q: %v", output, err)
	}
	if got["status"] != "not_authenticated" {
		t.Errorf("expected status not_authenticated, got %v", got["status"])
	}
	if _, ok := got["user"]; ok {
		t.Errorf("expected no user field, got %v", got["user"])
	}
}

func TestResolveListAssignee(t *testing.T) {
	tests := []struct {
		name      string
		assignee  string
		mine      bool
		wantIsMe  bool
		wantEmail string
		wantErr   string
	}{
		{name: "none"},
		{name: "mine", mine: true, wantIsMe: true},
		{name: "me", assignee: "me", wantIsMe: true},
		{name: "ME case-insensitive", assignee: "ME", wantIsMe: true},
		{name: "mine and me", assignee: "me", mine: true, wantIsMe: true},
		{name: "email", assignee: "dev@example.com", wantEmail: "dev@example.com"},
		{name: "mine and email", assignee: "dev@example.com", mine: true, wantErr: "use either --mine or --assignee, not both"},
		{name: "not an email", assignee: "bob", wantErr: "assignee must be an email address or 'me'"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isMe, email, err := resolveListAssignee(tt.assignee, tt.mine)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("expected error containing %q, got %v", tt.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if isMe != tt.wantIsMe || email != tt.wantEmail {
				t.Fatalf("got (isMe=%v, email=%q), want (isMe=%v, email=%q)", isMe, email, tt.wantIsMe, tt.wantEmail)
			}
		})
	}
}
