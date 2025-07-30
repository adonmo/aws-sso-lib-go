# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/adonmo/aws-sso-lib-go/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/adonmo/aws-sso-lib-go/releases/tag/v0.1.0