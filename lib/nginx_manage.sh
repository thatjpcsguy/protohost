#!/bin/bash

# Protohost Deploy - Nginx Management Script
# Manage nginx configurations for deployments

# Load configuration
if [ ! -f ".protohost.config" ]; then
    echo "❌ Error: .protohost.config not found"
    echo "   Run from project directory or install protohost-deploy"
    exit 1
fi

source .protohost.config

# Load local overrides if they exist
if [ -f ".protohost.config.local" ]; then
    source .protohost.config.local
fi

NGINX_DIR="${REMOTE_BASE_DIR}/.nginx"
REMOTE_SITES_ENABLED="/etc/nginx/sites-enabled"

show_usage() {
    echo "Usage: $0 <command> [project-name]"
    echo ""
    echo "Commands:"
    echo "  list              - List all available nginx configs"
    echo "  enable <project>  - Enable nginx config for project"
    echo "  disable <project> - Disable nginx config for project"
    echo "  test              - Test nginx configuration"
    echo "  reload            - Reload nginx"
    echo ""
    echo "Example:"
    echo "  $0 enable ${PROJECT_PREFIX}-main"
    echo "  $0 disable ${PROJECT_PREFIX}-feature-xyz"
}

list_configs() {
    if [ ! -d "$NGINX_DIR" ] || [ -z "$(ls -A $NGINX_DIR 2>/dev/null)" ]; then
        echo "No nginx configs found"
        return
    fi

    echo "Available Nginx Configs:"
    echo "========================"
    echo ""

    # Get list of enabled protohost configs from remote server
    enabled_configs=$(ssh $NGINX_SERVER "ls -1 $REMOTE_SITES_ENABLED/protohost-*.conf 2>/dev/null | xargs -n1 basename | sed 's/^protohost-//' | sed 's/\.conf$//' || true")

    for config in $NGINX_DIR/*.conf; do
        if [ -f "$config" ]; then
            project=$(basename "$config" .conf)
            enabled="disabled"

            # Check if this project is in the enabled list
            if echo "$enabled_configs" | grep -q "^${project}$"; then
                enabled="ENABLED ✓"
            fi

            echo "  $project - $enabled"
        fi
    done
}

enable_config() {
    local project=$1
    local config_file="$NGINX_DIR/$project.conf"

    if [ ! -f "$config_file" ]; then
        echo "❌ Config not found: $config_file"
        echo "   Run 'make nginx-config' from the project directory first"
        exit 1
    fi

    echo "Enabling nginx config for $project..."

    # Copy config to remote server with protohost- prefix
    echo "  Uploading config to nginx server..."
    scp "$config_file" "$NGINX_SERVER:/tmp/protohost-$project.conf"

    # Move to sites-enabled and restart nginx
    echo "  Installing config and restarting nginx..."
    ssh $NGINX_SERVER "sudo mv /tmp/protohost-$project.conf $REMOTE_SITES_ENABLED/protohost-$project.conf && sudo service nginx restart"

    if [ $? -eq 0 ]; then
        echo "✓ Nginx config deployed and nginx restarted"
        echo ""
        echo "🚀 Deployment is now live at: https://$project.${REMOTE_HOST}"
    else
        echo "❌ Failed to deploy nginx config"
        exit 1
    fi
}

disable_config() {
    local project=$1

    echo "Disabling nginx config for $project..."

    # Remove config from remote server and restart nginx (with protohost- prefix)
    ssh $NGINX_SERVER "sudo rm -f $REMOTE_SITES_ENABLED/protohost-$project.conf && sudo service nginx restart"

    if [ $? -eq 0 ]; then
        echo "✓ Nginx config removed and nginx restarted"
        echo "Deployment disabled: https://$project.${REMOTE_HOST}"
    else
        echo "❌ Failed to disable nginx config"
        exit 1
    fi
}

test_nginx() {
    echo "Testing nginx configuration on remote server..."
    ssh $NGINX_SERVER "sudo nginx -t"
}

reload_nginx() {
    echo "Reloading nginx on remote server..."
    ssh $NGINX_SERVER "sudo service nginx restart"
    echo "✓ Nginx restarted"
}

# Main script
case "${1:-}" in
    list)
        list_configs
        ;;
    enable)
        if [ -z "${2:-}" ]; then
            echo "❌ Error: Project name required"
            show_usage
            exit 1
        fi
        enable_config "$2"
        ;;
    disable)
        if [ -z "${2:-}" ]; then
            echo "❌ Error: Project name required"
            show_usage
            exit 1
        fi
        disable_config "$2"
        ;;
    test)
        test_nginx
        ;;
    reload)
        reload_nginx
        ;;
    *)
        show_usage
        exit 1
        ;;
esac
