# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Protohost is a CLI tool for managing multi-branch Docker Compose deployments with automatic port allocation and nginx configuration. It enables deploying multiple git branches simultaneously to the same server without port conflicts.

**Key concept**: Each branch gets an isolated deployment with its own allocated port, Docker network, and subdomain (e.g., `myapp-main.protohost.xyz`, `myapp-feature-x.protohost.xyz`).

## Build Commands

```bash
# Build the binary
make build           # Creates ./protohost

# Install to $GOPATH/bin (~/go/bin by default)
make install

# Clean build artifacts
make clean

# Run tests
make test

# Tidy dependencies
make tidy

# Lint (requires golangci-lint)
golangci-lint run
```

## Development Workflow

1. Make changes to Go source files
2. Build with `make build` or `make install`
3. Test locally: `./protohost [command]` or `protohost [command]` (if installed)
4. For remote testing, you can deploy to a test server using `protohost deploy --remote`

## Architecture

### Core Components

**Command Layer** (`internal/cmd/`)
- Each command (deploy, list, logs, etc.) has its own file
- Commands use Cobra framework for CLI parsing
- Entry point is `cmd/protohost/main.go`

**Configuration** (`internal/config/`)
- Parses `.protohost.config` files (bash-style KEY=VALUE format)
- Supports `.protohost.config.local` for local overrides
- Config loading happens at start of most commands

**Registry** (`internal/registry/`)
- SQLite database at `~/.protohost/registry.db`
- Tracks port allocations, deployment status, and TTL
- Each machine (local/remote) has its own independent registry
- Port allocation algorithm: finds first available port in range (BASE_WEB_PORT to BASE_WEB_PORT+99)
- Also validates ports are actually free by attempting to bind

**Deployment** (`internal/deploy/`)
- `local.go`: Local deployments using Docker Compose
- `remote.go`: SSH-based remote deployments
- Remote deploys: SSH to server → clone/pull repo → run `protohost deploy` on remote
- The remote server runs its own protohost instance with its own registry

**SSH Client** (`internal/ssh/`)
- Custom SSH client using `golang.org/x/crypto/ssh`
- Handles passphrase-protected SSH keys (prompts user securely)
- Methods: `Execute()` (capture output), `ExecuteInteractive()` (stream to terminal), `SCP()`

**Nginx Management** (`internal/nginx/`)
- Generates nginx config files from templates
- SCPs config to nginx server and restarts nginx
- Uses subdomain pattern: `{PROJECT_NAME}.protohost.xyz`

**Hooks** (`internal/hooks/`)
- Executes deployment hooks at various stages
- File-based (`.protohost/hooks/*.sh`) or config-based fallback
- Hook types: pre-deploy (local), post-deploy (local), post-start (remote), first-install (remote)

**Git Utils** (`internal/git/`)
- Detects current branch, checks if in git repo
- Simple wrapper around git commands

**Docker Compose** (`internal/docker/`)
- Wrapper for docker compose commands
- Passes WEB_PORT environment variable to compose

### Port Allocation Flow

1. User runs `protohost deploy`
2. Registry checks if project already has a port allocated
3. If yes: reuse port, update TTL
4. If no: find first available port (checks both registry and actual port binding)
5. Start containers with `WEB_PORT=<allocated_port> docker compose up`
6. User's `docker-compose.yml` must use `${WEB_PORT}` for port mapping

### Remote Deployment Flow

1. Load `.protohost.config` locally
2. Execute pre-deploy hook locally (if exists)
3. SSH to remote server
4. Check if protohost is installed on remote
5. On remote: clone/pull repo into `{REMOTE_BASE_DIR}/{PROJECT_NAME}`
6. On remote: `cd` into project dir and run `protohost deploy` (uses remote's own registry)
7. Post-start hook runs on remote after containers start
8. Generate nginx config and deploy to nginx server
9. Post-deploy hook runs locally after completion

### Registry Schema

```sql
CREATE TABLE port_allocations (
    id INTEGER PRIMARY KEY,
    project_name TEXT UNIQUE,      -- e.g., "myapp-feature-x"
    web_port INTEGER UNIQUE,       -- allocated port number
    branch TEXT,                   -- git branch name
    created_at TEXT,               -- RFC3339 timestamp
    expires_at TEXT,               -- RFC3339 timestamp (created + TTL_DAYS)
    status TEXT,                   -- 'running' or 'expired'
    repo_url TEXT
);
```

### Configuration Files

**`.protohost.config`** - Main configuration (committed to git)
```bash
PROJECT_PREFIX="myapp"
REPO_URL="git@github.com:user/repo.git"
TTL_DAYS=7
REMOTE_HOST="protohost.xyz"
REMOTE_USER="james"
REMOTE_BASE_DIR="~/protohost"
NGINX_PROXY_HOST="10.10.20.4"  # IP where Docker containers run
NGINX_SERVER="10.10.20.10"      # IP where nginx runs
BASE_WEB_PORT=3000
```

**`.protohost.config.local`** - Local overrides (gitignored)
```bash
REMOTE_HOST="staging.example.com"  # Override for testing
```

### Important Implementation Details

1. **SSH Key Passphrase Support**: The SSH client detects encrypted keys and prompts for passphrases using `golang.org/x/term` for secure input.

2. **TTL and Cleanup**: Deployments have a TTL (default 7 days). The `cleanup` command marks expired deployments and can remove them (stops containers, deletes files, removes nginx config).

3. **First Install Detection**: Registry returns `isNew` boolean from `AllocatePort()` to detect first deployment of a branch (used to trigger `first-install` hook).

4. **Nginx Configuration**: Nginx can run on a different server than Docker. Config uses `NGINX_PROXY_HOST` for proxy_pass target and gets deployed to `NGINX_SERVER`.

5. **Docker Project Isolation**: Uses `-p {PROJECT_NAME}` flag with docker compose to isolate containers, networks, and volumes per branch.

6. **Port Range**: Supports 100 deployments by default (BASE_WEB_PORT to BASE_WEB_PORT+99). Only web ports are allocated; other services (MySQL, Redis) run on Docker internal networks.

## Testing

No automated test suite currently. Test manually:

1. Create a test project with `docker-compose.yml`
2. Run `protohost init` in the project
3. Test local: `protohost deploy`
4. Test remote: `protohost deploy --remote` (requires remote server)
5. Verify: `protohost list`, `protohost logs`, `protohost info`
6. Test cleanup: `protohost cleanup --dry-run`

## Common Tasks

**Adding a new command:**
1. Create `internal/cmd/newcommand.go` with `NewCommandCmd()` function
2. Add command to root in `cmd/protohost/main.go`: `rootCmd.AddCommand(cmd.NewCommandCmd())`
3. Follow existing patterns (use config.Load(), handle errors consistently)

**Modifying SSH behavior:**
- Edit `internal/ssh/client.go`
- Key loading logic is in `NewClient()`
- Passphrase handling is in the key parsing section

**Changing port allocation:**
- Edit `internal/registry/registry.go`
- Main logic in `AllocatePort()` and `findAvailablePort()`

**Updating nginx config generation:**
- Edit `internal/nginx/nginx.go`
- Template is in `generateNginxConfig()`

## Version Bumping

1. Update version in `cmd/protohost/main.go` (line 11: `var version = "x.y.z"`)
2. Commit changes
3. Tag: `git tag vx.y.z`
4. Push: `git push && git push --tags`

## SSH Keys

The tool automatically tries these SSH keys in order:
1. `~/.ssh/id_rsa`
2. `~/.ssh/id_ed25519`

If the key is passphrase-protected, it prompts securely for the passphrase.

## Dependencies

Key Go modules:
- `github.com/spf13/cobra` - CLI framework
- `golang.org/x/crypto/ssh` - SSH client
- `github.com/mattn/go-sqlite3` - SQLite driver (requires CGO)
- `golang.org/x/term` - Secure password input

Note: SQLite requires CGO, so cross-compilation needs appropriate C toolchain.
