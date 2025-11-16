# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

This is a command-line interface for Linear issue tracking, designed for seamless integration with Claude Code and human workflows. The project uses a manual GraphQL client approach (no code generation) with 409 lines of readable, maintainable client code.

## Development Commands

### Building
```bash
go build -o linear
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with verbose output
go test -v ./...

# Run tests for specific package
go test ./internal/client
go test ./internal/auth
go test ./internal/output

# Run specific test
go test -v -run TestName ./internal/package
```

### Installation
```bash
# Install to $GOBIN (typically ~/go/bin)
go install github.com/dukky/linear@latest

# Or build and install locally
go build -o linear
sudo mv linear /usr/local/bin/
```

## Architecture

### Core Design Principles
- **Manual GraphQL Client**: No code generation or build steps - all types and queries are in plain Go
- **Simple HTTP Client**: Direct HTTP client with manual type definitions in `internal/client/graphql.go`
- **Context Timeouts**: All API calls use 30-second context timeouts for reliability
- **Secure Credentials**: API keys stored in system keyring (macOS Keychain, Windows Credential Manager, Linux Secret Service) via 99designs/keyring

### Package Structure

**`cmd/`** - Cobra CLI commands
- `root.go` - Root command and global `--json` flag
- `auth.go` - Authentication commands (login, status)
- `issue.go` - Issue commands (list, view, create)
- `team.go` - Team commands (list)

**`internal/auth/`** - Authentication and credential management
- `auth.go` - API key retrieval with precedence: 1) `LINEAR_API_KEY` env var, 2) system keyring
- `auth_test.go` - Tests for auth functionality
- Uses 99designs/keyring for cross-platform secure storage

**`internal/client/`** - Linear GraphQL API client
- `graphql.go` - Base GraphQL client (`Client.Do()` method)
- `issues.go` - Issue-related queries/mutations and type definitions
- `teams.go` - Team-related queries and type definitions
- Tests: `graphql_test.go`, `issues_test.go`, `teams_test.go`

**`internal/output/`** - Output formatting utilities
- `formatter.go` - Table and JSON output formatters
- `formatter_test.go` - Formatter tests

### GraphQL Client Design

The client in `internal/client/graphql.go` provides a simple `Do()` method:

```go
func (c *Client) Do(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error
```

All GraphQL operations:
1. Create a context with timeout (30 seconds)
2. Call the appropriate client method (`ListIssues`, `GetIssue`, `CreateIssue`, etc.)
3. Client marshals the GraphQL query and variables
4. Makes HTTP POST to `https://api.linear.app/graphql` with Authorization header
5. Unmarshals response into typed structs

Error handling includes both HTTP errors and GraphQL errors from the response.

### Authentication Flow

Priority order in `internal/auth/auth.go`:
1. **Environment Variable**: Check `LINEAR_API_KEY` first (highest priority)
2. **System Keyring**: Fallback to keyring if env var not set
3. **Error**: Return helpful message to run `linear auth login` or set env var

Keyring uses multiple backends in priority order:
- macOS: Keychain
- Windows: Credential Manager
- Linux: Secret Service, KWallet, encrypted file fallback

### Command Structure

All commands follow the pattern:
1. Parse flags/args
2. Create client via `client.NewClient()` (which calls `auth.GetAPIKey()`)
3. Create context with 30-second timeout
4. Call client method
5. Output as JSON (if `--json` flag) or human-readable table

**Global flag**: `--json` (available on all commands)

**Team filtering**: Issues can be filtered by team key using `--team` flag

### Type Definitions

Key types in `internal/client/`:

- **Issue**: Main issue type with nested State, User, Team, Project, Labels
- **Team**: Team with ID, Key, Name, Description
- **User**: Linear user (assignee, creator)
- **State**: Issue state (name, color, type)
- **Project**: Project association
- **Label**: Issue labels

Response wrappers:
- `IssuesResponse` - for listing issues
- `IssueResponse` - for single issue
- `TeamsResponse` - for teams
- `CreateIssueResponse` - for issue creation

## Claude Code Integration

This repository includes `.claude/skills/linear.md` which enables automatic tool calling for Linear operations. When users ask Claude to interact with Linear (list issues, create issues, view details), Claude will automatically invoke the CLI with appropriate commands and parse JSON output.

### Using the Linear CLI from Claude Code

Always use `--json` flag when programmatically parsing output:
- `linear issue list --team ENG --json`
- `linear issue view ENG-123 --json`
- `linear issue create --team ENG --title "..." --description "..." --json`

### Team Keys vs Names
- Commands use **team keys** (short codes like "ENG", "PROD"), not full names
- To find team keys: `linear team list --json`

## Important Implementation Notes

### Context and Timeouts
All API operations use `context.WithTimeout(context.Background(), 30*time.Second)` to prevent hanging on network issues.

### GraphQL Variables
Variables must be properly structured as nested maps for filters:
```go
vars := map[string]interface{}{
    "filter": map[string]interface{}{
        "team": map[string]interface{}{
            "key": map[string]interface{}{
                "eq": teamKey,
            },
        },
    },
}
```

### Issue Creation Flow
Creating issues requires two steps:
1. Get team ID from team key via `GetTeamByKey()`
2. Create issue with team ID via `CreateIssue()`

Team keys (e.g., "ENG") must be resolved to UUIDs before creating issues.

### Output Formatting
- **Table format**: Uses `text/tabwriter` for aligned columns
- **JSON format**: Uses `json.Encoder` with 2-space indentation
- String truncation: Issue titles truncated to 50 chars in table view

## Testing Notes

- Tests use standard Go testing framework
- No external dependencies required for tests
- Auth tests mock environment variables
- Client tests would require mocking or integration with Linear API