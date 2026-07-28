package main

import (
	_ "embed"

	"github.com/dukky/linear/cmd"
)

//go:embed .claude/skills/linear/SKILL.md
var skillContent []byte

func main() {
	cmd.SetSkillContent(skillContent)
	cmd.Execute()
}
