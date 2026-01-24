#!/bin/bash
# Install or update the Linear CLI skill for Claude Code

set -e

SKILL_DIR="$HOME/.claude/skills/linear"
SKILL_URL="https://raw.githubusercontent.com/dukky/linear-cli/master/.claude/skills/linear/SKILL.md"

mkdir -p "$SKILL_DIR"

if [ -f "$SKILL_DIR/SKILL.md" ]; then
    echo "Updating existing skill..."
else
    echo "Installing skill..."
fi

curl -fsSL "$SKILL_URL" -o "$SKILL_DIR/SKILL.md"

echo "Done. Skill installed at $SKILL_DIR/SKILL.md"
