# Protohost Deploy - Architecture

This document explains how protohost-deploy works under the hood.

## Overview

Protohost-deploy is a deployment orchestration tool that enables multiple branch-based deployments of Docker Compose applications on a single server without port conflicts.

## Core Concepts

### 1. Branch-Based Isolation

Each git branch gets its own isolated deployment:

```
main branch        → myapp-main        → ports 3000, 3306, 6379
feature-x branch   → myapp-feature-x   → ports 3001, 3307, 6380
bugfix-y branch    → myapp-bugfix-y    → ports 3002, 3308, 6381
```

**Key components**:
- Project name: `{PROJECT_PREFIX}-{BRANCH_NAME}`
- Docker Compose project: `-p {PROJECT_NAME}`
- Separate volumes, networks, and containers per project
- No conflicts between branches

### 2. Dynamic Port Allocation

The `get_ports.py` script intelligently allocates ports:

```python
def find_free_slot():
    offset = 0
    while offset < 100:
        web = 3000 + offset
        mysql = 3306 + offset
        redis = 6379 + offset

        if all ports free:
            return ports
        offset += 1
```

**Algorithm**:
1. Check if project already running → reuse existing ports
2. If not, find first available "slot" (offset)
3. A slot is three consecutive port numbers across services
4. Avoids port conflicts automatically

**Port Tracking**:
```bash
~/protohost/.ports/myapp-main
  WEB_PORT=3000
  MYSQL_PORT=3306
  REDIS_PORT=6379
  BRANCH=main
  CREATED=2025-12-05T10:00:00Z
  EXPIRES=2025-12-12T10:00:00Z
```

### 3. Nginx Reverse Proxy

Each deployment gets its own subdomain:

```
https://myapp-main.protohost.xyz → localhost:3000
https://myapp-feature-x.protohost.xyz → localhost:3001
```

**Nginx Config Generation**:
1. `make nginx-config` generates config from template
2. Config stored in `~/protohost/.nginx/{PROJECT_NAME}.conf`
3. SCP'd to nginx server
4. Moved to `/etc/nginx/sites-enabled/protohost-{PROJECT_NAME}.conf`
5. Nginx restarted

**Config Template**:
```nginx
server {
    listen 443 ssl;
    server_name myapp-main.protohost.xyz;

    ssl_certificate /etc/letsencrypt/live/protohost.xyz/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/protohost.xyz/privkey.pem;
    include ssl-params.conf;

    location / {
        proxy_pass http://10.10.20.4:3000;  # NGINX_PROXY_HOST:WEB_PORT
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        # ... more headers
    }
}
```

### 4. Lifecycle Management

Deployments have a time-to-live (TTL):

```
Day 0: Deployment created
  CREATED=2025-12-05T10:00:00Z
  EXPIRES=2025-12-12T10:00:00Z  (TTL_DAYS=7)

Day 7: Deployment expires
  Status: Active (grace period)

Day 8+: Next deployment triggers cleanup
  - Stop containers
  - Remove files
  - Delete nginx config
  - Clean up .ports file
```

**Cleanup Algorithm** (in `deploy.sh`):
```bash
for deployment in all_deployments:
    if EXPIRES < CURRENT_TIME:
        docker compose -p $deployment down
        rm -rf $deployment
        rm .ports/$deployment
        rm .nginx/$deployment.conf
        ssh nginx "rm /etc/nginx/sites-enabled/protohost-$deployment.conf"
```

## Deployment Flow

### Local Start (`make up`)

```
1. Detect current branch
   └─> BRANCH=$(git branch --show-current)

2. Generate project name
   └─> PROJECT_NAME={PROJECT_PREFIX}-{BRANCH}

3. Allocate ports
   └─> python3 get_ports.py $PROJECT_NAME
   └─> Returns: WEB_PORT MYSQL_PORT REDIS_PORT

4. Start containers
   └─> WEB_PORT=3000 MYSQL_PORT=3306 REDIS_PORT=6379 \
       docker compose -p $PROJECT_NAME up -d

5. Save port allocation
   └─> Write to .ports/$PROJECT_NAME
```

### Remote Deployment (`make deploy`)

```
1. Pre-deploy hook (if exists)
   └─> Run .protohost/hooks/pre-deploy.sh

2. SSH to remote server
   └─> ssh user@protohost.xyz

3. On remote server:
   a. Cleanup expired deployments
      └─> Check all .ports/* for EXPIRES < now
      └─> Stop and remove expired ones

   b. Nuke if requested (--nuke flag)
      └─> Stop containers
      └─> Remove directory
      └─> Delete port files

   c. Clone or update repository
      └─> If new: git clone -b $BRANCH $REPO_URL $PROJECT_NAME
      └─> If exists: cd $PROJECT_NAME && git pull

   d. Build containers
      └─> docker compose build

   e. Start application
      └─> make up (uses port allocation logic)

   f. Post-start hook (if exists)
      └─> Run .protohost/hooks/post-start.sh

   g. Generate and deploy nginx config
      └─> make nginx-config
      └─> SCP to nginx server
      └─> Restart nginx

4. Post-deploy hook (if exists)
   └─> Run .protohost/hooks/post-deploy.sh locally
```

## File Structure

### Local Project

```
your-project/
├── .protohost/
│   ├── bin/                # (Future: CLI commands)
│   ├── lib/                # Symlinks to protohost-deploy scripts
│   │   ├── deploy.sh       → /path/to/protohost-deploy/lib/deploy.sh
│   │   ├── get_ports.py    → /path/to/protohost-deploy/lib/get_ports.py
│   │   ├── list_deployments.sh
│   │   └── nginx_manage.sh
│   ├── hooks/              # Custom project hooks
│   │   ├── pre-deploy.sh
│   │   ├── post-deploy.sh
│   │   └── post-start.sh
│   └── Makefile.inc        # Generated Makefile targets
├── .protohost.config       # Project configuration
├── Makefile                # Includes .protohost/Makefile.inc
└── docker-compose.yml      # Your Docker Compose config
```

### Remote Server

```
~/protohost/
├── .ports/                 # Port allocation tracking
│   ├── myapp-main
│   ├── myapp-feature-x
│   └── myapp-bugfix-y
├── .nginx/                 # Generated nginx configs
│   ├── myapp-main.conf
│   ├── myapp-feature-x.conf
│   └── myapp-bugfix-y.conf
├── myapp-main/             # Cloned repository (main branch)
│   ├── .git/
│   ├── docker-compose.yml
│   ├── Makefile
│   └── ... (project files)
├── myapp-feature-x/        # Cloned repository (feature-x branch)
│   └── ...
└── myapp-bugfix-y/         # Cloned repository (bugfix-y branch)
    └── ...
```

### Nginx Server

```
/etc/nginx/
├── sites-enabled/
│   ├── protohost-myapp-main.conf
│   ├── protohost-myapp-feature-x.conf
│   └── protohost-myapp-bugfix-y.conf
└── ssl-params.conf
```

## Configuration Loading

### Precedence

1. **Command-line arguments** (highest priority)
   ```bash
   make deploy HOST=staging.example.com
   ```

2. **Environment variables**
   ```bash
   REMOTE_HOST=staging make deploy
   ```

3. **.protohost.config file**
   ```bash
   REMOTE_HOST="protohost.xyz"
   ```

4. **Default values** (lowest priority)
   - Defined in scripts
   - Example: `TTL_DAYS=7`

### Config Loading in Scripts

```bash
# In deploy.sh
source .protohost.config  # Load project config

# Override with environment
REMOTE_HOST="${REMOTE_HOST:-protohost.xyz}"

# Override with flags
while [[ $# -gt 0 ]]; do
  case $1 in
    --host)
      REMOTE_HOST="$2"
      ;;
  esac
done
```

## Docker Compose Integration

### Environment Variable Passing

```bash
# In Makefile
WEB_PORT=$(WEB_PORT) MYSQL_PORT=$(MYSQL_PORT) REDIS_PORT=$(REDIS_PORT) \
  docker compose -p $(PROJECT_NAME) up -d
```

### docker-compose.yml

```yaml
services:
  web:
    ports:
      - "${WEB_PORT:-3000}:3000"  # ${VAR:-default}
```

**Explanation**:
- `${WEB_PORT:-3000}` uses `WEB_PORT` if set, otherwise `3000`
- Make passes `WEB_PORT=3001` as environment variable
- Docker Compose expands: `"3001:3000"`
- Binds host port 3001 to container port 3000

### Project Isolation

```bash
docker compose -p myapp-main up     # Separate project
docker compose -p myapp-feature-x up  # Separate project
```

**Isolates**:
- Containers: `myapp-main-web-1`, `myapp-feature-x-web-1`
- Networks: `myapp-main_default`, `myapp-feature-x_default`
- Volumes: `myapp-main_mysql_data`, `myapp-feature-x_mysql_data`

## Security Considerations

### SSH Key Authentication

- All SSH connections use key-based authentication
- No passwords stored or transmitted
- Keys should be passphrase-protected

### Nginx Configuration

- SSL/TLS only (no HTTP)
- Wildcard certificate (`*.protohost.xyz`)
- Security headers in `ssl-params.conf`
- Reverse proxy hides container ports

### Port Binding

- Containers bind to `localhost` only by default
- Nginx proxies from public interface
- Direct container access requires SSH tunnel

### File Permissions

- Deployment files owned by deployment user
- Nginx configs require sudo to modify
- `.protohost.config` may contain sensitive data

## Performance Characteristics

### Port Allocation

- **Time complexity**: O(n) where n = number of used slots
- **Typical**: < 100ms for projects with < 10 deployments
- **Worst case**: ~5 seconds if checking 100 slots

### Deployment Time

Typical deployment timeline:
```
00:00 - SSH connect
00:01 - Cleanup check (fast if few deployments)
00:05 - Git clone/pull
00:15 - docker compose build (varies by image size)
00:20 - docker compose up (with health checks)
00:25 - Post-start hooks (if data generation needed)
00:30 - Nginx config generation and restart
00:35 - Complete
```

**Optimization tips**:
- Use Docker build cache
- Layer Dockerfiles efficiently
- Keep images small
- Use health checks to detect "ready" state

### Resource Usage

Per deployment (typical Flask app):
- **CPU**: 1-5% idle, 20-50% under load
- **Memory**: 500MB-2GB (depends on app)
- **Disk**: 1-5GB (code + volumes)
- **Network**: Minimal (nginx compression enabled)

## Scalability

### Current Limits

- **Max deployments**: ~100 (port range limit)
- **Max concurrent builds**: Limited by Docker daemon
- **Network**: All containers on same host network

### Extending Beyond Limits

To support more deployments:

1. **Use multiple port ranges**:
   ```bash
   BASE_WEB_PORT=3000   # First 100: 3000-3099
   BASE_WEB_PORT=4000   # Next 100: 4000-4099
   ```

2. **Use Docker networks**:
   - Don't expose MySQL/Redis ports
   - Only expose web service
   - Reduces port consumption

3. **Multiple protohost servers**:
   - Route branches to different servers
   - Use DNS round-robin or load balancer

## Future Enhancements

Potential improvements:

1. **CLI Tool**: `protohost deploy` instead of `make deploy`
2. **Web Dashboard**: View all deployments in browser
3. **Metrics & Monitoring**: Prometheus/Grafana integration
4. **Auto-scaling**: Automatically allocate resources
5. **Multi-host Support**: Deploy across multiple servers
6. **Container Registry**: Push/pull from private registry
7. **Rollback Support**: One-command rollback to previous version
8. **Deployment Snapshots**: Save/restore deployment state

## Troubleshooting

### Debug Mode

Enable verbose output:

```bash
# In deploy.sh
set -x  # Print every command
bash -x .protohost/lib/deploy.sh
```

### Common Issues

1. **Port conflicts**: Run `make list-all` to see allocations
2. **SSH failures**: Test with `ssh -v user@host`
3. **Docker build fails**: Check Dockerfile, clear cache
4. **Nginx 502 errors**: Container not healthy, check logs
5. **Expired deployments not cleaning**: Check EXPIRES format

### Logging

View logs:
```bash
# Application logs
make logs

# Docker daemon logs
sudo journalctl -u docker -f

# Nginx logs
ssh nginx-server "sudo tail -f /var/log/nginx/error.log"
```

## Contributing

To contribute to protohost-deploy:

1. Fork the repository
2. Create a feature branch
3. Test with real projects
4. Document changes in this file
5. Submit pull request

Key areas for contribution:
- Performance optimization
- Error handling
- Documentation
- Platform support (Windows, etc.)
- Testing framework
