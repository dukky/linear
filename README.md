# Linear CLI

A secure command-line interface for interacting with the [Linear](https://linear.app) API, built in Go with OAuth 2.0 authentication.

## Features

- **Secure OAuth 2.0 Authentication** with PKCE (Proof Key for Code Exchange)
- **Encrypted Token Storage** using AES-GCM encryption
- **Issue Management** - Create, list, and view issues
- **Team Management** - List and view teams
- **User-Friendly CLI** with intuitive commands and formatted output

## Security Features

### OAuth 2.0 with PKCE
This CLI implements OAuth 2.0 with PKCE (RFC 7636), which provides additional security for public clients like CLI applications by preventing authorization code interception attacks.

### Encrypted Token Storage
Access tokens are encrypted using AES-256-GCM before being stored locally. The encryption key is derived from machine-specific data using PBKDF2 with 100,000 iterations, making it resistant to brute-force attacks.

### Local Redirect Server
The OAuth callback uses a temporary local HTTP server (127.0.0.1:8793) that:
- Only accepts connections from localhost
- Implements CSRF protection via state parameter validation
- Automatically shuts down after receiving the callback
- Has a 5-minute timeout for security

## Installation

### From Source

```bash
git clone https://github.com/your-username/linear-cli.git
cd linear-cli
go build -o linear
sudo mv linear /usr/local/bin/
```

### Using Go Install

```bash
go install github.com/linear-cli/linear@latest
```

## Setup

### 1. Create a Linear OAuth Application

1. Go to [Linear Settings > API](https://linear.app/settings/api)
2. Click "Create new OAuth application"
3. Set the callback URL to: `http://127.0.0.1:8793/callback`
4. Note your Client ID and Client Secret

### 2. Set Environment Variables

```bash
export LINEAR_CLIENT_ID="your-client-id"
export LINEAR_CLIENT_SECRET="your-client-secret"
```

Or add them to your shell profile (`~/.bashrc`, `~/.zshrc`, etc.):

```bash
echo 'export LINEAR_CLIENT_ID="your-client-id"' >> ~/.bashrc
echo 'export LINEAR_CLIENT_SECRET="your-client-secret"' >> ~/.bashrc
source ~/.bashrc
```

### 3. Authenticate

```bash
linear auth
```

This will:
1. Open your browser to Linear's authorization page
2. Start a local server to receive the OAuth callback
3. Securely store your encrypted access token

## Usage

### Authentication

```bash
# Authenticate with Linear
linear auth

# Check who you're authenticated as
linear auth whoami

# Log out and remove stored credentials
linear auth logout
```

You can also provide credentials via flags:

```bash
linear auth --client-id="your-id" --client-secret="your-secret"
```

### Issues

```bash
# List issues (default: 10 most recent)
linear issue list

# List more issues
linear issue list -n 25

# Filter issues by team
linear issue list -t ENG

# View issue details
linear issue view ISSUE-123

# Create a new issue
linear issue create --title "Bug in login" --description "Users cannot log in" --team team_abc123

# Create issue with assignee
linear issue create --title "Feature request" --team team_abc123 --assignee user_xyz789
```

### Teams

```bash
# List all teams
linear team list
```

### Help

```bash
# General help
linear --help

# Command-specific help
linear issue --help
linear auth --help
```

## Configuration

The CLI stores its configuration in `~/.linear/`:

- `tokens.enc` - Encrypted access token (AES-256-GCM)

All files are created with restrictive permissions (0600/0700) for security.

## Development

### Building

```bash
go build -o linear
```

### Running Tests

```bash
go test ./...
```

### Project Structure

```
linear-cli/
├── cmd/              # CLI commands
│   ├── root.go      # Root command
│   ├── auth.go      # Authentication commands
│   ├── issue.go     # Issue management commands
│   └── team.go      # Team management commands
├── internal/
│   ├── api/         # Linear API client
│   │   ├── client.go
│   │   └── types.go
│   ├── auth/        # OAuth implementation
│   │   └── oauth.go
│   └── config/      # Configuration and token storage
│       └── config.go
└── main.go          # Application entry point
```

## Security Considerations

### Token Storage
- Tokens are encrypted using AES-256-GCM
- Encryption key derived from machine-specific data
- Token files have restrictive permissions (0600)
- Configuration directory has restrictive permissions (0700)

### OAuth Flow
- Implements PKCE for additional security
- Uses state parameter for CSRF protection
- Local redirect server only accepts localhost connections
- Server automatically times out after 5 minutes
- Browser callback validates state before accepting code

### Best Practices
- Never commit your Client ID or Secret to version control
- Use environment variables for credentials
- Regularly rotate your OAuth credentials
- Use `linear auth logout` when done to remove tokens

## Troubleshooting

### Authentication Issues

**Browser doesn't open automatically**
- The authentication URL will be printed in the terminal
- Copy and paste it into your browser manually

**"No token found" error**
- Run `linear auth` to authenticate first

**"State mismatch" error**
- This indicates a potential CSRF attack or timing issue
- Try authenticating again
- Ensure no other OAuth flow is running

### Port Conflicts

If port 8793 is already in use, you'll need to:
1. Change the port in `internal/config/config.go`
2. Update your Linear OAuth application callback URL
3. Rebuild the CLI

## API Reference

This CLI uses the [Linear GraphQL API](https://developers.linear.app/docs/graphql/working-with-the-graphql-api).

## Contributing

Contributions are welcome! Please:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests if applicable
5. Submit a pull request

## License

MIT License - See LICENSE file for details

## Support

For issues and questions:
- Open an issue on GitHub
- Check [Linear's API documentation](https://developers.linear.app)
- Review the [OAuth 2.0 PKCE specification](https://tools.ietf.org/html/rfc7636)
