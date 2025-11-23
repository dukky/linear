package cmd

import (
	"testing"
)

func TestTeamCommand(t *testing.T) {
	if teamCmd == nil {
		t.Fatal("teamCmd should not be nil")
	}

	if teamCmd.Use != "team" {
		t.Errorf("expected Use to be 'team', got '%s'", teamCmd.Use)
	}

	if teamCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestTeamCommandHasSubcommands(t *testing.T) {
	commands := teamCmd.Commands()
	if len(commands) == 0 {
		t.Error("teamCmd should have subcommands")
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"list"}
	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestTeamListCommand(t *testing.T) {
	if teamListCmd == nil {
		t.Fatal("teamListCmd should not be nil")
	}

	if teamListCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got '%s'", teamListCmd.Use)
	}

	if teamListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	if teamListCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}
