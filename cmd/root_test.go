package cmd

import (
	"testing"
)

func TestRootCommand(t *testing.T) {
	if rootCmd == nil {
		t.Fatal("rootCmd should not be nil")
	}

	if rootCmd.Use != "linear" {
		t.Errorf("expected Use to be 'linear', got '%s'", rootCmd.Use)
	}

	if rootCmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if rootCmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestRootCommandFlags(t *testing.T) {
	flag := rootCmd.PersistentFlags().Lookup("json")
	if flag == nil {
		t.Fatal("--json flag should be defined")
	}

	if flag.DefValue != "false" {
		t.Errorf("--json flag default value should be 'false', got '%s'", flag.DefValue)
	}
}

func TestRootCommandHasSubcommands(t *testing.T) {
	commands := rootCmd.Commands()
	if len(commands) == 0 {
		t.Error("rootCmd should have subcommands")
	}

	commandNames := make(map[string]bool)
	for _, cmd := range commands {
		commandNames[cmd.Name()] = true
	}

	expectedCommands := []string{"auth", "issue", "team"}
	for _, expected := range expectedCommands {
		if !commandNames[expected] {
			t.Errorf("expected subcommand '%s' not found", expected)
		}
	}
}
