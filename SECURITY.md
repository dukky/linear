# Security Policy

## Security Features

### OAuth 2.0 with PKCE (RFC 7636)

This CLI implements OAuth 2.0 with Proof Key for Code Exchange (PKCE), which provides enhanced security for public clients:

- **Code Verifier**: A cryptographically random string (64 bytes)
- **Code Challenge**: SHA-256 hash of the code verifier
- **CSRF Protection**: Random state parameter validation
- **Authorization Code Interception Prevention**: PKCE prevents attackers from using intercepted authorization codes

### Token Storage Security

Access tokens are protected using multiple layers of security:

1. **AES-256-GCM Encryption**
   - Industry-standard authenticated encryption
   - Provides both confidentiality and integrity
   - 256-bit key size for maximum security

2. **PBKDF2 Key Derivation**
   - 100,000 iterations (OWASP recommended minimum)
   - SHA-256 as the pseudo-random function
   - Machine-specific salt (hostname)
   - Path-based password (user home directory)

3. **File System Permissions**
   - Token file: 0600 (read/write owner only)
   - Config directory: 0700 (full access owner only)
   - Prevents unauthorized access by other users

### Network Security

1. **Local Redirect Server**
   - Binds only to 127.0.0.1 (localhost)
   - Cannot be accessed from network
   - Automatic timeout after 5 minutes
   - Validates state parameter for CSRF protection

2. **HTTPS API Communication**
   - All Linear API requests use HTTPS
   - Token sent via Authorization header (not URL)
   - TLS certificate validation enforced

## Reporting Security Vulnerabilities

If you discover a security vulnerability, please:

1. **Do NOT** open a public GitHub issue
2. Email security concerns to: [your-security-email]
3. Include:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fix (if any)

We will acknowledge receipt within 48 hours and provide a detailed response within 7 days.

## Security Best Practices

### For Users

1. **Protect Your Credentials**
   - Never commit `LINEAR_CLIENT_ID` or `LINEAR_CLIENT_SECRET` to version control
   - Use environment variables or secure secret management
   - Don't share your OAuth application credentials

2. **Token Management**
   - Use `linear auth logout` when done to clear tokens
   - Regularly rotate OAuth credentials
   - Monitor Linear's security logs for unusual activity

3. **Environment Security**
   - Keep your system updated
   - Use full-disk encryption
   - Enable firewall
   - Use antivirus software

4. **OAuth Application Security**
   - Limit OAuth scopes to minimum required (read/write)
   - Regularly review authorized applications in Linear
   - Use separate OAuth apps for different environments
   - Enable 2FA on your Linear account

### For Developers

1. **Code Security**
   - Never log sensitive data (tokens, secrets)
   - Use secure random number generation (`crypto/rand`)
   - Validate all input
   - Follow Go security best practices

2. **Dependency Management**
   - Regularly update dependencies: `go get -u && go mod tidy`
   - Review security advisories: `go list -m -u all`
   - Use `govulncheck`: `go install golang.org/x/vuln/cmd/govulncheck@latest`

3. **Testing**
   - Test OAuth flow with various error conditions
   - Verify CSRF protection
   - Test token encryption/decryption
   - Validate file permissions

## Security Checklist

Before deploying or using the CLI:

- [ ] OAuth credentials stored in environment variables (not hardcoded)
- [ ] Latest version of Go installed
- [ ] Dependencies up to date (`go mod tidy`)
- [ ] No security vulnerabilities in dependencies (`govulncheck`)
- [ ] Token file has 0600 permissions
- [ ] Config directory has 0700 permissions
- [ ] HTTPS used for all API communication
- [ ] 2FA enabled on Linear account
- [ ] OAuth application has minimum required scopes

## Known Limitations

1. **Machine Binding**: Tokens are encrypted with machine-specific keys. Moving the token file to another machine will not work (by design).

2. **User Binding**: Tokens are not bound to a specific user. Any user who can read the token file (same account) can use it.

3. **Token Revocation**: The CLI does not currently support automatic token refresh. Tokens must be manually re-authenticated when expired.

4. **Memory Security**: Tokens are held in memory during execution. On systems with memory dumps enabled, this could pose a risk.

## Cryptographic Details

### Encryption

- **Algorithm**: AES-256-GCM
- **Key Size**: 256 bits (32 bytes)
- **Nonce**: 12 bytes (randomly generated per encryption)
- **Authentication Tag**: Included in GCM mode

### Key Derivation

- **Function**: PBKDF2-HMAC-SHA256
- **Iterations**: 100,000
- **Salt**: Machine hostname + constant string
- **Password**: User home directory + constant string
- **Output**: 32 bytes (256 bits)

### Random Generation

- **Source**: `crypto/rand` (cryptographically secure)
- **PKCE Code Verifier**: 64 bytes, base64-url encoded
- **State Parameter**: 32 bytes, base64-url encoded

## Compliance

This implementation follows:

- **RFC 6749**: OAuth 2.0 Authorization Framework
- **RFC 7636**: PKCE for OAuth Public Clients
- **RFC 6750**: Bearer Token Usage
- **OWASP Top 10**: Web Application Security Risks
- **NIST SP 800-132**: Password-Based Key Derivation

## Updates

This security policy is reviewed quarterly and updated as needed.

Last Updated: 2025-11-15
