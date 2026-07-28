package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

// embeddedSkillContent holds the SKILL.md bytes injected by main via SetSkillContent.
var embeddedSkillContent []byte

// SetSkillContent is called from main to inject the embedded skill file content.
// It must be called before Execute().
func SetSkillContent(content []byte) {
	embeddedSkillContent = content
}

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Install the Linear Claude Code skill to ~/.claude/skills/linear/",
	Long: `Install the Linear Claude Code skill to your global Claude skills directory.

This enables Claude to automatically invoke linear CLI commands when you ask
it to interact with Linear (list issues, create issues, view details, etc.).

Run this command once after installing the linear CLI, and again after
updating to pick up any skill improvements.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runSetup()
	},
}

func init() {
	rootCmd.AddCommand(setupCmd)
}

func runSetup() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("could not determine home directory: %w", err)
	}

	skillDir := filepath.Join(home, ".claude", "skills", "linear")
	if err := os.MkdirAll(skillDir, 0755); err != nil {
		return fmt.Errorf("could not create skill directory %s: %w", skillDir, err)
	}

	skillPath := filepath.Join(skillDir, "SKILL.md")

	action := "Installed"
	if _, err := os.Stat(skillPath); err == nil {
		action = "Updated"
	}

	if err := os.WriteFile(skillPath, embeddedSkillContent, 0644); err != nil {
		return fmt.Errorf("could not write skill file %s: %w", skillPath, err)
	}

	fmt.Printf("✓ %s Linear Claude Code skill at %s\n", action, skillPath)
	fmt.Println()
	fmt.Println("Claude will now be able to interact with Linear on your behalf.")
	fmt.Println(`Try asking: "List my Linear issues" or "Create a bug report in ENG"`)
	return nil
}
