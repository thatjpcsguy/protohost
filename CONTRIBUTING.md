# Contributing to Protohost Deploy

Thank you for your interest in contributing to protohost-deploy! This document provides guidelines and information for contributors.

## Code of Conduct

Be respectful, inclusive, and professional in all interactions. We're building a tool to help developers be more productive.

## How to Contribute

### Reporting Bugs

Before creating a bug report:
1. Check existing issues to avoid duplicates
2. Test with the latest version
3. Verify the issue is reproducible

When creating a bug report, include:
- **Clear title**: Summarize the issue
- **Steps to reproduce**: Detailed steps to trigger the bug
- **Expected behavior**: What should happen
- **Actual behavior**: What actually happens
- **Environment**: OS, Docker version, Python version, etc.
- **Logs**: Relevant error messages or logs
- **Configuration**: Sanitized `.protohost.config` (remove sensitive data)

Example:
```
Title: Port allocation fails with more than 50 deployments

Steps:
1. Create 50+ branch deployments
2. Run `make up` on a new branch
3. Observe timeout error

Expected: Should find available port in reasonable time
Actual: Times out after 30 seconds

Environment:
- macOS 13.5
- Docker 24.0.6
- Python 3.11.4

Logs:
```
[paste error logs here]
```

Configuration: (see attached)
```

### Suggesting Features

Feature requests are welcome! Please include:
- **Use case**: What problem does this solve?
- **Proposed solution**: How should it work?
- **Alternatives considered**: What other approaches did you think about?
- **Impact**: Who benefits from this feature?

Good feature request:
```
Title: Add support for PostgreSQL port allocation

Use case: Many projects use PostgreSQL instead of MySQL. Currently, only MySQL ports are allocated, forcing manual port configuration for PostgreSQL services.

Proposed solution: Add BASE_POSTGRES_PORT configuration option and allocate PostgreSQL ports alongside web/MySQL/Redis.

Alternatives: Users could manually configure ports, but this defeats the purpose of automatic allocation.

Impact: Benefits any project using PostgreSQL, which is a significant portion of modern web applications.
```

### Pull Requests

1. **Fork the repository**
   ```bash
   git clone git@github.com:YOUR_USERNAME/protohost.git
   cd protohost
   ```

2. **Create a feature branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Follow the coding standards (see below)
   - Add tests if applicable
   - Update documentation

4. **Test your changes**
   - Test with a real project
   - Verify nothing breaks existing functionality
   - Test on multiple platforms if possible

5. **Commit with descriptive messages**
   ```bash
   git commit -m "Add PostgreSQL port allocation support

   - Add BASE_POSTGRES_PORT config option
   - Update get_ports.py to allocate Postgres port
   - Add postgres port to Makefile.inc template
   - Update documentation with Postgres examples"
   ```

6. **Push to your fork**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Create a pull request**
   - Describe what changed and why
   - Link related issues
   - Include screenshots for UI changes
   - Mark as draft if work-in-progress

## Development Setup

### Prerequisites

- Git
- Docker & Docker Compose V2
- Python 3.6+
- Bash (or compatible shell)
- Make

### Testing Your Changes

1. **Create a test project**
   ```bash
   mkdir ~/test-protohost-project
   cd ~/test-protohost-project
   git init
   # Add docker-compose.yml
   ```

2. **Install your development version**
   ```bash
   /path/to/your/protohost-deploy/install.sh
   ```

3. **Test locally**
   ```bash
   make up
   make down
   ```

4. **Test deployment** (if you have a test server)
   ```bash
   make deploy HOST=test-server.example.com
   ```

### Testing Checklist

- [ ] Install script works on fresh project
- [ ] Install script works on project with existing Makefile
- [ ] `make up` starts services correctly
- [ ] Port allocation works with multiple branches
- [ ] `make deploy` successfully deploys
- [ ] Nginx configuration is generated correctly
- [ ] Cleanup removes expired deployments
- [ ] Documentation is updated
- [ ] No credentials or sensitive data in commits

## Coding Standards

### Shell Scripts

- Use `#!/bin/bash` shebang
- Use `set -e` to exit on errors
- Quote variables: `"${VAR}"` not `$VAR`
- Use descriptive variable names: `PROJECT_NAME` not `pn`
- Add comments for complex logic
- Use functions for reusable code

Example:
```bash
#!/bin/bash
set -e

# Load configuration
if [ ! -f ".protohost.config" ]; then
    echo "❌ Error: Configuration not found"
    exit 1
fi

source .protohost.config

# Validate required settings
validate_config() {
    local required_vars=("REMOTE_HOST" "REMOTE_USER" "PROJECT_PREFIX")

    for var in "${required_vars[@]}"; do
        if [ -z "${!var}" ]; then
            echo "❌ Error: ${var} not set in .protohost.config"
            exit 1
        fi
    done
}

validate_config
```

### Python Scripts

- Use Python 3.6+ compatible syntax
- Follow PEP 8 style guide
- Add docstrings to functions
- Use type hints where appropriate
- Handle errors gracefully

Example:
```python
#!/usr/bin/env python3
"""
Port allocation script for protohost-deploy.
"""

import sys
from typing import Dict, Optional

def get_running_ports(project_name: str) -> Optional[Dict[str, int]]:
    """
    Check if containers for a project are running and return their ports.

    Args:
        project_name: Docker Compose project name

    Returns:
        Dictionary with port mappings or None if not running
    """
    try:
        # Implementation
        pass
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        return None
```

### Makefile

- Use tabs for indentation (Make requirement)
- Add `.PHONY` targets for non-file targets
- Add comments above complex targets
- Use `@echo` for user-facing output

Example:
```makefile
.PHONY: deploy

# Deploy current branch to remote server
# Usage: make deploy [RESET_DB=true] [NUKE=true] [HOST=custom.host]
deploy:
	@echo "🚀 Deploying $(PROJECT_NAME)..."
	@./.protohost/lib/deploy.sh $(if $(RESET_DB),--reset-db) $(if $(NUKE),--nuke)
```

### Documentation

- Use Markdown for all documentation
- Include code examples
- Keep line length reasonable (~100 chars)
- Use proper heading hierarchy
- Add table of contents for long documents

## Project Structure

Understanding the structure helps you know where to make changes:

```
protohost-deploy/
├── README.md              # Main documentation (user-facing)
├── CONTRIBUTING.md        # This file (contributor guide)
├── LICENSE                # MIT license
├── CHANGELOG.md           # Version history
├── install.sh             # Installation script (entry point)
├── .gitignore             # Git ignore rules
├── bin/                   # Future: CLI commands
├── lib/                   # Core library scripts
│   ├── deploy.sh          # Remote deployment logic
│   ├── get_ports.py       # Port allocation algorithm
│   ├── list_deployments.sh  # Deployment listing
│   └── nginx_manage.sh    # Nginx configuration management
├── templates/             # Templates for generated files
│   └── Makefile.template  # Template for projects without Makefile
└── docs/                  # Additional documentation
    ├── SETUP.md           # Setup and configuration guide
    └── ARCHITECTURE.md    # How it works internally
```

**Where to make changes**:
- Core logic: `lib/*.sh` or `lib/*.py`
- Installation: `install.sh`
- Documentation: `README.md`, `docs/*.md`
- Templates: `templates/*`

## Testing

Currently, testing is manual. Future improvements:
- Automated integration tests
- Unit tests for Python code
- Shell script testing with bats
- CI/CD pipeline

For now, test thoroughly before submitting PRs:
1. Test on your own projects
2. Test edge cases (no Makefile, existing .protohost, etc.)
3. Test on different platforms (macOS, Linux if possible)

## Documentation

When adding features, update:
1. **README.md** - User-facing feature description
2. **SETUP.md** - If it changes setup/configuration
3. **ARCHITECTURE.md** - If it changes how things work internally
4. **CHANGELOG.md** - Add to "Unreleased" section
5. **Inline comments** - Explain complex logic in code

## Release Process

(For maintainers)

1. Update version in CHANGELOG.md
2. Create release notes summarizing changes
3. Tag release: `git tag -a v1.0.0 -m "Release 1.0.0"`
4. Push tags: `git push origin --tags`
5. Create GitHub release
6. Update install script URL in README if needed

## Getting Help

- **Questions**: Open a discussion on GitHub
- **Bugs**: Open an issue
- **Security**: Email maintainers directly (don't open public issue)
- **Feature ideas**: Open an issue with "enhancement" label

## Recognition

Contributors will be:
- Listed in release notes
- Added to AUTHORS file (future)
- Acknowledged in documentation

Thank you for contributing! 🙏
