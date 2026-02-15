---
name: linear
description: Interact with Linear issues and teams using the Linear CLI
tags: [linear, issues, project-management, productivity]
---

# Linear CLI Skill

Manage Linear issues, teams, and projects from the command line.

**Always use `--json` flag** for programmatic parsing. Use `linear --help` or `linear <command> --help` to discover available commands and flags.

## Quick Reference

| Action | Command |
|--------|---------|
| List issues | `linear issue list [--team KEY] [--project NAME] [--limit N] [--all] [--json]` |
| View issue | `linear issue view ID [--json]` |
| Create issue | `linear issue create --team KEY --title "..." [--description "..."] [--project "..."] [--assignee "user@example.com"] [--json]` |
| Update issue | `linear issue update ID [--title "..."] [--description "..."] [--priority 0-4] [--project "..."] [--json]` |
| List teams | `linear team list [--json]` |
| List projects | `linear project list [--team KEY] [--json]` |
| Auth status | `linear auth status` |

## Key Concepts

**Team keys**: Short codes like `ENG`, `PROD` (not full names). Find with `linear team list`.

**Issue IDs**: Use human-readable IDs (`ENG-123`) or UUIDs.

**Project IDs**: Use project name (`"Mobile App"`) or UUID. Names are matched case-insensitively.

**JSON output**: Add `--json` to any command for structured output. Use for parsing/analysis.

**Pagination**: Default limit is 50 issues. Use `--limit N` for custom count, `--all` for complete list.

## Data Models

### Issue
```json
{"id": "uuid", "identifier": "ENG-123", "title": "...", "description": "...", "priority": 1,
 "state": {"name": "In Progress", "type": "started"},
 "assignee": {"name": "...", "email": "..."},
 "team": {"key": "ENG", "name": "Engineering"},
 "project": {"id": "uuid", "name": "Q1 Goals"},
 "labels": {"nodes": [{"name": "bug"}]}}
```

### Team
```json
{"id": "uuid", "key": "ENG", "name": "Engineering", "description": "..."}
```

### Project
```json
{"id": "uuid", "name": "Mobile App"}
```

## Common Patterns

**List issues by team:**
```bash
linear issue list --team ENG --json
```

**List issues by project:**
```bash
linear issue list --project "Mobile App" --all --json
```

**Create issue in project:**
```bash
linear issue create --team ENG --title "Add feature" --project "Mobile App" --json
```

**Update issue title and priority:**
```bash
linear issue update ENG-123 --title "Refine onboarding copy" --priority 2 --json
```

**Move issue to another project:**
```bash
linear issue update ENG-123 --project "Q2 Goals" --json
```

**View issue details:**
```bash
linear issue view ENG-123 --json
```

**Count issues (fetch all, count array):**
```bash
linear issue list --team ENG --all --json | jq length
```

## Reference Values

**state.type**: `triage`, `backlog`, `unstarted`, `started`, `completed`, `canceled`

**priority**: 0=None, 1=Urgent, 2=High, 3=Medium, 4=Low

## Troubleshooting

**"Failed to get Linear API key"**
- Run `linear auth login` or set `LINEAR_API_KEY` env var

**"Need to change API key"**
- Run `linear auth logout` to remove and then `linear auth login`, or set new `LINEAR_API_KEY` env var

**"Team not found"**
- Use team key (e.g., `ENG`), not full name. Check `linear team list`

**"Issue not found"**
- Verify identifier format (`ENG-123` or UUID)

**"Error fetching project"**
- Check available projects: `linear project list --team KEY`
- Project names are matched case-insensitively
- If multiple projects match, use exact name or UUID to disambiguate
