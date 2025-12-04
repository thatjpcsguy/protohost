# Changelog

All notable changes to protohost-deploy will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial release of protohost-deploy
- Automatic port allocation for multi-branch deployments
- Branch-based Docker Compose project isolation
- Nginx reverse proxy configuration generator
- SSH-based remote deployment
- Automatic cleanup of expired deployments (7-day TTL)
- Pre-deploy, post-deploy, and post-start hooks
- Local development workflow with `make` targets
- Deployment listing and management commands
- Nginx configuration enable/disable commands
- Support for custom remote hosts
- Database reset and nuke deployment options
- Configuration via `.protohost.config` file
- Installation script for easy project setup
- Comprehensive documentation (README, SETUP, ARCHITECTURE)

### Changed
- N/A (initial release)

### Deprecated
- N/A (initial release)

### Removed
- N/A (initial release)

### Fixed
- N/A (initial release)

### Security
- SSH key-based authentication only
- No credentials stored in config files
- SSL/TLS enforced for all web traffic

## [0.1.0] - 2025-12-05

### Added
- Initial development version
- Core deployment scripts extracted from vibe-insights project
- Generalized for use across multiple projects
- Port allocation algorithm
- Nginx configuration management
- Deployment lifecycle management

[Unreleased]: https://github.com/YOUR_ORG/protohost-deploy/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/YOUR_ORG/protohost-deploy/releases/tag/v0.1.0
