# Protohost Deploy

A drop-in deployment tool for managing multi-branch Docker Compose deployments to remote servers with automatic port allocation and nginx configuration.

## Features

- 🚀 **Branch-based deployments**: Deploy multiple branches simultaneously without port conflicts
- 🔄 **Automatic port allocation**: Smart port management for web, MySQL, and Redis services
- 🌐 **Nginx integration**: Automatic reverse proxy configuration with SSL
- 🧹 **Auto-cleanup**: Automatically removes expired deployments (7-day default TTL)
- 📊 **Deployment tracking**: List and manage all running deployments
- 🔧 **Easy integration**: Add to any Docker Compose project in minutes

## Quick Start

### 1. Install in Your Project

```bash
# From your project directory
curl -sSL https://raw.githubusercontent.com/thatjpcsguy/protohost/main/install.sh | bash
```

Or clone and install manually:

```bash
# Clone the repo
git clone git@github.com:thatjpcsguy/protohost.git /tmp/protohost-deploy

# Install in your project
cd /path/to/your/project
/tmp/protohost-deploy/install.sh
```

### 2. Configure

Copy the example config and customize it:

```bash
cp .protohost.config.example .protohost.config
# Edit .protohost.config with your settings
```

Example configuration:

```bash
REMOTE_HOST="protohost.xyz"
REMOTE_USER="your-username"
REMOTE_BASE_DIR="~/protohost"
NGINX_PROXY_HOST="10.10.20.4"    # Host where Docker runs
NGINX_SERVER="10.10.20.10"        # Host where nginx runs
PROJECT_PREFIX="myapp"             # Creates myapp-<branch> deployments
REPO_URL="git@github.com:org/repo.git"
```

**Configuration files:**
- `.protohost.config.example` - Template (committed to git)
- `.protohost.config` - Your local config (gitignored)
- `.protohost.config.local` - Optional local overrides (gitignored)

### 3. Ensure Docker Compose Supports Dynamic Ports

Update your `docker-compose.yml` to use environment variables:

```yaml
services:
  web:
    ports:
      - "${WEB_PORT:-3000}:3000"

  mysql:
    ports:
      - "${MYSQL_PORT:-3306}:3306"

  redis:
    ports:
      - "${REDIS_PORT:-6379}:6379"
```

### 4. Deploy!

```bash
# Deploy current branch
make deploy

# Deploy with database reset
make deploy RESET_DB=true

# Nuke and redeploy (removes everything first)
make deploy NUKE=true

# Deploy to different host
make deploy HOST=staging.example.com
```

## Usage

After installation, your project gains these Makefile targets:

### Local Development

```bash
make up              # Start services for current branch
make down            # Stop services for current branch
make logs            # View logs
make info            # Show connection details
make restart         # Restart services
```

### Remote Deployment

```bash
make deploy                    # Deploy current branch to protohost
make deploy RESET_DB=true      # Deploy and regenerate data
make deploy NUKE=true          # Nuke and fresh deploy
make deploy HOST=custom.host   # Deploy to custom host
```

### Nginx Management

```bash
make nginx-config    # Generate and deploy nginx config
make nginx-enable    # Enable nginx for this deployment
make nginx-disable   # Disable nginx for this deployment
make nginx-list      # List all nginx configs
```

### Deployment Management

```bash
make list-all        # List all running deployments
```

### Maintenance

```bash
make update-protohost  # Update protohost-deploy to latest version
```

## How It Works

### Port Allocation

The system automatically allocates ports for each branch:
- `main` branch: Uses default ports (3000, 3306, 6379)
- Other branches: Finds next available slot (3001, 3307, 6380, etc.)
- Reuses existing ports if containers are already running

### Project Naming

Deployments are named `{PROJECT_PREFIX}-{BRANCH}`:
- `main` → `myapp-main`
- `feature-xyz` → `myapp-feature-xyz`

### Nginx Configuration

Each deployment gets:
- Subdomain: `https://{PROJECT_PREFIX}-{BRANCH}.protohost.xyz`
- SSL certificate (shared wildcard cert)
- Reverse proxy to allocated port

### Automatic Cleanup

Deployments have a 7-day TTL by default:
- Tracked in `.ports/` directory with expiration timestamps
- Next deployment automatically cleans up expired ones
- Stops containers, removes nginx config, deletes files

## Architecture

```
your-project/
├── .protohost/              # Created by install.sh
│   ├── bin/                 # Symlinks to protohost-deploy binaries
│   ├── lib/                 # Symlinks to protohost-deploy libraries
│   └── config              # Symlink to protohost-deploy config
├── .protohost.config        # Your project configuration
├── Makefile                 # Modified to include protohost targets
└── docker-compose.yml       # Your existing docker-compose file
```

On the remote server:

```
~/protohost/
├── .ports/                  # Port allocation tracking
│   ├── myapp-main          # Stores WEB_PORT, MYSQL_PORT, REDIS_PORT, EXPIRES
│   └── myapp-feature-xyz
├── .nginx/                  # Generated nginx configs
│   ├── myapp-main.conf
│   └── myapp-feature-xyz.conf
├── myapp-main/              # Cloned repository for main branch
│   └── docker-compose.yml
└── myapp-feature-xyz/       # Cloned repository for feature branch
    └── docker-compose.yml
```

## Requirements

### Local Machine
- Git
- Docker & Docker Compose V2
- Make
- Python 3.6+
- SSH access to remote server

### Remote Server (Protohost)
- Docker & Docker Compose V2
- Git
- Python 3.6+
- SSH key authentication

### Nginx Server
- Nginx with SSL configured
- SSH access with sudo privileges for nginx management

## Configuration Reference

### .protohost.config

| Variable | Description | Example |
|----------|-------------|---------|
| `REMOTE_HOST` | SSH hostname for deployment target | `protohost.xyz` |
| `REMOTE_USER` | SSH username | `james` |
| `REMOTE_BASE_DIR` | Base directory for all deployments | `~/protohost` |
| `NGINX_PROXY_HOST` | IP where Docker containers run | `10.10.20.4` |
| `NGINX_SERVER` | IP/hostname of nginx server | `10.10.20.10` |
| `PROJECT_PREFIX` | Prefix for deployment names | `myapp` |
| `REPO_URL` | Git repository URL | `git@github.com:org/repo.git` |
| `TTL_DAYS` | Days until deployment expires | `7` |

### Configuration Loading Order

Configuration is loaded in this order (later values override earlier ones):

1. `.protohost.config` - Main configuration
2. `.protohost.config.local` - Local overrides (optional)

**Example use case for `.protohost.config.local`:**

```bash
# .protohost.config (committed, shared by team)
REMOTE_HOST="protohost.xyz"
REMOTE_USER="deploy"
PROJECT_PREFIX="myapp"

# .protohost.config.local (gitignored, personal)
REMOTE_HOST="dev.protohost.xyz"  # Use personal dev server
REMOTE_USER="john"                # Use your SSH username
```

This allows team members to:
- Share common configuration via `.protohost.config.example`
- Keep personal settings in `.protohost.config` (gitignored)
- Override specific values in `.protohost.config.local` without modifying the main config

### Optional Hooks

Create these scripts in your project to customize behavior:

- `.protohost/hooks/pre-deploy.sh` - Runs before deployment
- `.protohost/hooks/post-deploy.sh` - Runs after successful deployment
- `.protohost/hooks/post-start.sh` - Runs after containers start

Example post-start hook for data generation:

```bash
#!/bin/bash
# .protohost/hooks/post-start.sh
docker compose -p $PROJECT_NAME exec web python scripts/generate_data.py
```

## Troubleshooting

### Port conflicts

If ports are already in use, the system will automatically find the next available slot. To check current allocations:

```bash
make list-all
```

### Deployment not accessible

1. Check nginx is configured:
   ```bash
   make nginx-list
   ```

2. Enable if needed:
   ```bash
   make nginx-enable
   ```

3. Check logs:
   ```bash
   make logs
   ```

### Clean up old deployments

Old deployments auto-cleanup after 7 days. To manually remove:

```bash
# On remote server
cd ~/protohost/myapp-old-branch
make down
cd ..
rm -rf myapp-old-branch
rm -f .ports/myapp-old-branch
make nginx-disable PROJECT=myapp-old-branch
```

## Updating protohost-deploy

To update to the latest version, use the built-in update command:

```bash
make update-protohost
```

This will:
- Clone/update protohost-deploy to `/tmp/protohost-deploy`
- Re-run the installer to update symlinks
- Preserve your `.protohost.config` and `.protohost.config.local`
- Update `.protohost/Makefile.inc` with new features

**Alternative:** You can also run the install script directly:

```bash
curl -sSL https://raw.githubusercontent.com/thatjpcsguy/protohost/main/install.sh | bash
```

## Contributing

Contributions welcome! Please:
1. Fork the repository
2. Create a feature branch
3. Test with a sample project
4. Submit a pull request

## License

MIT License - See LICENSE file for details
