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
- `issue.go` - Issue commands (list, view, create, update, comments, comment)
- `team.go` - Team commands (list)
- `project.go` - Project commands (list, view)

**`internal/auth/`** - Authentication and credential management
- `auth.go` - API key retrieval with precedence: 1) `LINEAR_API_KEY` env var, 2) system keyring
- `auth_test.go` - Tests for auth functionality
- Uses 99designs/keyring for cross-platform secure storage

**`internal/client/`** - Linear GraphQL API client
- `graphql.go` - Base GraphQL client (`Client.Do()` method)
- `issues.go` - Issue-related queries/mutations and type definitions
- `teams.go` - Team-related queries and type definitions
- `projects.go` - Project queries (list, by team, lookup by name or UUID) sharing the `fullProjectSelection` field set
- `states.go` - Workflow state lookup (`GetTeamStates`, `GetStateByName`)
- `users.go` - User lookup (assignee resolution)
- Tests: `graphql_test.go`, `issues_test.go`, `teams_test.go`, `projects_test.go`, `states_test.go`, `users_test.go`

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

`Do()` also applies a per-request timeout (`DefaultRequestTimeout`, 30s, held in the package var `requestTimeout` so tests can shorten it) by deriving a child context for each HTTP request. A caller-supplied deadline still applies. `ListAllIssues` pages at `issuePageSize` (100) per request and honours `ListIssuesOptions.MaxIssues` (0 = unlimited), stopping and trimming once that many issues are collected. `issue list` routes both `--all` and `--limit` through it and uses a context with no overall deadline.

`ListIssuesOptions` filter fields all go into one `IssueFilter` object built in `ListIssues` (so they survive the `ListAllIssues` path): `TeamKey` → `team.key.eq`, `ProjectID` → `project.id.eq`, `AssigneeIsMe` → `assignee.isMe.eq: true` (takes precedence over email), `AssigneeEmail` → `assignee.email.eq`, `StateName` → `state.name.eqIgnoreCase`, `StateTypes` → `state.type.in` (name and type combine in the same `state` object). `ListIssues` validates `StateTypes` with `ValidateStateTypes` against `ValidStateTypes` (triage, backlog, unstarted, started, completed, canceled, duplicate; trimmed and lowercased) before sending any request, returning an error that names the bad value and lists the valid ones. `issue list` exposes these as `--assignee <email|me>`, `--mine`, `--state`, and `--state-type` (StringSlice), validating them (via `resolveListAssignee` and `client.ValidateStateTypes`) before creating a client.

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

- **Issue**: Main issue type with nested State, User, Team, Project, Parent, Labels, Children, Relations (outgoing edges: this issue is `issue`), InverseRelations (incoming edges: this issue is `relatedIssue`). `Parent` is `*IssueRef` without omitempty (`null` = no parent). `Children` (`*IssueConnection`), `Relations` and `InverseRelations` (`*IssueRelationConnection`) are pointers with `omitempty`: nil and absent from JSON when not selected (`issue list`), so "not fetched" differs from "none" (empty `nodes`)
- **IssueRef**: Lightweight issue summary (id, identifier, title, url) used for `Parent`, `Children.Nodes` and relation endpoints, so link selections never recurse
- **IssueRelation**: Relation with id, type (`RelationTypeBlocks` / `RelationTypeRelated`), `Issue` and `RelatedIssue` refs; for blocks, `Issue` blocks `RelatedIssue`
- **Team**: Team with ID, Key, Name, Description
- **User**: Linear user (assignee, creator)
- **State**: Issue state (name, color, type)
- **Project**: Project with ID, Name, URL, Description, Status (`ProjectStatus`: name, color, type). Description/Status are `omitempty`: the project nested in issue JSON only selects id/name/url, while project queries select the full `fullProjectSelection`
- **Label**: Issue labels

Response wrappers:
- `IssuesResponse` - for listing issues
- `IssueResponse` - for single issue
- `TeamsResponse` - for teams
- `CreateIssueResponse` - for issue creation
- `CreateIssueRelationResponse` - for relation creation (issueRelationCreate mutation; payload field is `issueRelation`)
- `CommentsResponse` - for listing comments (Issue is a pointer — nil means issue not found)
- `CreateCommentResponse` - for creating comments (commentCreate mutation)

## Claude Code Integration

This repository includes `.claude/skills/linear/SKILL.md` which enables automatic tool calling for Linear operations. When users ask Claude to interact with Linear (list issues, create issues, view details), Claude will automatically invoke the CLI with appropriate commands and parse JSON output.

### Using the Linear CLI from Claude Code

Always use `--json` flag when programmatically parsing output:
- `linear issue list --team ENG --json`
- `linear issue list --mine --state-type started,unstarted --json` (what's assigned to me)
- `linear auth status --json` (includes the authenticated `user` {id, name, email} when reachable)
- `linear issue view ENG-123 --json`
- `linear issue create --team ENG --title "..." --description "..." --json`
- `linear issue create --team ENG --title "..." --priority 2 --json`
- `linear issue update ENG-123 --title "..." --priority 2 --json`
- `linear issue update ENG-123 --state "In Progress" --json`
- `linear issue create --team ENG --title "..." --parent ENG-100 --json` (sub-issue)
- `linear issue update ENG-123 --parent ENG-100 --blocks ENG-124 --blocked-by ENG-120 --related ENG-130 --json`
- `linear issue comments ENG-123 --json`
- `linear issue comment ENG-123 "message" --json`
- `linear project list [--team ENG] --json`
- `linear project view "Mobile App" [--team ENG] --json`

### Team Keys vs Names
- Commands use **team keys** (short codes like "ENG", "PROD"), not full names
- To find team keys: `linear team list --json`

## Important Implementation Notes

### Context and Timeouts
All API operations use `context.WithTimeout(context.Background(), 30*time.Second)` to prevent hanging on network issues, except `issue list`, which uses `context.WithCancel` with no overall deadline so paginated fetches are not cut off; there, `Client.Do()`'s per-request `DefaultRequestTimeout` bounds each request.

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

### Auth Status and the Viewer
`auth status` works offline: it prints Status/Source, then best-effort calls `Client.GetViewer` (`query { viewer { id name email } }`) under a 10s timeout and prints `User:   Name <email>`. On failure it writes `Could not fetch user: ...` to stderr and still exits 0. `--json` prints `{status, source, user}` (`status` is `authenticated` or `not_authenticated`; `userError` is set when the lookup failed). The lookup is behind the package var `fetchViewer` in `cmd/auth.go` so tests stub it, like `getAuthStatus`.

### Issue Creation Flow
Creating issues requires two steps:
1. Get team ID from team key via `GetTeamByKey()`
2. Create issue with team ID via `CreateIssue()`

Team keys (e.g., "ENG") must be resolved to UUIDs before creating issues.

`create --parent <id>` resolves the parent (identifier or UUID) with `GetIssue()` and sends its UUID as `CreateIssueInput.ParentID`, since `parentId` does not accept identifiers.

`create` accepts `--priority` (0-4) alongside `--title`, `--description`, `--project`, `--assignee`, and `--state`. Like `update`, priority uses `cmd.Flags().Changed("priority")` to distinguish "unset" from an explicit `0` ("No priority"): `CreateIssueInput.Priority` is `*int` (`json:"priority,omitempty"`) and is only populated when the flag was passed.

### Issue Update Flow
Updating issues may involve project, state, parent and relation resolution:
1. Build update input from changed flags (`title`, `description`, `priority`, `project`, `assignee`, `state`, `parent`)
2. If `project`, `state` or any relation flag is set, fetch issue via `GetIssue()` once: its team scopes the project/state lookups (which may differ from any `--team` used elsewhere) and its UUID is the relation endpoint
3. Resolve project using `GetProjectByIdentifier()`; resolve state using `GetStateByName()` (case-insensitive match against `team.states`); resolve `--parent` to a UUID via `GetIssue()` into `UpdateIssueInput.ParentID`
4. Resolve relation targets via `buildRelationInputs()` (pure, unit-tested): `--blocks X` → `{issueId: current, relatedIssueId: X, type: blocks}`, `--blocked-by X` → the same with endpoints swapped, `--related X` → `{current, X, related}`. Order: all blocks, then blocked-by, then related. All targets resolve before any write
5. Update issue via `UpdateIssue()` only if an issueUpdate field changed; a relations-only invocation sends no update mutation
6. Create each relation via `CreateIssueRelation()`; `success == false` or a nil `issueRelation` is an error
7. With `--json`, if relations were created, re-fetch via `GetIssue()` so the output includes them

The relation flags are `StringArray` (repeat-only, no comma splitting); empty values are rejected during validation, before any client call. Relations are only added, never removed. A `--blocked-by` edge is stored on the blocking issue, so it reads back under the current issue's `inverseRelations`.

`issue update`'s logic lives in `runIssueUpdate(cmd, args) error`; `Run` prints the returned error (whose text is the full user-facing message) and exits 1, so cmd tests call `runIssueUpdate` directly against the `stubLinearAPI` transport stub in `cmd/issue_test.go`. `buildRelationInputs` returns `[]relationRequest`, pairing each input with the identifier as typed so relation errors name `ENG-2`, not a UUID.

`issue view` text renders both connections through the pure, nil-safe `formatRelations(outgoing, incoming *IssueRelationConnection)`, joined with `, `: outgoing `blocks X` / `related X`, incoming `blocked by X` / `related X`, other types `<type> X` / `<type> from X`, deduplicated by relation ID.

### Issue Link Selections
`fullIssueSelection` includes `parent` (one cheap ref), so every issue output has it, including `issue list`. `children(first: 50)`, `relations(first: 50)` and `inverseRelations(first: 50)` live in `issueLinksSelection`; `fullIssueWithLinksSelection` (= both selections) is used by `GetIssue`, `CreateIssue` and `UpdateIssue` only. `ListIssues` keeps `fullIssueSelection`, because three nested connections per issue would turn a 100-issue page into 300 nested connections, which made list pagination fragile (#18).

### State Resolution
Workflow state names are team-scoped, so resolving a `--state` flag to a `stateId` requires the team the issue belongs to (for `create`, the `--team` being created into; for `update`, the issue's own team):
1. `GetTeamStates()` queries `team(id: ...).states.nodes` for `{id, name, type}`
2. `GetStateByName()` matches the provided name case-insensitively; if no match, the error lists the team's valid state names

### Comment Resolution Flow
Creating a comment on an issue requires resolving the human-readable identifier to a UUID:
1. `GetIssue()` resolves the identifier (e.g., "ENG-123") to a UUID
2. `CreateComment()` passes the UUID in `CommentCreateInput.IssueId`

This is necessary because `issueId` in the mutation input does not accept human identifiers — only UUIDs.

### Output Formatting
- **Table format**: Uses `text/tabwriter` for aligned columns
- **JSON format**: Uses `json.Encoder` with 2-space indentation
- String truncation: Issue titles truncated to 50 chars in table view

## Testing Notes

- Tests use standard Go testing framework
- No external dependencies required for tests
- Auth tests mock environment variables
- Client tests would require mocking or integration with Linear API
