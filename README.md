# Linear CLI

A command-line interface for Linear issue tracking, designed for seamless integration with Claude Code and human workflows.

## Features

- 🔐 Secure API key storage using your system's keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- 📋 List and filter issues by team
- 👁️ View detailed issue information
- ✨ Create new issues
- 🛠️ Update existing issues
- 👥 Manage teams
- 📊 Multiple output formats (human-readable tables and JSON)
- 🤖 Perfect for automation and Claude Code integration

## Installation

### Using `go install` (Recommended)

```bash
go install github.com/dukky/linear@latest
```

This will install the `linear` binary to your `$GOBIN` directory (typically `~/go/bin`). Make sure this directory is in your `PATH`.

### From Source

```bash
git clone https://github.com/dukky/linear.git
cd linear
go build -o linear
sudo mv linear /usr/local/bin/  # Optional: move to PATH
```

## Quick Start

### 1. Authenticate

Get your API key from [Linear Settings](https://linear.app/settings/api) and run:

```bash
linear auth login
```

Follow the prompts to paste your API key. It will be stored securely in your system's keyring.

Alternatively, set the `LINEAR_API_KEY` environment variable:

```bash
export LINEAR_API_KEY=lin_api_...
```

### 2. List Teams

```bash
linear team list
```

### 3. List Issues

```bash
# List all issues
linear issue list

# Filter by team
linear issue list --team ENG
```

### 4. View Issue Details

```bash
linear issue view ENG-123
```

### 5. Create an Issue

```bash
linear issue create --team ENG --title "Fix critical bug" --description "Details here"
```

## Usage

### Authentication Commands

#### `linear auth login`
Interactively store your Linear API key in the system keyring.

```bash
linear auth login
```

#### `linear auth status`
Check your current authentication status.

```bash
linear auth status
```

#### `linear auth logout`
Remove your Linear API key from the system keyring. Note: `LINEAR_API_KEY` environment variable will be unaffected if set

```bash
linear auth logout
```

### Team Commands

#### `linear team list`
List all teams in your workspace.

```bash
# Human-readable table
linear team list

# JSON output
linear team list --json
```

### Issue Commands

#### `linear issue list`
List issues with optional filtering and pagination control.

```bash
# List all issues (default: 50 most recent)
linear issue list

# Filter by team
linear issue list --team ENG

# Limit the number of issues returned
linear issue list --team ENG --limit 10

# Fetch all issues using automatic pagination
linear issue list --team ENG --all

# JSON output for automation
linear issue list --team ENG --json
```

**Pagination options:**
- `--limit N`: Fetch up to N issues (default: 50)
- `--all`: Automatically fetch all issues using cursor-based pagination

The default behavior returns up to 50 issues. Use `--limit` to fetch more or fewer issues in a single request, or use `--all` to automatically fetch all issues across multiple pages.

#### `linear issue view <issue-id>`
View detailed information about a specific issue.

```bash
# Using issue identifier
linear issue view ENG-123

# Using issue UUID
linear issue view <uuid>

# JSON output
linear issue view ENG-123 --json
```

#### `linear issue create`
Create a new issue.

```bash
# Basic issue
linear issue create --team ENG --title "New feature request"

# With description
linear issue create \
  --team ENG \
  --title "Fix bug in authentication" \
  --description "Users are experiencing login failures"

# JSON output
linear issue create --team ENG --title "Bug fix" --json
```

#### `linear issue update <issue-id>`
Update fields on an existing issue.

```bash
# Update title
linear issue update ENG-123 --title "Updated issue title"

# Update priority (0=None, 1=Urgent, 2=High, 3=Medium, 4=Low)
linear issue update ENG-123 --priority 2

# Move issue to another project
linear issue update ENG-123 --project "Mobile App"

# Clear description
linear issue update ENG-123 --description ""

# JSON output
linear issue update ENG-123 --title "Updated issue title" --json
```

## Claude Code Integration

This CLI is designed to work seamlessly with Claude Code through a skill that enables automatic tool calling.

### Install the Claude Code Skill

From the repo, copy the skill to your global Claude skills directory:

```bash
mkdir -p ~/.claude/skills/linear
cp .claude/skills/linear/SKILL.md ~/.claude/skills/linear/
```

Run the same commands to update after pulling new changes.

### Setup

1. **Install the Linear CLI**:
   ```bash
   go install github.com/dukky/linear@latest
   ```

2. **Authenticate**:
   ```bash
   linear auth login
   ```

3. **Install the skill** (see above)

That's it! Claude will now be able to interact with Linear on your behalf.

### What this enables

- Ask Claude to list issues: "Show me all issues in the ENG team"
- Create issues naturally: "Create a bug report for the login issue in PROD"
- Update issues naturally: "Update ENG-123 priority to high and change the title"
- View issue details: "What's the status of ENG-123?"
- Get team information: "List all teams in the workspace"

Claude will automatically invoke the appropriate CLI commands and parse the results for you.

### Manual Usage Examples

You can also use the CLI directly:

#### Example 1: Working on a Linear ticket

```bash
# View the issue details
linear issue view ENG-123

# Claude Code can then read the issue and work on it
# The JSON output is particularly useful for programmatic access
linear issue view ENG-123 --json
```

#### Example 2: Creating tickets from Claude Code

```bash
# Claude Code can create tickets for new bugs or features
linear issue create \
  --team ENG \
  --title "Implement user authentication" \
  --description "Add JWT-based authentication system" \
  --json
```

#### Example 3: Listing issues for a sprint

```bash
# Get all issues for a specific team
linear issue list --team ENG --json

# Claude Code can parse this and help prioritize or work on them
```

## Configuration

### Environment Variables

- `LINEAR_API_KEY`: Your Linear API key (alternative to using `linear auth login`)

### Authentication Priority

The CLI checks for credentials in the following order:

1. `LINEAR_API_KEY` environment variable
2. System keyring (set via `linear auth login`)

## Output Formats

All issue, team, and project commands support both human-readable output (default) and JSON output (with `--json` flag).

### Human-Readable Output (Default)

```bash
$ linear issue list --team ENG

ID       TITLE                  STATUS       ASSIGNEE      PRIORITY
------   --------------------   ----------   -----------   --------
ENG-123  Fix login bug          In Progress  John Doe      High
ENG-124  Add new feature        Todo         Jane Smith    Medium
```

### JSON Output

```bash
$ linear issue list --team ENG --json
```

Returns structured JSON data perfect for automation and scripting.

## Examples

### Example Workflow: Bug Triage

```bash
# 1. List all issues for the engineering team
linear issue list --team ENG

# 2. View a specific issue
linear issue view ENG-123

# 3. Create a new issue for a bug you found
linear issue create \
  --team ENG \
  --title "Memory leak in user service" \
  --description "The user service shows increasing memory usage over time"
```

### Example: Automation Script

```bash
#!/bin/bash

# Get all issues as JSON (fetches all pages automatically)
issues=$(linear issue list --team ENG --all --json)

# Process with jq or other tools
echo "$issues" | jq '.[] | select(.priority > 2)'
```

### Example: Working with Large Issue Lists

```bash
# Get the first 10 issues for quick overview
linear issue list --team ENG --limit 10

# Get all issues when you need the complete list
linear issue list --team ENG --all

# Combine with JSON for processing large datasets
linear issue list --all --json | jq 'length'  # Count total issues
```

## Development

### Prerequisites

- Go 1.21 or later
- Linear API key

### Building from Source

```bash
# Clone the repository
git clone https://github.com/dukky/linear.git
cd linear

# Install dependencies
go mod download

# Build
go build -o linear

# Run
./linear --help
```

## Architecture

- **CLI Framework**: Cobra
- **GraphQL Client**: Simple HTTP client with manual type definitions
- **Secure Storage**: 99designs/keyring (cross-platform keyring access)
- **API**: Linear GraphQL API

### Design Philosophy

This CLI uses a simple, manual GraphQL client approach instead of code generation:
- **409 lines** of readable, maintainable client code
- **No build steps** - just `go build` and you're done
- **Full control** over JSON serialization and error handling
- **Easy to modify** - all types and queries are in plain Go
- **No dependencies** on schema files or code generators

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT

## Support

For issues and feature requests, please open an issue on GitHub.

## Links

- [Linear API Documentation](https://developers.linear.app/docs/graphql/working-with-the-graphql-api)
- [Linear GraphQL Schema](https://github.com/linear/linear/blob/master/packages/sdk/src/schema.graphql)
- [Claude Code](https://claude.com/claude-code)
