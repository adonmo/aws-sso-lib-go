# Structured Logging

This library includes comprehensive structured logging support using Go's standard `log/slog` package (available since Go 1.21).

## Features

- **Configurable log levels**: DEBUG, INFO, WARN, ERROR
- **Multiple output formats**: Text, JSON, or custom handlers
- **Detailed operation tracking**: Authentication flows, credential retrieval, caching operations
- **Security-aware logging**: Sensitive information is never logged
- **Zero dependencies**: Uses Go's standard library `log/slog`

## Usage

### Basic Usage with Default Configuration

```go
package main

import (
    "context"
    "github.com/adonmo/aws-sso-lib-go/awsssolib"
)

func main() {
    // Use default configuration (INFO level, text output to stderr)
    config := awsssolib.DefaultConfig()
    
    _, err := awsssolib.Login(context.Background(), awsssolib.LoginInput{
        StartURL:  "https://my-sso.awsapps.com/start",
        SSORegion: "us-east-1",
        Config:    config, // Pass the config to enable logging
    })
    if err != nil {
        panic(err)
    }
}
```

### Custom Logger Configuration

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
    
    // Use the configuration
    accounts, err := awsssolib.ListAvailableAccounts(context.Background(), awsssolib.ListAccountsInput{
        StartURL:  "https://my-sso.awsapps.com/start",
        SSORegion: "us-east-1",
        Config:    config,
    })
    if err != nil {
        panic(err)
    }
    
    // The library will output detailed JSON logs
    for _, account := range accounts {
        // Process accounts...
    }
}
```

### Silent Operation

```go
package main

import (
    "context"
    "log/slog"
    "io"
    "github.com/adonmo/aws-sso-lib-go/awsssolib"
)

func main() {
    // Create a logger that discards all output
    silentLogger := slog.New(slog.NewTextHandler(io.Discard, nil))
    config := awsssolib.NewConfig(silentLogger, slog.LevelError)
    
    // Or simply pass nil for Config to use minimal logging
    _, err := awsssolib.Login(context.Background(), awsssolib.LoginInput{
        StartURL:  "https://my-sso.awsapps.com/start",
        SSORegion: "us-east-1",
        Config:    nil, // Uses default INFO level logging
    })
    if err != nil {
        panic(err)
    }
}
```

## CLI Usage

The CLI commands support a `--verbose` flag to enable debug logging:

```bash
# Normal operation with INFO level logging
aws-sso-util login

# Enable debug logging to see all internal operations
aws-sso-util login --verbose

# The verbose flag shows detailed information about:
# - Token cache lookups
# - Authentication flows
# - API calls to AWS SSO
# - Credential caching operations
```

## Log Levels and Content

### DEBUG Level
- Token cache operations
- API call details
- Configuration validation
- Internal state transitions

### INFO Level
- Authentication success/failure
- Token and credential expiration times
- Major operation completions

### WARN Level
- Non-critical errors (e.g., cache failures)
- Fallback operations

### ERROR Level
- Authentication failures
- API errors
- Configuration validation failures

## Example Log Output

### Text Format (Default)
```
time=2024-01-15T10:30:45.123Z level=INFO msg="Starting SSO login" start_url=https://my-sso.awsapps.com/start sso_region=us-east-1 force_refresh=false disable_browser=false
time=2024-01-15T10:30:45.124Z level=DEBUG msg="Checking for cached SSO token"
time=2024-01-15T10:30:45.125Z level=INFO msg="Using cached SSO token" expires_at=2024-01-15T18:30:45Z expires_in=8h0m0s
```

### JSON Format
```json
{"time":"2024-01-15T10:30:45.123Z","level":"INFO","msg":"Starting SSO login","start_url":"https://my-sso.awsapps.com/start","sso_region":"us-east-1","force_refresh":false,"disable_browser":false}
{"time":"2024-01-15T10:30:45.124Z","level":"DEBUG","msg":"Checking for cached SSO token"}
{"time":"2024-01-15T10:30:45.125Z","level":"INFO","msg":"Using cached SSO token","expires_at":"2024-01-15T18:30:45Z","expires_in":"8h0m0s"}
```

## Security Considerations

The logging implementation is designed with security in mind:

- **No sensitive data**: Access tokens, credentials, and other sensitive information are never logged
- **Sanitized errors**: Error messages are sanitized to prevent information disclosure
- **Configurable levels**: Production deployments can use higher log levels to reduce verbosity
- **Standard library**: Uses Go's standard `log/slog` package for reliability and security

## Performance

- **Minimal overhead**: Logging checks are performed efficiently
- **Lazy evaluation**: Log messages are only formatted when the log level is enabled
- **No allocations**: When logging is disabled, there are no memory allocations for log formatting

## Integration with Existing Systems

Since this library uses Go's standard `log/slog`, it integrates seamlessly with:

- **Structured logging systems**: Elasticsearch, Splunk, etc.
- **Cloud logging**: AWS CloudWatch, Google Cloud Logging, Azure Monitor
- **Observability platforms**: Datadog, New Relic, Honeycomb
- **Custom handlers**: You can implement custom `slog.Handler` for specialized needs

## Migration from Previous Versions

If you're upgrading from a version without structured logging:

1. **No breaking changes**: All existing code continues to work without modification
2. **Opt-in logging**: Pass a `Config` to enable structured logging
3. **Gradual adoption**: You can enable logging for specific operations as needed

```go
// Before (still works)
awsssolib.Login(ctx, awsssolib.LoginInput{
    StartURL:  startURL,
    SSORegion: ssoRegion,
})

// After (with logging)
awsssolib.Login(ctx, awsssolib.LoginInput{
    StartURL:  startURL,
    SSORegion: ssoRegion,
    Config:    awsssolib.DefaultConfig(), // Add this line
})
```