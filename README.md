# aws-sso-lib-go

A Go library and CLI tool for AWS IAM Identity Center (formerly AWS SSO) that makes it easier to work with multiple AWS accounts and roles.

This project is inspired by the Python [aws-sso-util](https://github.com/benkehoe/aws-sso-util) project and provides similar functionality for Go developers.

## Features

- **Library (`awsssolib`)**: Core functionality for programmatic interaction with AWS SSO
  - Get AWS credentials for specific accounts and roles
  - Interactive browser-based SSO login
  - List available accounts and roles
  - Credential caching and management
  - Support for multiple SSO instances
  - **🎉 NEW**: Comprehensive structured logging with `log/slog`

- **CLI Tool (`aws-sso-util`)**: Command-line utilities for AWS SSO operations
  - Configure AWS profiles in `~/.aws/config`
  - Login/logout from SSO
  - List available roles
  - Run commands with specific account/role credentials
  - Open AWS Console in browser
  - Admin utilities for SSO management
  - **🎉 NEW**: `--verbose` flag for detailed debug logging

## Installation

### CLI Tool

#### Go Install (Recommended)
```bash
go install github.com/adonmo/aws-sso-lib-go/cmd/aws-sso-util@latest
```

#### Download Pre-built Binaries
Download the latest release from [GitHub Releases](https://github.com/adonmo/aws-sso-lib-go/releases):

```bash
# Linux (x64)
curl -L https://github.com/adonmo/aws-sso-lib-go/releases/latest/download/aws-sso-util-linux-amd64.tar.gz | tar xz

# macOS (Intel)
curl -L https://github.com/adonmo/aws-sso-lib-go/releases/latest/download/aws-sso-util-darwin-amd64.tar.gz | tar xz

# macOS (Apple Silicon)
curl -L https://github.com/adonmo/aws-sso-lib-go/releases/latest/download/aws-sso-util-darwin-arm64.tar.gz | tar xz

# Windows
# Download aws-sso-util-windows-amd64.zip from releases page
```

#### Build from Source
```bash
git clone https://github.com/adonmo/aws-sso-lib-go.git
cd aws-sso-lib-go
make build
# Binary will be in ./dist/aws-sso-util
```

### Library

```bash
go get github.com/adonmo/aws-sso-lib-go
```

## Library Usage

### Get a session for a specific account and role

```go
package main

import (
    "context"
    "fmt"
    "log"
    
    "github.com/adonmo/aws-sso-lib-go/awsssolib"
    "github.com/aws/aws-sdk-go-v2/service/s3"
)

func main() {
    ctx := context.Background()
    
    // Optional: Configure structured logging
    config := awsssolib.DefaultConfig() // INFO-level logging
    
    // Get AWS SDK config for a specific account and role
    cfg, err := awsssolib.GetAWSConfig(ctx, awsssolib.GetAWSConfigInput{
        StartURL:   "https://my-sso.awsapps.com/start",
        SSORegion:  "us-east-1",
        AccountID:  "123456789012",
        RoleName:   "MyRole",
        Region:     "us-west-2",
        Login:      true, // Interactively log in if needed
        Config:     config, // Enable structured logging
    })
    if err != nil {
        log.Fatal(err)
    }
    
    // Use the config with any AWS SDK v2 client
    client := s3.NewFromConfig(cfg)
    
    // ... use the client
}
```

### Login to SSO

```go
// Basic login
token, err := awsssolib.Login(ctx, awsssolib.LoginInput{
    StartURL:  "https://my-sso.awsapps.com/start",
    SSORegion: "us-east-1",
})
if err != nil {
    log.Fatal(err)
}
fmt.Printf("Logged in successfully, token expires at: %s\n", token.ExpiresAt)

// Login with structured logging
config := awsssolib.DefaultConfig()
token, err := awsssolib.Login(ctx, awsssolib.LoginInput{
    StartURL:  "https://my-sso.awsapps.com/start",
    SSORegion: "us-east-1",
    Config:    config, // Enable logging
})
```

### List available accounts and roles

```go
// List all available accounts
accounts, err := awsssolib.ListAvailableAccounts(ctx, awsssolib.ListAccountsInput{
    StartURL:  "https://my-sso.awsapps.com/start",
    SSORegion: "us-east-1",
    Login:     true,
})
if err != nil {
    log.Fatal(err)
}

for _, account := range accounts {
    fmt.Printf("Account: %s (%s)\n", account.AccountName, account.AccountID)
}

// List all available roles
roles, err := awsssolib.ListAvailableRoles(ctx, awsssolib.ListRolesInput{
    StartURL:  "https://my-sso.awsapps.com/start",
    SSORegion: "us-east-1",
    Login:     true,
})
if err != nil {
    log.Fatal(err)
}

for _, role := range roles {
    fmt.Printf("Role: %s in account %s (%s)\n", 
        role.RoleName, role.AccountName, role.AccountID)
}
```

### Structured Logging

The library includes comprehensive structured logging support using Go's standard `log/slog` package:

```go
package main

import (
    "context"
    "log/slog"
    "os"
    "github.com/adonmo/aws-sso-lib-go/awsssolib"
)

func main() {
    // Create a JSON logger with DEBUG level
    jsonLogger := slog.New(slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
        Level: slog.LevelDebug,
    }))
    
    // Create configuration with custom logger
    config := awsssolib.NewConfig(jsonLogger, slog.LevelDebug)
    
    // All operations will now produce structured logs
    accounts, err := awsssolib.ListAvailableAccounts(context.Background(), awsssolib.ListAccountsInput{
        StartURL:  "https://my-sso.awsapps.com/start",
        SSORegion: "us-east-1",
        Config:    config, // Enable detailed logging
    })
    
    // Logs will include structured data like:
    // {"time":"2024-12-19T10:30:45Z","level":"INFO","msg":"Starting SSO login","start_url":"https://..."}
    // {"time":"2024-12-19T10:30:46Z","level":"DEBUG","msg":"Checking for cached SSO token"}
    // {"time":"2024-12-19T10:30:47Z","level":"INFO","msg":"Using cached SSO token","expires_at":"2024-12-19T18:30:45Z"}
}
```

**Key Features:**
- **Security-aware**: No sensitive data (tokens, credentials) ever logged
- **Production-ready**: Configurable log levels and output formats
- **Zero overhead**: No performance impact when logging is disabled
- **Integration-friendly**: Works with any `slog.Handler` implementation

For complete documentation, see [STRUCTURED_LOGGING.md](./STRUCTURED_LOGGING.md).

## CLI Usage

### Configure AWS profiles

```bash
# Set default SSO configuration
export AWS_DEFAULT_SSO_START_URL=https://my-sso.awsapps.com/start
export AWS_DEFAULT_SSO_REGION=us-east-1

# Configure a single profile interactively
aws-sso-util configure profile my-profile

# Populate all available roles as profiles
aws-sso-util configure populate --regions us-east-1,us-west-2
```

### Login and logout

```bash
# Login to SSO (will open browser)
aws-sso-util login

# Login with verbose debug logging
aws-sso-util login --verbose

# Login with specific start URL
aws-sso-util login --start-url https://my-sso.awsapps.com/start --sso-region us-east-1

# Logout
aws-sso-util logout
```

### List available roles

```bash
# List all available roles
aws-sso-util roles

# Filter by account
aws-sso-util roles --account 123456789012
```

### Run commands with specific credentials

```bash
# Run a command as a specific account/role
aws-sso-util run-as --account 123456789012 --role MyRole -- aws s3 ls

# Run with a specific region
aws-sso-util run-as --account 123456789012 --role MyRole --region us-west-2 -- aws ec2 describe-instances
```

### Open AWS Console

```bash
# Open console for a specific account/role
aws-sso-util console --account 123456789012 --role MyRole

# Open a specific service
aws-sso-util console --account 123456789012 --role MyRole --service ec2
```

## Configuration

The tool respects the following environment variables:

- `AWS_DEFAULT_SSO_START_URL`: Default SSO start URL
- `AWS_DEFAULT_SSO_REGION`: Default SSO region
- `AWS_SSO_CACHE_DIR`: Directory for SSO token cache (default: `~/.aws/sso/cache`)
- `AWS_CLI_CACHE_DIR`: Directory for CLI credential cache (default: `~/.aws/cli/cache`)

## Development

### Prerequisites

- Go 1.21 or later
- Make (optional, for using Makefile)

### Building

```bash
# Build the library
go build ./...

# Build the CLI
go build -o aws-sso-util ./cmd/aws-sso-util

# Run tests
go test ./...

# Run with race detector
go test -race ./...
```

### Project Structure

```
aws-sso-lib-go/
├── awsssolib/          # Core library package
│   ├── sso.go          # SSO authentication and token management
│   ├── config.go       # Configuration and profile management
│   ├── credentials.go  # Credential fetching and caching
│   ├── browser.go      # Browser interaction for login
│   └── cache.go        # Token and credential caching
├── cmd/
│   └── aws-sso-util/   # CLI application
│       ├── main.go
│       └── commands/   # CLI command implementations
├── examples/           # Example usage
└── docs/              # Additional documentation
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

## Acknowledgments

This project is inspired by and based on the design of [aws-sso-util](https://github.com/benkehoe/aws-sso-util) by Ben Kehoe.