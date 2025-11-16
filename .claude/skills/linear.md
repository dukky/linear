---
description: Interact with Linear issues and teams using the Linear CLI
tags: [linear, issues, project-management, productivity]
---

# Linear CLI Skill

This skill enables you to interact with Linear (the project management tool) to manage issues, teams, and workflows directly from Claude Code.

## Prerequisites

1. Linear CLI must be installed:
   ```bash
   go install github.com/dukky/linear@latest
   ```

2. Authentication must be configured:
   ```bash
   linear auth login
   ```
   Or set the `LINEAR_API_KEY` environment variable.

## When to Use This Skill

Use the Linear CLI when the user asks you to:
- List, view, or search for Linear issues
- Create new issues in Linear
- Check issue status or details
- List teams in the workspace
- Manage Linear workspace resources

## Available Commands

### Authentication

#### Check auth status
```bash
linear auth status
```
Shows which API key is being used (environment variable or keychain).

### Team Management

#### List all teams
```bash
linear team list
```
Shows teams in human-readable table format.

#### List teams as JSON
```bash
linear team list --json
```
Returns structured JSON data for programmatic use.

### Issue Management

#### List all issues
```bash
linear issue list
```
Shows up to 50 most recent issues across all teams.

#### List issues for a specific team
```bash
linear issue list --team ENG
```
Filter issues by team key (e.g., ENG, PROD, DESIGN).

#### List issues as JSON
```bash
linear issue list --json
linear issue list --team ENG --json
```
Returns structured JSON data.

#### View issue details
```bash
linear issue view ENG-123
linear issue view <issue-uuid>
```
Shows detailed information about a specific issue. Accepts either:
- Issue identifier (e.g., ENG-123)
- Issue UUID

#### View issue as JSON
```bash
linear issue view ENG-123 --json
```
Returns structured JSON data.

#### Create a new issue
```bash
linear issue create --team ENG --title "Fix login bug"
linear issue create --team ENG --title "Add feature" --description "Detailed description here"
```
Creates a new issue in the specified team.

**Required parameters:**
- `--team TEAM_KEY`: The team where the issue should be created
- `--title "TITLE"`: The issue title

**Optional parameters:**
- `--description "DESC"`: Issue description (supports markdown)

#### Create issue and get JSON response
```bash
linear issue create --team ENG --title "New feature" --json
```

## Output Formats

All commands support two output formats:

1. **Human-readable table** (default): Easy to read in terminal
2. **JSON format** (with `--json` flag): Structured data for parsing

**Best Practices:**
- Use JSON format when you need to parse or analyze the data
- Use table format when displaying results directly to the user

## Data Models

### Issue Object (JSON)
```json
{
  "id": "uuid",
  "identifier": "ENG-123",
  "title": "Issue title",
  "description": "Issue description",
  "priority": 1,
  "state": {
    "name": "In Progress",
    "color": "#f2c94c",
    "type": "started"
  },
  "assignee": {
    "name": "John Doe",
    "email": "john@example.com"
  },
  "team": {
    "key": "ENG",
    "name": "Engineering"
  },
  "project": {
    "name": "Q1 Goals"
  },
  "labels": [
    {
      "name": "bug",
      "color": "#eb5757"
    }
  ],
  "creator": {
    "name": "Jane Smith"
  }
}
```

### Team Object (JSON)
```json
{
  "id": "uuid",
  "key": "ENG",
  "name": "Engineering",
  "description": "Engineering team"
}
```

## Usage Examples

### Example 1: List issues for a team
```bash
linear issue list --team ENG --json
```

When the user asks: "Show me all issues in the Engineering team"
- Use the `linear issue list --team ENG --json` command
- Parse the JSON output
- Present the results in a readable format

### Example 2: Create an issue
```bash
linear issue create --team PROD --title "Update documentation" --description "Add API examples to the docs" --json
```

When the user asks: "Create a new issue in PROD to update the documentation"
- Use the `linear issue create` command with appropriate parameters
- Include relevant details in title and description
- Return the created issue details

### Example 3: View issue details
```bash
linear issue view ENG-123 --json
```

When the user asks: "What's the status of ENG-123?"
- Use the `linear issue view` command
- Parse the JSON to extract relevant information (state, assignee, priority)
- Present the key details clearly

### Example 4: Search for specific issues
```bash
linear issue list --team ENG --json
```

When the user asks: "Find all bugs in the Engineering team"
- List issues for the team using `--json` flag
- Filter the results by looking at labels or title
- Present matching issues

## Important Notes

1. **Authentication**: Always check auth status first if you encounter authentication errors
2. **Team Keys**: Team keys are short codes (e.g., ENG, PROD), not full team names
3. **Issue Identifiers**: Can use either human-readable IDs (ENG-123) or UUIDs
4. **JSON Parsing**: When using `--json`, always parse the output as JSON for accurate data extraction
5. **Rate Limits**: Be mindful of API rate limits when making multiple requests
6. **Error Handling**: If a command fails, check the error message and suggest solutions

## Troubleshooting

### "Failed to get Linear API key"
- Check auth status: `linear auth status`
- Login again: `linear auth login`
- Or set environment variable: `export LINEAR_API_KEY=your_key`

### "Team not found"
- List available teams: `linear team list`
- Use the correct team key (short code, not full name)

### "Issue not found"
- Verify the issue identifier is correct
- Check if you have access to that issue/team

## Best Practices for Claude Code

1. **Use JSON by default** when you need to parse data or extract specific information
2. **Use table format** when displaying results directly to the user without processing
3. **Always validate team keys** by listing teams first if unsure
4. **Provide context** when creating issues - include relevant description
5. **Handle errors gracefully** and suggest solutions to the user
6. **Batch operations carefully** - Linear CLI returns up to 50 issues at a time
7. **Quote strings** with spaces in bash commands (e.g., `--title "Multi word title"`)

## Security Notes

- API keys are stored securely in system keychain (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- Environment variable `LINEAR_API_KEY` takes precedence over keychain
- Never expose API keys in output or logs
