#!/bin/bash

# Protohost Deploy - List Deployments Script
# Lists all running deployments from the central .ports directory

# Load configuration
if [ -f ".protohost.config" ]; then
    source .protohost.config

    # Load local overrides if they exist
    if [ -f ".protohost.config.local" ]; then
        source .protohost.config.local
    fi
else
    # Fallback defaults if not in a project directory
    REMOTE_BASE_DIR="${REMOTE_BASE_DIR:-../protohost}"
    NGINX_SERVER="${NGINX_SERVER:-10.10.10.10}"
fi

PORTS_DIR="${REMOTE_BASE_DIR}/.ports"
NGINX_DIR="${REMOTE_BASE_DIR}/.nginx"
REMOTE_SITES_ENABLED="/etc/nginx/sites-enabled"

# Check if we're on the remote server or local machine
if [ -d "$PORTS_DIR" ]; then
    # We're on the remote server or .ports is local
    IS_REMOTE=false
else
    # We're on local machine, need to SSH
    IS_REMOTE=true
fi

if [ "$IS_REMOTE" = "true" ]; then
    # Run this script on the remote server
    if [ -z "$REMOTE_HOST" ] || [ -z "$REMOTE_USER" ]; then
        echo "❌ Error: Cannot determine remote host. Run from project directory with .protohost.config"
        exit 1
    fi

    ssh "${REMOTE_USER}@${REMOTE_HOST}" "cd ${REMOTE_BASE_DIR} && bash -s" << 'ENDSSH'
PORTS_DIR=".ports"
NGINX_DIR=".nginx"
NGINX_SERVER="NGINX_SERVER_PLACEHOLDER"
REMOTE_SITES_ENABLED="/etc/nginx/sites-enabled"

if [ ! -d "$PORTS_DIR" ]; then
    echo "No deployments found (no .ports directory)"
    exit 0
fi

# Check if any port files exist
if [ -z "$(ls -A $PORTS_DIR 2>/dev/null)" ]; then
    echo "No deployments found"
    exit 0
fi

echo "Running Deployments:"
echo "===================="
echo ""

# Get list of enabled protohost configs from remote nginx server
enabled_configs=$(ssh $NGINX_SERVER "ls -1 $REMOTE_SITES_ENABLED/protohost-*.conf 2>/dev/null | xargs -n1 basename | sed 's/^protohost-//' | sed 's/\.conf$//' || true")

for port_file in $PORTS_DIR/*; do
    if [ -f "$port_file" ]; then
        project_name=$(basename "$port_file")

        # Source the port file to get variables
        source "$port_file"

        # Check if nginx config exists and is enabled
        nginx_status="not configured"
        if [ -f "$NGINX_DIR/$project_name.conf" ]; then
            # Check if this project is in the enabled list
            if echo "$enabled_configs" | grep -q "^${project_name}$"; then
                nginx_status="enabled ✓"
            else
                nginx_status="available (not enabled)"
            fi
        fi

        # Calculate time until expiration
        expires_msg="unknown"
        if [ -n "$EXPIRES" ]; then
            current_time=$(date +%s)
            expires_time=$(date -d "$EXPIRES" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$EXPIRES" +%s 2>/dev/null || echo "0")
            if [ "$expires_time" -gt 0 ]; then
                time_diff=$((expires_time - current_time))
                if [ "$time_diff" -gt 0 ]; then
                    days_left=$((time_diff / 86400))
                    expires_msg="in $days_left days"
                else
                    days_overdue=$(( (current_time - expires_time) / 86400 ))
                    expires_msg="EXPIRED $days_overdue days ago"
                fi
            fi
        elif [ -n "$TIMESTAMP" ]; then
            # Fallback to old format
            expires_msg="$TIMESTAMP (old format)"
        fi

        echo "Project: $project_name"
        echo "  Branch:    ${BRANCH:-unknown}"
        echo "  Web:       http://localhost:${WEB_PORT:-unknown}"
        echo "  URL:       https://$project_name.REMOTE_HOST_PLACEHOLDER"
        echo "  Nginx:     $nginx_status"
        echo "  MySQL:     ${MYSQL_PORT:-unknown}"
        echo "  Redis:     ${REDIS_PORT:-unknown}"
        echo "  Created:   ${CREATED:-$TIMESTAMP}"
        echo "  Expires:   $expires_msg"
        echo ""
    fi
done

echo "Commands:"
echo "  cd ~/protohost/<project-name> && make down           # Stop a deployment"
echo "  cd ~/protohost/<project-name> && make logs           # View logs"
echo "  cd ~/protohost/<project-name> && make nginx-enable   # Enable nginx for this deployment"
echo "  cd ~/protohost/<project-name> && make nginx-disable  # Disable nginx for this deployment"
ENDSSH
    # Replace placeholders
    ssh "${REMOTE_USER}@${REMOTE_HOST}" "cd ${REMOTE_BASE_DIR} && bash -s" << ENDSSH2 | sed "s/NGINX_SERVER_PLACEHOLDER/${NGINX_SERVER}/g" | sed "s/REMOTE_HOST_PLACEHOLDER/${REMOTE_HOST}/g"
$(cat << 'INNERSCRIPT'
PORTS_DIR=".ports"
NGINX_DIR=".nginx"
NGINX_SERVER="NGINX_SERVER_PLACEHOLDER"
REMOTE_SITES_ENABLED="/etc/nginx/sites-enabled"
REMOTE_HOST="REMOTE_HOST_PLACEHOLDER"

if [ ! -d "$PORTS_DIR" ]; then
    echo "No deployments found (no .ports directory)"
    exit 0
fi

if [ -z "$(ls -A $PORTS_DIR 2>/dev/null)" ]; then
    echo "No deployments found"
    exit 0
fi

echo "Running Deployments:"
echo "===================="
echo ""

enabled_configs=$(ssh $NGINX_SERVER "ls -1 $REMOTE_SITES_ENABLED/protohost-*.conf 2>/dev/null | xargs -n1 basename | sed 's/^protohost-//' | sed 's/\.conf$//' || true")

for port_file in $PORTS_DIR/*; do
    if [ -f "$port_file" ]; then
        project_name=$(basename "$port_file")
        source "$port_file"

        nginx_status="not configured"
        if [ -f "$NGINX_DIR/$project_name.conf" ]; then
            if echo "$enabled_configs" | grep -q "^${project_name}$"; then
                nginx_status="enabled ✓"
            else
                nginx_status="available (not enabled)"
            fi
        fi

        expires_msg="unknown"
        if [ -n "$EXPIRES" ]; then
            current_time=$(date +%s)
            expires_time=$(date -d "$EXPIRES" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$EXPIRES" +%s 2>/dev/null || echo "0")
            if [ "$expires_time" -gt 0 ]; then
                time_diff=$((expires_time - current_time))
                if [ "$time_diff" -gt 0 ]; then
                    days_left=$((time_diff / 86400))
                    expires_msg="in $days_left days"
                else
                    days_overdue=$(( (current_time - expires_time) / 86400 ))
                    expires_msg="EXPIRED $days_overdue days ago"
                fi
            fi
        elif [ -n "$TIMESTAMP" ]; then
            expires_msg="$TIMESTAMP (old format)"
        fi

        echo "Project: $project_name"
        echo "  Branch:    ${BRANCH:-unknown}"
        echo "  Web:       http://localhost:${WEB_PORT:-unknown}"
        echo "  URL:       https://$project_name.$REMOTE_HOST"
        echo "  Nginx:     $nginx_status"
        echo "  MySQL:     ${MYSQL_PORT:-unknown}"
        echo "  Redis:     ${REDIS_PORT:-unknown}"
        echo "  Created:   ${CREATED:-$TIMESTAMP}"
        echo "  Expires:   $expires_msg"
        echo ""
    fi
done

echo "Commands:"
echo "  cd ~/${REMOTE_BASE_DIR##*/}/<project-name> && make down           # Stop a deployment"
echo "  cd ~/${REMOTE_BASE_DIR##*/}/<project-name> && make logs           # View logs"
echo "  cd ~/${REMOTE_BASE_DIR##*/}/<project-name> && make nginx-enable   # Enable nginx"
echo "  cd ~/${REMOTE_BASE_DIR##*/}/<project-name> && make nginx-disable  # Disable nginx"
INNERSCRIPT
)
ENDSSH2
else
    # We're on the remote server, run directly
    if [ ! -d "$PORTS_DIR" ]; then
        echo "No deployments found (no .ports directory)"
        exit 0
    fi

    if [ -z "$(ls -A $PORTS_DIR 2>/dev/null)" ]; then
        echo "No deployments found"
        exit 0
    fi

    echo "Running Deployments:"
    echo "===================="
    echo ""

    enabled_configs=$(ssh $NGINX_SERVER "ls -1 $REMOTE_SITES_ENABLED/protohost-*.conf 2>/dev/null | xargs -n1 basename | sed 's/^protohost-//' | sed 's/\.conf$//' || true")

    for port_file in $PORTS_DIR/*; do
        if [ -f "$port_file" ]; then
            project_name=$(basename "$port_file")
            source "$port_file"

            nginx_status="not configured"
            if [ -f "$NGINX_DIR/$project_name.conf" ]; then
                if echo "$enabled_configs" | grep -q "^${project_name}$"; then
                    nginx_status="enabled ✓"
                else
                    nginx_status="available (not enabled)"
                fi
            fi

            expires_msg="unknown"
            if [ -n "$EXPIRES" ]; then
                current_time=$(date +%s)
                expires_time=$(date -d "$EXPIRES" +%s 2>/dev/null || date -j -f "%Y-%m-%dT%H:%M:%SZ" "$EXPIRES" +%s 2>/dev/null || echo "0")
                if [ "$expires_time" -gt 0 ]; then
                    time_diff=$((expires_time - current_time))
                    if [ "$time_diff" -gt 0 ]; then
                        days_left=$((time_diff / 86400))
                        expires_msg="in $days_left days"
                    else
                        days_overdue=$(( (current_time - expires_time) / 86400 ))
                        expires_msg="EXPIRED $days_overdue days ago"
                    fi
                fi
            elif [ -n "$TIMESTAMP" ]; then
                expires_msg="$TIMESTAMP (old format)"
            fi

            echo "Project: $project_name"
            echo "  Branch:    ${BRANCH:-unknown}"
            echo "  Web:       http://localhost:${WEB_PORT:-unknown}"
            echo "  URL:       https://$project_name.${REMOTE_HOST}"
            echo "  Nginx:     $nginx_status"
            echo "  MySQL:     ${MYSQL_PORT:-unknown}"
            echo "  Redis:     ${REDIS_PORT:-unknown}"
            echo "  Created:   ${CREATED:-$TIMESTAMP}"
            echo "  Expires:   $expires_msg"
            echo ""
        fi
    done

    echo "Commands:"
    echo "  cd ${REMOTE_BASE_DIR}/<project-name> && make down           # Stop a deployment"
    echo "  cd ${REMOTE_BASE_DIR}/<project-name> && make logs           # View logs"
    echo "  cd ${REMOTE_BASE_DIR}/<project-name> && make nginx-enable   # Enable nginx"
    echo "  cd ${REMOTE_BASE_DIR}/<project-name> && make nginx-disable  # Disable nginx"
fi
