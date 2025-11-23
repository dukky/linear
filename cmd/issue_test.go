package cmd

import (
	"testing"
)

func TestIssueCommand(t *testing.T) {
	if issueCmd == nil {
		t.Fatal("issueCmd should not be nil")
	}

	if issueCmd.Use != "issue" {
		t.Errorf("expected Use to be 'issue', got '%s'", issueCmd.Use)
	}

	if issueCmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestIssueCommandHasSubcommands(t *testing.T) {
	commands := issueCmd.Commands()
	if len(commands) == 0 {
		t.Error("issueCmd should have subcommands")
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"list", "view", "create"}
	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}

func TestIssueListCommand(t *testing.T) {
	if issueListCmd == nil {
		t.Fatal("issueListCmd should not be nil")
	}

	if issueListCmd.Use != "list" {
		t.Errorf("expected Use to be 'list', got '%s'", issueListCmd.Use)
	}

	if issueListCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Test flags
	teamFlag := issueListCmd.Flags().Lookup("team")
	if teamFlag == nil {
		t.Error("--team flag should be defined")
	}

	limitFlag := issueListCmd.Flags().Lookup("limit")
	if limitFlag == nil {
		t.Error("--limit flag should be defined")
	}
	if limitFlag.DefValue != "50" {
		t.Errorf("--limit default should be '50', got '%s'", limitFlag.DefValue)
	}

	allFlag := issueListCmd.Flags().Lookup("all")
	if allFlag == nil {
		t.Error("--all flag should be defined")
	}
}

func TestIssueViewCommand(t *testing.T) {
	if issueViewCmd == nil {
		t.Fatal("issueViewCmd should not be nil")
	}

	if issueViewCmd.Use != "view <issue-id>" {
		t.Errorf("expected Use to be 'view <issue-id>', got '%s'", issueViewCmd.Use)
	}

	if issueViewCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Test that it requires exactly 1 argument
	if issueViewCmd.Args == nil {
		t.Error("Args validator should be set")
	}
}

func TestIssueCreateCommand(t *testing.T) {
	if issueCreateCmd == nil {
		t.Fatal("issueCreateCmd should not be nil")
	}

	if issueCreateCmd.Use != "create" {
		t.Errorf("expected Use to be 'create', got '%s'", issueCreateCmd.Use)
	}

	if issueCreateCmd.RunE == nil {
		t.Error("RunE should not be nil")
	}

	// Test flags
	titleFlag := issueCreateCmd.Flags().Lookup("title")
	if titleFlag == nil {
		t.Error("--title flag should be defined")
	}

	descFlag := issueCreateCmd.Flags().Lookup("description")
	if descFlag == nil {
		t.Error("--description flag should be defined")
	}

	teamFlag := issueCreateCmd.Flags().Lookup("team")
	if teamFlag == nil {
		t.Error("--team flag should be defined")
	}
}

func TestIssueCreateCommandValidation(t *testing.T) {
	// Test that the command returns an error when required flags are missing
	tests := []struct {
		name      string
		title     string
		team      string
		wantError bool
	}{
		{
			name:      "missing title",
			title:     "",
			team:      "ENG",
			wantError: true,
		},
		{
			name:      "missing team",
			title:     "Test Issue",
			team:      "",
			wantError: true,
		},
		{
			name:      "both title and team missing",
			title:     "",
			team:      "",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original values
			origTitle := issueTitle
			origTeam := issueTeamID

			// Set test values
			issueTitle = tt.title
			issueTeamID = tt.team

			// Execute command
			err := issueCreateCmd.RunE(issueCreateCmd, []string{})

			// Restore original values
			issueTitle = origTitle
			issueTeamID = origTeam

			if tt.wantError && err == nil {
				t.Errorf("expected error but got none")
			}
		})
	}
}
