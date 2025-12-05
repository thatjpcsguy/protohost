# Protohost Deploy - Setup Guide

This guide walks through setting up protohost-deploy for the first time, both on your local machine and on the remote server.

## Prerequisites

### Local Machine

1. **Git** - For cloning repositories
2. **Docker & Docker Compose V2** - For running containers locally
   ```bash
   docker --version  # Should be 20.10+
   docker compose version  # Should be 2.0+
   ```
3. **Make** - For running Makefile targets
4. **Python 3.6+** - For port allocation script
5. **SSH** - For deploying to remote server

### Remote Server (Protohost)

1. **Docker & Docker Compose V2** installed
2. **Git** installed
3. **Python 3.6+** installed
4. **SSH access** with key-based authentication
5. **Directory permissions** - User should be able to create `~/protohost` directory

### Nginx Server (If separate from protohost)

1. **Nginx** with SSL configured
2. **SSH access** with sudo privileges
3. **Wildcard SSL certificate** (e.g., `*.protohost.xyz`)

## Server Setup

### 1. Set Up Remote Server

SSH into your protohost server and verify prerequisites:

```bash
ssh your-user@protohost.xyz

# Verify Docker
docker --version
docker compose version

# Verify Python
python3 --version

# Create base directory
mkdir -p ~/protohost
cd ~/protohost

# Create subdirectories
mkdir -p .ports .nginx
```

### 2. Set Up Nginx Server

If nginx is running on a separate server:

```bash
ssh your-user@nginx-server

# Verify nginx
sudo nginx -v

# Check SSL certificate
sudo ls -l /etc/letsencrypt/live/protohost.xyz/

# Create ssl-params.conf if it doesn't exist
sudo nano /etc/nginx/ssl-params.conf
```

Example `ssl-params.conf`:

```nginx
ssl_protocols TLSv1.2 TLSv1.3;
ssl_prefer_server_ciphers on;
ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512:ECDHE-RSA-AES256-GCM-SHA384:DHE-RSA-AES256-GCM-SHA384;
ssl_session_timeout 10m;
ssl_session_cache shared:SSL:10m;
ssl_session_tickets off;
ssl_stapling on;
ssl_stapling_verify on;
resolver 8.8.8.8 8.8.4.4 valid=300s;
resolver_timeout 5s;
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";
```

### 3. Set Up SSH Keys

Ensure you have SSH key authentication set up:

```bash
# On your local machine
ssh-keygen -t ed25519 -C "your-email@example.com"

# Copy to remote servers
ssh-copy-id your-user@protohost.xyz
ssh-copy-id your-user@nginx-server  # If separate

# Test connections
ssh your-user@protohost.xyz "echo 'Connected to protohost'"
ssh nginx-server "echo 'Connected to nginx'"
```

### 4. Configure SSH Access Between Servers

If nginx is on a separate server, the protohost server needs SSH access to nginx:

```bash
# On protohost server
ssh-keygen -t ed25519 -C "protohost-to-nginx"
ssh-copy-id nginx-server

# Test
ssh nginx-server "echo 'Can connect'"
```

## Project Setup

### 1. Install in Your Project

From your project directory:

```bash
cd /path/to/your/project

# Clone protohost-deploy (or download)
git clone git@github.com:thatjpcsguy/protohost.git /tmp/protohost-deploy

# Run installer
/tmp/protohost-deploy/install.sh

# Or use curl (once published)
curl -sSL https://raw.githubusercontent.com/thatjpcsguy/protohost/main/install.sh | bash
```

### 2. Configure Your Project

Edit `.protohost.config`:

```bash
# Remote server configuration
REMOTE_HOST="protohost.xyz"
REMOTE_USER="james"
REMOTE_BASE_DIR="~/protohost"

# Network configuration
NGINX_PROXY_HOST="10.10.20.4"    # IP where Docker containers run
NGINX_SERVER="10.10.20.10"        # IP where nginx runs

# Project configuration
PROJECT_PREFIX="myapp"

# Git repository
REPO_URL="git@github.com:yourorg/yourrepo.git"

# Deployment settings
TTL_DAYS=7
```

**Important**: If `NGINX_PROXY_HOST` and `NGINX_SERVER` are the same machine, use the same IP.

### 3. Update docker-compose.yml

Ensure your `docker-compose.yml` uses environment variables for ports:

```yaml
services:
  web:
    build: .
    ports:
      - "${WEB_PORT:-3000}:3000"  # Use env var with default
    # ... rest of config

  mysql:
    image: mysql:8.0
    ports:
      - "${MYSQL_PORT:-3306}:3306"
    # ... rest of config

  redis:
    image: redis:7-alpine
    ports:
      - "${REDIS_PORT:-6379}:6379"
    # ... rest of config
```

**Key points**:
- Use `${VAR:-default}` syntax for backwards compatibility
- Port mapping format: `"${HOST_PORT}:CONTAINER_PORT"`
- Container port (right side) stays the same
- Host port (left side) uses environment variable

### 4. Test Locally

```bash
# Start services
make up

# Check they're running
make info

# View logs
make logs

# Stop services
make down
```

### 5. Deploy to Protohost

```bash
# First deployment
make deploy

# Or with database initialization
make deploy RESET_DB=true
```

Your deployment will be available at:
- `https://{PROJECT_PREFIX}-{BRANCH}.protohost.xyz`
- Example: `https://myapp-main.protohost.xyz`

## Configuration Options

### Environment Variables

You can override configuration in `.protohost.config` with environment variables:

```bash
REMOTE_HOST=staging.example.com make deploy
```

### Custom Base Ports

If default ports (3000, 3306, 6379) conflict with other services:

```bash
# In .protohost.config
BASE_WEB_PORT=4000
BASE_MYSQL_PORT=4306
BASE_REDIS_PORT=7379
```

### Custom TTL

Change how long deployments stay active:

```bash
# In .protohost.config
TTL_DAYS=14  # Keep deployments for 2 weeks
```

## Post-Installation Hooks

Create custom scripts to run at different deployment stages:

### Post-Start Hook

Runs after containers start (useful for data generation):

```bash
# .protohost/hooks/post-start.sh
#!/bin/bash
echo "Generating sample data..."
docker compose -p $PROJECT_NAME exec web python scripts/generate_data.py
```

### Pre-Deploy Hook

Runs before deployment starts:

```bash
# .protohost/hooks/pre-deploy.sh
#!/bin/bash
echo "Running tests before deployment..."
npm test
```

### Post-Deploy Hook

Runs after successful deployment:

```bash
# .protohost/hooks/post-deploy.sh
#!/bin/bash
echo "Sending notification..."
curl -X POST https://slack.com/api/webhook -d "Deployed ${PROJECT_NAME}"
```

Make hooks executable:

```bash
chmod +x .protohost/hooks/*.sh
```

## Troubleshooting

### SSH Connection Issues

```bash
# Test SSH connection
ssh your-user@protohost.xyz "echo 'Connected'"

# Check SSH key is loaded
ssh-add -l

# Add key if needed
ssh-add ~/.ssh/id_ed25519
```

### Docker Not Found

```bash
# On remote server, add docker to PATH
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Port Conflicts

```bash
# Check what's using a port
lsof -i :3000

# Or on Linux
netstat -tuln | grep 3000

# Let protohost-deploy auto-allocate
make up  # Will find next available slot
```

### Nginx Configuration Issues

```bash
# Test nginx config
make nginx-test

# Check nginx error logs
ssh nginx-server "sudo tail -f /var/log/nginx/error.log"

# Restart nginx
ssh nginx-server "sudo service nginx restart"
```

### Permissions Issues

```bash
# On remote server, check directory permissions
ls -la ~/protohost

# Fix if needed
chmod 755 ~/protohost
```

## Updating Protohost Deploy

To update to the latest version, simply run:

```bash
make update-protohost
```

This automated command will:
- Clone/update the latest version to `/tmp/protohost-deploy`
- Update symlinks to latest scripts
- Preserve your `.protohost.config` and `.protohost.config.local`
- Update `.protohost/Makefile.inc`

**Manual update (alternative):**

```bash
cd /tmp
git clone git@github.com:thatjpcsguy/protohost.git
cd /path/to/your/project
/tmp/protohost-deploy/install.sh
```

## Uninstalling

To remove protohost-deploy from a project:

```bash
# Stop any running deployments
make down

# Remove protohost files
rm -rf .protohost
rm .protohost.config

# Remove from Makefile (if you want)
# Edit Makefile and remove the "include .protohost/Makefile.inc" line

# Remove from .gitignore (optional)
# Edit .gitignore and remove .protohost/ entry
```

## Next Steps

- Read the [README.md](../README.md) for usage examples
- Check out [ARCHITECTURE.md](ARCHITECTURE.md) for how it works
- Set up [monitoring and alerts](MONITORING.md) for your deployments
