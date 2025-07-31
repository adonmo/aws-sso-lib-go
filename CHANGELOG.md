# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2024-12-19

### Added
- **AWS CLI Compatible Token Caching**: Full compatibility with AWS CLI SSO token format and cache locations
  - SHA1-based cache file naming matching AWS CLI behavior
  - Compatible JSON token format for seamless interoperability
  - Proper file permissions (0600 for tokens, 0700 for directories)
- **Structured Logging Support**: Optional structured logging using Go's `log/slog`
  - Configurable log levels and output formats
  - Security-aware (no sensitive data logged)
  - CLI `--verbose` flag for debug output
- **Enhanced Input Validation**: Centralized validation for AWS resources
  - Start URL, region, account ID, and role name validation
- **Improved Timeout Handling**: Better context-based timeouts for operations

### Fixed
- Security improvements: removed debug output that could leak sensitive information
- Improved error message sanitization
- Better handling of authentication polling edge cases

### Changed
- **Breaking**: None - fully backward compatible
- Added optional `Config` parameter to input structs for logging configuration
- Enhanced authentication flow reliability

## [0.2.0] - 2024-07-30

### Fixed
- **Critical**: Fixed SSO authentication polling to properly handle `AuthorizationPendingException`
- Authentication flow now correctly waits for user to complete browser authorization instead of failing prematurely
- Improved error handling with proper AWS SDK v2 typed errors instead of string matching
- Added support for `SlowDownException` to respect server rate limiting
- Added user-friendly polling status messages during authentication

### Technical Improvements
- Use `errors.As()` for robust error type checking
- Handle both `AuthorizationPendingException` and `SlowDownException` properly
- Maintain fallback string matching for compatibility
- Enhanced authentication flow reliability

This release fixes the main authentication issue where users would click "Allow" in the browser but the CLI would exit with an error instead of completing the login.

## [0.1.0] - 2024-07-30

### Added
- Initial release of aws-sso-lib-go
- Core library (`awsssolib`) for programmatic AWS SSO access
- Full-featured CLI tool (`aws-sso-util`) with comprehensive command set
- Interactive browser-based SSO authentication
- Token and credential caching (file and memory backends)
- AWS CLI profile management and bulk population
- Cross-platform support (Linux, macOS, Windows)

#### Library Features
- `GetAWSConfig`: Get AWS SDK v2 config for specific account/role
- `Login`/`Logout`: Interactive SSO authentication with browser support
- `ListAvailableAccounts`/`ListAvailableRoles`: Discover available resources
- Configuration management for AWS profiles
- Comprehensive caching system for tokens and credentials

#### CLI Commands
- `login`/`logout`: Manage SSO sessions
- `roles`: List available accounts and roles
- `configure profile`: Configure individual AWS CLI profiles
- `configure populate`: Bulk create profiles for all available access
- `run-as`: Execute commands with specific credentials
- `check`: Diagnose SSO configuration and access issues
- `credential-process`: AWS CLI credential process integration

#### Development
- Comprehensive unit tests
- CI/CD pipeline with GitHub Actions
- Cross-platform binary builds
- Example usage code
- Complete documentation

### Technical Details
- Built with Go 1.21+
- Uses AWS SDK for Go v2
- Cobra CLI framework
- Compatible with existing AWS SSO workflows

[Unreleased]: https://github.com/adonmo/aws-sso-lib-go/compare/v0.3.0...HEAD
[0.3.0]: https://github.com/adonmo/aws-sso-lib-go/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/adonmo/aws-sso-lib-go/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/adonmo/aws-sso-lib-go/releases/tag/v0.1.0