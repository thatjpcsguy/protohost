# Protohost

A command-line tool for managing multi-branch Docker Compose deployments with automatic port allocation and nginx configuration.

## Features

- 🚀 **Branch-based deployments**: Deploy multiple branches simultaneously without port conflicts
- 🔄 **Automatic port allocation**: Smart port management tracked in SQLite registry
- 🌐 **Nginx integration**: Automatic reverse proxy configuration with SSL
- 🧹 **Auto-cleanup**: Automatically removes expired deployments (configurable TTL)
- 📊 **Deployment tracking**: List and manage all running deployments
- 🔧 **Easy setup**: Single binary, no dependencies
- 🌍 **Local & Remote**: Same commands work locally or remotely with `--remote` flag

## Quick Start

### Installation

**Option 1: Using Go**
```bash
go install github.com/thatjpcsguy/protohost/cmd/protohost@latest
```

**Option 2: Build from source**
```bash
git clone https://github.com/thatjpcsguy/protohost
cd protohost
make install
```

**Option 3: Download binary**
```bash
# Download from GitHub releases (coming soon)
curl -sSL https://github.com/thatjpcsguy/protohost/releases/latest/download/protohost-$(uname -s)-$(uname -m) -o /usr/local/bin/protohost
chmod +x /usr/local/bin/protohost
```

### Initialize Your Project

```bash
cd your-project
protohost init
```

This creates:
- `.protohost.config` - Configuration file
- `.protohost/hooks/` - Directory for deployment hooks
- Updates `.gitignore` to exclude `.protohost.config.local`

### Configure

Edit `.protohost.config`:

```bash
# Project configuration
PROJECT_PREFIX="myapp"              # Creates myapp-<branch> deployments
REPO_URL="git@github.com:org/repo.git"
TTL_DAYS=7                          # Auto-cleanup after 7 days

# Remote server configuration
REMOTE_HOST="protohost.xyz"
REMOTE_USER="${USER}"
REMOTE_BASE_DIR="~/protohost"
NGINX_PROXY_HOST="10.10.20.4"      # Where Docker runs
NGINX_SERVER="10.10.20.10"          # Where nginx runs

# Optional: Port configuration
BASE_WEB_PORT=3000
```

### Update docker-compose.yml

Ensure your `docker-compose.yml` uses `${WEB_PORT}`:

```yaml
services:
  web:
    ports:
      - "${WEB_PORT}:3000"  # External port is dynamic

  mysql:
    # No external ports needed - internal to network only

  redis:
    # No external ports needed - internal to network only
```

### Deploy

```bash
# Deploy locally
protohost deploy

# Deploy to remote server
protohost deploy --remote

# Clean deploy (removes volumes)
protohost deploy --clean

# Force rebuild
protohost deploy --build
```

## Commands

### `protohost init`
Initialize protohost in current project.

### `protohost deploy [flags]`
Deploy current branch.

**Flags:**
- `--remote` - Deploy to remote server
- `--clean` - Remove everything before deploying (includes volumes)
- `--build` - Force rebuild containers
- `--branch NAME` - Override current branch
- `--auto-bootstrap` - Automatically install protohost on remote if missing

### `protohost list [flags]`
List all deployments.

**Flags:**
- `--remote` - List remote deployments

### `protohost logs [flags]`
View logs for current branch deployment.

**Flags:**
- `--remote` - View remote logs
- `--follow, -f` - Follow log output
- `--branch NAME` - View logs for different branch

### `protohost down [flags]`
Stop deployment.

**Flags:**
- `--remote` - Stop remote deployment
- `--remove-volumes, -v` - Remove volumes
- `--branch NAME` - Stop different branch

### `protohost info [flags]`
Show deployment info.

**Flags:**
- `--remote` - Show remote deployment info

### `protohost cleanup [flags]`
Remove expired deployments.

**Flags:**
- `--remote` - Cleanup remote deployments
- `--dry-run` - Show what would be removed

### `protohost bootstrap-remote`
Install protohost on remote server (first-time setup).

## How It Works

### Port Management

Protohost uses a SQLite database (`~/.protohost/registry.db`) to track port allocations:

- Each deployment gets a unique web port
- MySQL and Redis run in isolated Docker networks (no external ports needed)
- Ports are automatically allocated from a configurable range (default: 3000-3099)
- Expired deployments automatically release their ports

### Local Deployments

1. Detects current git branch
2. Allocates a port from the local registry
3. Clones/pulls repo to `~/.protohost/deployments/{project}-{branch}`
4. Starts containers with allocated port
5. Tracks deployment in registry with TTL

### Remote Deployments

When you run `protohost deploy --remote`:

1. Loads config from `.protohost.config`
2. SSHs to remote server
3. Clones/pulls repo on remote
4. Runs `protohost deploy` on remote (uses remote's own registry)
5. Streams output back to your terminal

**You don't need to:**
- SSH manually
- Manage directories on remote
- Track which branches are deployed where

## Hooks

Customize deployment behavior with hooks:

### File-based hooks (`.protohost/hooks/`)

Create executable bash scripts:

- `pre-deploy.sh` - Runs locally before deployment
- `post-deploy.sh` - Runs locally after deployment
- `post-start.sh` - Runs on target after containers start
- `first-install.sh` - Runs on target only on first deployment

**Example: post-start.sh**
```bash
#!/bin/bash
# Run database migrations
docker compose -p $PROJECT_NAME exec web python manage.py migrate
```

### Config-based hooks (`.protohost.config`)

Fallback if hook files don't exist:

```bash
POST_START_SCRIPT="docker compose -p \$PROJECT_NAME exec web npm run migrate"
FIRST_INSTALL_SCRIPT="docker compose -p \$PROJECT_NAME exec web npm run seed"
```

## Architecture

### Single Binary

Protohost is a single Go binary with no runtime dependencies (except Docker and Git).

### Port Registry

Each protohost instance maintains its own SQLite database:
- Local: `~/.protohost/registry.db`
- Remote: `{REMOTE_BASE_DIR}/.protohost/registry.db`

No synchronization needed - each instance is independent.

### Docker Network Isolation

Each deployment gets its own Docker network:
- Only web port is exposed externally
- MySQL, Redis, etc. are internal to the network
- Simplified port management (only track web ports)

## Remote Setup

### First-Time Remote Deployment

**Option 1: Bootstrap command (recommended)**
```bash
protohost bootstrap-remote
```

**Option 2: Manual install on remote**
```bash
ssh user@remote
curl -sSL https://raw.githubusercontent.com/thatjpcsguy/protohost/main/install.sh | bash
```

**Option 3: Auto-bootstrap during deploy**
```bash
protohost deploy --remote --auto-bootstrap
```

## Configuration

### Required Fields

- `PROJECT_PREFIX` - Prefix for deployment names
- `REPO_URL` - Git repository URL
- `REMOTE_HOST` - SSH hostname
- `REMOTE_USER` - SSH username
- `REMOTE_BASE_DIR` - Base directory for deployments
- `NGINX_PROXY_HOST` - IP where Docker runs
- `NGINX_SERVER` - IP where nginx runs

### Optional Fields

- `TTL_DAYS` - Days until auto-cleanup (default: 7)
- `BASE_WEB_PORT` - Starting port (default: 3000)
- `SSL_CERT_PATH` - SSL certificate path
- `SSL_KEY_PATH` - SSL key path
- Hook scripts (see Hooks section)

### Local Overrides

Create `.protohost.config.local` for machine-specific overrides (gitignored):

```bash
# Override remote host for testing
REMOTE_HOST="staging.example.com"
```

## Examples

### Deploy feature branch locally
```bash
git checkout feature-123
protohost deploy
```

### Deploy to staging
```bash
protohost deploy --remote
```

### Clean deploy with rebuild
```bash
protohost deploy --remote --clean --build
```

### View logs
```bash
# Local
protohost logs -f

# Remote
protohost logs --remote -f
```

### List all deployments
```bash
# Local
protohost list

# Remote
protohost list --remote
```

### Cleanup expired deployments
```bash
# Dry run first
protohost cleanup --dry-run

# Actually remove
protohost cleanup
```

## Comparison with Previous Version

### Before (Makefile-based)
```bash
make deploy
make logs
make down
```

### Now (protohost CLI)
```bash
protohost deploy
protohost logs
protohost down
```

### Benefits
- ✅ Single binary, easy to install
- ✅ Works on any machine (Linux, macOS)
- ✅ Better error messages
- ✅ Cleaner architecture
- ✅ Same commands for local and remote (`--remote` flag)
- ✅ No Makefiles needed in your project

## Development

### Build
```bash
make build
```

### Install
```bash
make install
```

### Clean
```bash
make clean
```

## License

MIT License - see [LICENSE](LICENSE)

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md)
