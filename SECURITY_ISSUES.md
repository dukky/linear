# Security & Bug Issues for Linear CLI

## Issue 1: API Key Exposure in Error Messages (CRITICAL SECURITY)

**Priority**: Urgent
**Type**: Security Bug
**File**: `internal/client/graphql.go:79`

### Description
When HTTP requests fail, the entire response body is logged in error messages. If the API server includes request details or headers in error responses, the API key could be exposed in error messages.

### Current Code
```go
body, _ := io.ReadAll(resp.Body)
return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(body))
```

### Risk
- API keys could leak through error messages
- Error messages may be logged to files, sent to error tracking services, or displayed in terminal history
- Violates principle of least privilege for error information

### Recommended Fix
1. Limit error message detail to prevent sensitive data exposure
2. Redact or sanitize error responses before including in error messages
3. Consider using a limited reader to prevent reading large responses

```go
body, _ := io.ReadAll(io.LimitReader(resp.Body, 500))
return fmt.Errorf("unexpected status %d: %s", resp.StatusCode, sanitizeError(string(body)))
```

### Test Plan
1. Trigger various HTTP error responses
2. Verify error messages don't contain sensitive information
3. Test with malformed API keys to ensure they're not echoed back

---

## Issue 2: API Key Visible During Login (CRITICAL SECURITY)

**Priority**: Urgent
**Type**: Security Bug
**File**: `cmd/auth.go:34-35`

### Description
The API key is echoed to the terminal as users type it during the login flow. This means the key is visible on screen and could appear in terminal history, screenshots, or screen recordings.

### Current Code
```go
reader := bufio.NewReader(os.Stdin)
apiKey, err := reader.ReadString('\n')
```

### Risk
- API keys visible to anyone watching the screen
- Keys may be captured in terminal recordings or screenshots
- Terminal history could expose the key
- Violates standard password/secret input practices

### Recommended Fix
Use `golang.org/x/term` package to disable echo during password input:

```go
import "golang.org/x/term"

fmt.Print("> ")
apiKeyBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
if err != nil {
    fmt.Fprintf(os.Stderr, "Error reading input: %v\n", err)
    os.Exit(1)
}
apiKey := strings.TrimSpace(string(apiKeyBytes))
fmt.Println() // Print newline after hidden input
```

### Dependencies
Add to `go.mod` (already imported as indirect dependency):
- `golang.org/x/term`

### Test Plan
1. Run `linear auth login`
2. Verify API key is not displayed while typing
3. Verify newline is printed after input
4. Confirm key is still correctly saved to keyring

---

## Issue 3: No HTTP Timeouts (HIGH PRIORITY BUG)

**Priority**: High
**Type**: Bug
**File**: `internal/client/graphql.go:31`

### Description
The HTTP client has no timeout configuration, which means CLI operations can hang indefinitely on slow or unresponsive network connections.

### Current Code
```go
httpClient: &http.Client{},
```

### Impact
- CLI can freeze indefinitely waiting for responses
- Poor user experience on unreliable networks
- No way to detect or recover from hung connections
- Resource leaks if multiple requests hang

### Recommended Fix
```go
httpClient: &http.Client{
    Timeout: 30 * time.Second,
},
```

### Configuration Options
Consider making timeout configurable via:
- Environment variable: `LINEAR_TIMEOUT`
- Config file option
- Command-line flag: `--timeout`

### Test Plan
1. Test normal operations complete successfully
2. Test with simulated slow network (use `tc` or similar)
3. Test with unreachable server
4. Verify timeout occurs and returns appropriate error message

---

## Issue 4: Hardcoded Pagination Limit (HIGH PRIORITY BUG)

**Priority**: High
**Type**: Bug / Missing Feature
**File**: `internal/client/issues.go:70`

### Description
Issue listing only returns the first 50 issues with no pagination support or ability to retrieve more results.

### Current Code
```graphql
issues(filter: $filter, first: 50) {
```

### Impact
- Users cannot access issues beyond the first 50
- No visibility into total issue count
- Critical limitation for teams with many issues
- Inconsistent with Linear web UI capabilities

### Recommended Fix
Implement cursor-based pagination:

1. Add pagination parameters to `ListIssues`:
```go
func (c *Client) ListIssues(ctx context.Context, teamKey string, limit int, cursor *string) (*IssuesResponse, error)
```

2. Update GraphQL query to support pagination:
```graphql
query($filter: IssueFilter, $first: Int, $after: String) {
    issues(filter: $filter, first: $first, after: $after) {
        nodes { ... }
        pageInfo {
            hasNextPage
            endCursor
        }
    }
}
```

3. Update response struct:
```go
type IssuesResponse struct {
    Issues struct {
        Nodes []Issue `json:"nodes"`
        PageInfo struct {
            HasNextPage bool   `json:"hasNextPage"`
            EndCursor   string `json:"endCursor"`
        } `json:"pageInfo"`
    } `json:"issues"`
}
```

4. Add CLI flags:
- `--limit N` - number of results per page
- `--all` - fetch all results

### Test Plan
1. Test with workspace having >50 issues
2. Verify pagination works across multiple pages
3. Test `--all` flag retrieves all issues
4. Verify cursor-based pagination is efficient

---

## Issue 5: Missing Context Timeouts (HIGH PRIORITY BUG)

**Priority**: High
**Type**: Bug
**Files**: `cmd/issue.go`, `cmd/team.go`

### Description
All operations use `context.Background()` with no timeout, which means they can hang indefinitely regardless of HTTP client timeout.

### Current Code
```go
ctx := context.Background()
resp, err := c.ListIssues(ctx, teamFilter)
```

### Impact
- Operations can hang indefinitely
- No graceful timeout handling
- Inconsistent with modern Go best practices
- Difficult to implement request cancellation

### Recommended Fix
Add timeout context to all operations:

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
resp, err := c.ListIssues(ctx, teamFilter)
```

### Configuration
Make timeout configurable:
- Default: 30 seconds
- Environment variable: `LINEAR_TIMEOUT`
- Command flag: `--timeout duration`

### Files to Update
- `cmd/issue.go` - All command handlers
- `cmd/team.go` - All command handlers

### Test Plan
1. Test normal operations complete within timeout
2. Test operations properly timeout when server is slow
3. Test timeout error messages are user-friendly
4. Verify context cancellation works properly

---

## Issue 6: No Input Validation (MEDIUM PRIORITY)

**Priority**: Medium
**Type**: Security / Bug
**Files**: `cmd/issue.go`, `cmd/auth.go`

### Description
User inputs (titles, descriptions, team keys) are passed directly to GraphQL without validation. While GraphQL typically handles this, there's no defense in depth.

### Current Issues
- No length limits on titles/descriptions
- No sanitization of special characters
- No validation of team key format
- No validation of issue ID format

### Impact
- Potential for extremely large inputs
- Poor error messages when invalid data is provided
- No early feedback to users about invalid inputs
- Possible GraphQL query issues with malformed input

### Recommended Fix

1. **Title validation**:
```go
const maxTitleLength = 255

if len(issueTitle) == 0 {
    return errors.New("title cannot be empty")
}
if len(issueTitle) > maxTitleLength {
    return fmt.Errorf("title too long (max %d characters)", maxTitleLength)
}
```

2. **Description validation**:
```go
const maxDescriptionLength = 10000

if len(issueDesc) > maxDescriptionLength {
    return fmt.Errorf("description too long (max %d characters)", maxDescriptionLength)
}
```

3. **Team key validation**:
```go
// Team keys are typically uppercase letters/numbers
if !regexp.MustCompile(`^[A-Z0-9]+$`).MatchString(teamKey) {
    return errors.New("invalid team key format (expected uppercase letters/numbers)")
}
```

4. **API key validation** (make warning an error):
```go
if !strings.HasPrefix(apiKey, "lin_api_") {
    return errors.New("invalid API key format (must start with 'lin_api_')")
}
if len(apiKey) < 20 {
    return errors.New("API key appears to be too short")
}
```

### Test Plan
1. Test with empty inputs
2. Test with extremely long inputs
3. Test with special characters
4. Test with invalid team keys
5. Verify error messages are helpful

---

## Issue 7: Incomplete Error Handling (MEDIUM PRIORITY)

**Priority**: Medium
**Type**: Bug
**File**: `internal/client/graphql.go:88-90`

### Description
Only the first GraphQL error is returned; additional errors are silently ignored. This can make debugging difficult when multiple validation errors occur.

### Current Code
```go
if len(gqlResp.Errors) > 0 {
    return fmt.Errorf("%s", gqlResp.Errors[0].Message)
}
```

### Impact
- Missing error context
- Difficult to debug multiple validation failures
- Users don't get complete feedback
- Inconsistent with GraphQL best practices

### Recommended Fix

**Option 1: Return all errors**
```go
if len(gqlResp.Errors) > 0 {
    var messages []string
    for _, err := range gqlResp.Errors {
        messages = append(messages, err.Message)
    }
    return fmt.Errorf("GraphQL errors: %s", strings.Join(messages, "; "))
}
```

**Option 2: Custom error type**
```go
type GraphQLErrors struct {
    Errors []struct {
        Message string `json:"message"`
        Path    []any  `json:"path,omitempty"`
    }
}

func (e GraphQLErrors) Error() string {
    var messages []string
    for _, err := range e.Errors {
        if len(err.Path) > 0 {
            messages = append(messages, fmt.Sprintf("%v: %s", err.Path, err.Message))
        } else {
            messages = append(messages, err.Message)
        }
    }
    return strings.Join(messages, "\n")
}
```

### Test Plan
1. Trigger multiple validation errors
2. Verify all errors are reported
3. Ensure error messages are readable
4. Test error formatting with paths

---

## Issue 8: No Retry Logic (MEDIUM PRIORITY)

**Priority**: Medium
**Type**: Missing Feature
**File**: `internal/client/graphql.go`

### Description
Network requests have no retry mechanism for transient failures such as network blips, rate limiting, or temporary server issues.

### Impact
- Poor reliability on unstable networks
- Manual retries required for transient failures
- Bad user experience
- No handling of rate limits

### Recommended Fix

Implement exponential backoff retry logic:

```go
func (c *Client) Do(ctx context.Context, query string, variables map[string]interface{}, result interface{}) error {
    maxRetries := 3
    baseDelay := 1 * time.Second

    for attempt := 0; attempt <= maxRetries; attempt++ {
        err := c.doRequest(ctx, query, variables, result)

        if err == nil {
            return nil
        }

        // Don't retry on context cancellation or non-retryable errors
        if ctx.Err() != nil || !isRetryable(err) {
            return err
        }

        if attempt < maxRetries {
            delay := baseDelay * time.Duration(1<<uint(attempt)) // Exponential backoff
            time.Sleep(delay)
        }
    }

    return fmt.Errorf("request failed after %d attempts", maxRetries+1)
}

func isRetryable(err error) bool {
    // Retry on network errors and 5xx status codes
    // Don't retry on 4xx (client errors)
    // Handle rate limiting (429)
    return true // Implement based on error type
}
```

### Configuration Options
Make retries configurable:
- `LINEAR_MAX_RETRIES` - number of retry attempts
- `LINEAR_RETRY_DELAY` - base delay between retries
- `--no-retry` flag to disable

### Considerations
1. Only retry idempotent operations (GET/query, not mutations by default)
2. Respect Retry-After headers
3. Add jitter to prevent thundering herd
4. Log retry attempts at debug level

### Test Plan
1. Test with simulated network failures
2. Verify exponential backoff timing
3. Test rate limit handling
4. Ensure mutations are not retried unsafely
5. Verify max retry limit is respected

---

## Summary

| # | Issue | Priority | Type | Estimated Effort |
|---|-------|----------|------|------------------|
| 1 | API Key Exposure in Error Messages | Urgent | Security | 2-4 hours |
| 2 | API Key Visible During Login | Urgent | Security | 1-2 hours |
| 3 | No HTTP Timeouts | High | Bug | 1 hour |
| 4 | Hardcoded Pagination Limit | High | Feature | 4-6 hours |
| 5 | Missing Context Timeouts | High | Bug | 2-3 hours |
| 6 | No Input Validation | Medium | Security/Bug | 3-4 hours |
| 7 | Incomplete Error Handling | Medium | Bug | 2-3 hours |
| 8 | No Retry Logic | Medium | Feature | 4-6 hours |

**Total Estimated Effort**: 19-31 hours

### Recommended Implementation Order
1. Issues #2, #3, #5 (Quick security and timeout fixes)
2. Issue #1 (Error message sanitization)
3. Issue #7 (Better error handling)
4. Issue #6 (Input validation)
5. Issue #4 (Pagination)
6. Issue #8 (Retry logic)
