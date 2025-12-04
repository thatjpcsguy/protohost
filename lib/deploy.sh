#!/bin/bash
set -e

# Protohost Deploy - Remote Deployment Script
# Deploys Docker Compose projects to remote servers with branch isolation

# Load configuration from .protohost.config
if [ ! -f ".protohost.config" ]; then
    echo "❌ Error: .protohost.config not found"
    echo "   Run './install.sh' first to set up protohost-deploy"
    exit 1
fi

source .protohost.config

# Validate required config
if [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_USER" ] || [ -z "$REMOTE_BASE_DIR" ] || [ -z "$PROJECT_PREFIX" ] || [ -z "$REPO_URL" ]; then
    echo "❌ Error: Missing required configuration in .protohost.config"
    echo "   Required: REMOTE_HOST, REMOTE_USER, REMOTE_BASE_DIR, PROJECT_PREFIX, REPO_URL"
    exit 1
fi

# Get current branch
BRANCH=$(git branch --show-current)
if [ -z "$BRANCH" ]; then
    echo "❌ Error: Could not determine current git branch"
    exit 1
fi

PROJECT_NAME="${PROJECT_PREFIX}-${BRANCH}"
TTL_DAYS="${TTL_DAYS:-7}"

# Flags
RESET_DB=false
NUKE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --reset-db)
      RESET_DB=true
      shift
      ;;
    --nuke)
      NUKE=true
      shift
      ;;
    --host)
      REMOTE_HOST="$2"
      shift 2
      ;;
    --branch)
      BRANCH="$2"
      PROJECT_NAME="${PROJECT_PREFIX}-${BRANCH}"
      shift 2
      ;;
    *)
      echo "Unknown option: $1"
      echo "Usage: $0 [--reset-db] [--nuke] [--host HOST] [--branch BRANCH]"
      exit 1
      ;;
  esac
done

echo "🚀 Deploying branch '${BRANCH}' to ${REMOTE_USER}@${REMOTE_HOST}..."
echo "   Project: ${PROJECT_NAME}"
echo ""

# Check if pre-deploy hook exists
if [ -f ".protohost/hooks/pre-deploy.sh" ]; then
    echo "🪝 Running pre-deploy hook..."
    bash .protohost/hooks/pre-deploy.sh
    echo ""
fi

# Construct the remote command
REMOTE_CMD="
set -e

# Ensure docker is in PATH
export PATH=\"/usr/local/bin:\$PATH\"

# Verify docker command is available
if ! command -v docker >/dev/null 2>&1; then
    echo \"❌ Docker command not found in PATH\"
    exit 1
fi

# Ensure base directory exists
mkdir -p ${REMOTE_BASE_DIR}
cd ${REMOTE_BASE_DIR}

# Cleanup old deployments (older than TTL_DAYS)
echo \"🧹 Checking for old deployments to clean up...\"
CLEANUP_COUNT=0
NGINX_NEEDS_RESTART=false
if [ -d \".ports\" ]; then
    CURRENT_TIME=\$(date +%s)
    for port_file in .ports/*; do
        if [ -f \"\$port_file\" ]; then
            project=\$(basename \"\$port_file\")

            # Skip current deployment
            if [ \"\$project\" = \"${PROJECT_NAME}\" ]; then
                continue
            fi

            # Read expiration timestamp from port file
            expires=\$(grep '^EXPIRES=' \"\$port_file\" | cut -d'=' -f2)

            # Fallback to old TIMESTAMP field for backwards compatibility
            if [ -z \"\$expires\" ]; then
                expires=\$(grep '^TIMESTAMP=' \"\$port_file\" | cut -d'=' -f2)
            fi

            if [ -n \"\$expires\" ]; then
                # Convert timestamp to epoch (handles ISO 8601 format)
                # Linux uses -d, macOS uses -j, try both
                expiration_time=\$(date -d \"\$expires\" +%s 2>/dev/null || date -j -f \"%Y-%m-%dT%H:%M:%SZ\" \"\$expires\" +%s 2>/dev/null || echo \"0\")

                if [ \"\$expiration_time\" -gt 0 ]; then
                    if [ \"\$CURRENT_TIME\" -gt \"\$expiration_time\" ]; then
                        days_overdue=\$(( (CURRENT_TIME - expiration_time) / 86400 ))
                        echo \"   Removing expired deployment: \$project (expired \$days_overdue days ago)\"

                        # Stop and remove containers
                        if [ -d \"\$project\" ]; then
                            cd \"\$project\"
                            docker compose -p \"\$project\" down || true
                            cd ..
                        fi

                        # Remove nginx config from remote server
                        if ssh ${NGINX_SERVER} \"sudo rm -f /etc/nginx/sites-enabled/protohost-\$project.conf\" 2>/dev/null; then
                            NGINX_NEEDS_RESTART=true
                        fi

                        # Remove files
                        rm -rf \"\$project\"
                        rm -f .ports/\"\$project\"
                        rm -f .nginx/\"\$project.conf\"

                        echo \"   ✓ Removed \$project\"
                        CLEANUP_COUNT=\$((CLEANUP_COUNT + 1))
                    fi
                fi
            fi
        fi
    done
fi

if [ \"\$CLEANUP_COUNT\" -gt 0 ]; then
    echo \"   Cleaned up \$CLEANUP_COUNT old deployment(s)\"

    # Restart nginx if any configs were removed
    if [ \"\$NGINX_NEEDS_RESTART\" = \"true\" ]; then
        echo \"   Restarting nginx...\"
        ssh ${NGINX_SERVER} \"sudo service nginx restart\" 2>/dev/null || true
    fi
else
    echo \"   No old deployments to clean up\"
fi
echo \"\"

# Nuke option: Stop everything and delete directory
if [ \"${NUKE}\" = \"true\" ]; then
    echo \"💥 Nuke requested! Cleaning up...\"
    # Stop containers if running (ignore errors if not running)
    if [ -d \"${PROJECT_NAME}\" ]; then
        cd ${PROJECT_NAME}
        echo \"   Stopping containers...\"
        docker compose -p ${PROJECT_NAME} down || true
        cd ..
    fi

    echo \"   Removing directory ${PROJECT_NAME}...\"
    rm -rf ${PROJECT_NAME}

    echo \"   Removing port file...\"
    rm -f .ports/${PROJECT_NAME}

    echo \"   Removing nginx config...\"
    rm -f .nginx/${PROJECT_NAME}.conf

    echo \"   Removing nginx config from server...\"
    ssh ${NGINX_SERVER} \"sudo rm -f /etc/nginx/sites-enabled/protohost-${PROJECT_NAME}.conf\" 2>/dev/null || true
fi

# Clone or Pull
IS_NEW_INSTALL=false
if [ ! -d \"${PROJECT_NAME}\" ]; then
    echo \"📦 Cloning repository...\"
    git clone -b ${BRANCH} ${REPO_URL} ${PROJECT_NAME}
    IS_NEW_INSTALL=true
    cd ${PROJECT_NAME}
else
    echo \"🔄 Updating repository...\"
    cd ${PROJECT_NAME}
    git fetch origin
    git reset --hard origin/${BRANCH}
    git pull origin ${BRANCH}
fi

# Rebuild containers to pick up any changes (Dockerfile, requirements.txt, etc.)
echo \"🔨 Rebuilding Docker containers...\"
docker compose -p ${PROJECT_NAME} build

# Start Application (waits for health checks)
echo \"🚀 Starting application...\"
make up

# Run post-start hook if it exists
if [ -f \".protohost/hooks/post-start.sh\" ]; then
    echo \"🪝 Running post-start hook...\"
    export PROJECT_NAME=${PROJECT_NAME}
    bash .protohost/hooks/post-start.sh
fi

# Generate nginx config
echo \"📝 Generating nginx config...\"
make nginx-config

# Show Info
echo \"✅ Deployment complete!\"
make info
echo \"\"
echo \"🌐 Deployment is live at: https://${PROJECT_NAME}.${REMOTE_HOST}\"
echo \"📋 Nginx config deployed to: ${NGINX_SERVER}:/etc/nginx/sites-enabled/protohost-${PROJECT_NAME}.conf\"
echo \"\"
echo \"Useful commands:\"
echo \"  make logs              # View application logs\"
echo \"  make nginx-disable     # Disable nginx for this deployment\"
echo \"  make down              # Stop this deployment\"
"

# Execute via SSH
ssh "${REMOTE_USER}@${REMOTE_HOST}" "${REMOTE_CMD}"

# Run post-deploy hook if it exists
if [ -f ".protohost/hooks/post-deploy.sh" ]; then
    echo ""
    echo "🪝 Running post-deploy hook..."
    bash .protohost/hooks/post-deploy.sh
fi

echo ""
echo "✨ Deployment complete!"
