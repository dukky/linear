# Linear CLI

A command-line interface for Linear issue tracking, designed for seamless integration with Claude Code and human workflows.

## Features

- 🔐 Secure API key storage using your system's keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service)
- 📋 List and filter issues by team
- 👁️ View detailed issue information
- ✨ Create new issues
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
cd linear-cli
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
List issues with optional filtering.

```bash
# List all issues
linear issue list

# Filter by team
linear issue list --team ENG

# JSON output for automation
linear issue list --team ENG --json
```

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

## Claude Code Integration

This CLI is designed to work seamlessly with Claude Code. Here's an example workflow:

### Example 1: Working on a Linear ticket

```bash
# View the issue details
linear issue view ENG-123

# Claude Code can then read the issue and work on it
# The JSON output is particularly useful for programmatic access
linear issue view ENG-123 --json
```

### Example 2: Creating tickets from Claude Code

```bash
# Claude Code can create tickets for new bugs or features
linear issue create \
  --team ENG \
  --title "Implement user authentication" \
  --description "Add JWT-based authentication system" \
  --json
```

### Example 3: Listing issues for a sprint

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

All list and view commands support both human-readable table output (default) and JSON output (with `--json` flag).

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

# Get all high-priority issues as JSON
issues=$(linear issue list --team ENG --json)

# Process with jq or other tools
echo "$issues" | jq '.[] | select(.priority > 2)'
```

## Security

### Keychain Access on macOS

The Linear CLI stores your API key in the system keychain with "allow all applications" access on macOS. This design choice prevents repeated password prompts when rebuilding the tool during development (each rebuild changes the binary hash, which would normally require re-authorization).

**Security Trade-offs:**
- ✅ **Convenient**: No password prompts on every rebuild during development
- ⚠️ **Less Restrictive**: Any application on your Mac can access the stored API key

**Alternative Options:**

If you prefer stricter security:

1. **Use Environment Variable**: Set `LINEAR_API_KEY` instead of storing in keychain
   ```bash
   export LINEAR_API_KEY=lin_api_...
   ```

2. **Manual Keychain Configuration**: After running `linear auth login`, open Keychain Access.app and manually configure access control for the "Linear API Key" item

3. **Code Signing**: Sign the binary with an Apple Developer certificate to avoid re-prompts while maintaining strict keychain access control

For most users, the default keychain storage provides a good balance of security and usability for a development tool.

## Development

### Prerequisites

- Go 1.21 or later
- Linear API key

### Building from Source

```bash
# Clone the repository
git clone https://github.com/dukky/linear.git
cd linear-cli

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
