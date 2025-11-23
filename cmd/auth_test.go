package cmd

import (
	"testing"
)

func TestAuthCommand(t *testing.T) {
	if authCmd == nil {
		t.Fatal("authCmd should not be nil")
	}

	if authCmd.Use != "auth" {
		t.Errorf("expected Use to be 'auth', got '%s'", authCmd.Use)
	}

	if authCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestAuthCommandHasSubcommands(t *testing.T) {
	commands := authCmd.Commands()
	if len(commands) == 0 {
		t.Error("authCmd should have subcommands")
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"login", "status"}
	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestAuthLoginCommand(t *testing.T) {
	if authLoginCmd == nil {
		t.Fatal("authLoginCmd should not be nil")
	}

	if authLoginCmd.Use != "login" {
		t.Errorf("expected Use to be 'login', got '%s'", authLoginCmd.Use)
	}

	if authLoginCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if authLoginCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}
}

func TestAuthStatusCommand(t *testing.T) {
	if authStatusCmd == nil {
		t.Fatal("authStatusCmd should not be nil")
	}

	if authStatusCmd.Use != "status" {
		t.Errorf("expected Use to be 'status', got '%s'", authStatusCmd.Use)
	}

	if authStatusCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if authStatusCmd.Run == nil {
		t.Error("Run should not be nil")
	}
}
