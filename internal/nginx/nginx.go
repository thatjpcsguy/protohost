package nginx

import (
	"fmt"
	"strings"

	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// GenerateConfig generates an nginx configuration for a deployment
func GenerateConfig(cfg *config.Config, projectName string, port int) string {
	serverName := fmt.Sprintf("%s.%s", projectName, cfg.RemoteHost)
	proxyPass := fmt.Sprintf("http://%s:%d", cfg.NginxProxyHost, port)

	sslCert := ""
	sslKey := ""

	if cfg.SSLCertPath != "" && cfg.SSLKeyPath != "" {
		sslCert = cfg.SSLCertPath
		sslKey = cfg.SSLKeyPath
	} else {
		// Default Let's Encrypt paths
		sslCert = fmt.Sprintf("/etc/letsencrypt/live/%s/fullchain.pem", cfg.RemoteHost)
		sslKey = fmt.Sprintf("/etc/letsencrypt/live/%s/privkey.pem", cfg.RemoteHost)
	}

	config := fmt.Sprintf(`server {
    listen 443 ssl;
    server_name %s;

    ssl_certificate %s;
    ssl_certificate_key %s;
    include ssl-params.conf;

    location / {
        proxy_pass %s;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_read_timeout 86400;
        proxy_buffering off;
    }
}
`, serverName, sslCert, sslKey, proxyPass)

	return config
}

// Deploy deploys nginx configuration to the remote nginx server
func Deploy(cfg *config.Config, projectName string, configContent string) error {
	if cfg.NginxServer == "" {
		return fmt.Errorf("NGINX_SERVER not configured")
	}

	client, err := ssh.NewClient(cfg.RemoteUser, cfg.NginxServer)
	if err != nil {
		return fmt.Errorf("failed to connect to nginx server: %w", err)
	}
	defer func() { _ = client.Close() }()

	configFilename := fmt.Sprintf("protohost-%s.conf", projectName)
	tmpPath := fmt.Sprintf("/tmp/%s", configFilename)
	finalPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s", configFilename)

	// Write config to temp file
	writeCmd := fmt.Sprintf("cat > %s << 'NGINX_CONFIG_EOF'\n%s\nNGINX_CONFIG_EOF", tmpPath, configContent)
	if _, err := client.Execute(writeCmd); err != nil {
		return fmt.Errorf("failed to write config to temp file: %w", err)
	}

	// Move to sites-enabled and restart nginx
	deployCmd := fmt.Sprintf("sudo mv %s %s && sudo nginx -t && sudo service nginx restart", tmpPath, finalPath)
	if _, err := client.Execute(deployCmd); err != nil {
		return fmt.Errorf("failed to deploy nginx config: %w", err)
	}

	return nil
}

// Remove removes nginx configuration from the remote nginx server
func Remove(cfg *config.Config, projectName string) error {
	if cfg.NginxServer == "" {
		// No nginx server configured, skip silently
		return nil
	}

	client, err := ssh.NewClient(cfg.RemoteUser, cfg.NginxServer)
	if err != nil {
		return fmt.Errorf("failed to connect to nginx server: %w", err)
	}
	defer func() { _ = client.Close() }()

	configFilename := fmt.Sprintf("protohost-%s.conf", projectName)
	finalPath := fmt.Sprintf("/etc/nginx/sites-enabled/%s", configFilename)

	// Remove config and restart nginx
	removeCmd := fmt.Sprintf("sudo rm -f %s && sudo service nginx restart", finalPath)
	if _, err := client.Execute(removeCmd); err != nil {
		// Don't fail if file doesn't exist
		if !strings.Contains(err.Error(), "No such file") {
			return fmt.Errorf("failed to remove nginx config: %w", err)
		}
	}

	return nil
}
