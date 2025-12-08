package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Config represents the protohost configuration
type Config struct {
	// Project settings
	ProjectPrefix string
	RepoURL       string
	TTLDays       int

	// Remote settings
	RemoteHost     string
	RemoteUser     string
	RemoteBaseDir  string
	NginxProxyHost string
	NginxServer    string

	// Port settings
	BaseWebPort int

	// SSH settings
	SSHKeyPath string

	// SSL settings
	SSLCertPath   string
	SSLKeyPath    string
	SSLParamsFile string

	// Hooks (fallback if hook files don't exist)
	PreDeployScript    string
	PostDeployScript   string
	PostStartScript    string
	FirstInstallScript string
}

// Load reads and parses the .protohost.config file
func Load() (*Config, error) {
	cfg := &Config{
		// Set defaults
		TTLDays:       7,
		BaseWebPort:   3000,
		SSLParamsFile: "ssl-params.conf",
	}

	// Load global config first (lowest priority)
	home, err := os.UserHomeDir()
	if err == nil {
		globalConfigPath := filepath.Join(home, ".protohost", "config")
		if _, err := os.Stat(globalConfigPath); err == nil {
			if err := loadConfigFile(globalConfigPath, cfg); err != nil {
				return nil, fmt.Errorf("failed to load global config: %w", err)
			}
		}
	}

	// Load main config
	if err := loadConfigFile(".protohost.config", cfg); err != nil {
		return nil, fmt.Errorf("failed to load .protohost.config: %w", err)
	}

	// Load local overrides if they exist (highest priority)
	if _, err := os.Stat(".protohost.config.local"); err == nil {
		if err := loadConfigFile(".protohost.config.local", cfg); err != nil {
			return nil, fmt.Errorf("failed to load .protohost.config.local: %w", err)
		}
	}

	// Expand environment variables and tildes
	if err := cfg.expandVariables(); err != nil {
		return nil, err
	}

	// Validate required fields
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

// loadConfigFile parses a bash-style config file
func loadConfigFile(filename string, cfg *Config) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// Regex to match KEY="value" or KEY=value
	re := regexp.MustCompile(`^([A-Z_]+)=(.*)$`)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		key := matches[1]
		value := strings.Trim(matches[2], `"'`)

		// Set config values
		switch key {
		case "PROJECT_PREFIX":
			cfg.ProjectPrefix = value
		case "REPO_URL":
			cfg.RepoURL = value
		case "TTL_DAYS":
			_, _ = fmt.Sscanf(value, "%d", &cfg.TTLDays)
		case "REMOTE_HOST":
			cfg.RemoteHost = value
		case "REMOTE_USER":
			cfg.RemoteUser = value
		case "REMOTE_BASE_DIR":
			cfg.RemoteBaseDir = value
		case "NGINX_PROXY_HOST":
			cfg.NginxProxyHost = value
		case "NGINX_SERVER":
			cfg.NginxServer = value
		case "BASE_WEB_PORT":
			_, _ = fmt.Sscanf(value, "%d", &cfg.BaseWebPort)
		case "SSH_KEY_PATH":
			cfg.SSHKeyPath = value
		case "SSL_CERT_PATH":
			cfg.SSLCertPath = value
		case "SSL_KEY_PATH":
			cfg.SSLKeyPath = value
		case "SSL_PARAMS_FILE":
			cfg.SSLParamsFile = value
		case "PRE_DEPLOY_SCRIPT":
			cfg.PreDeployScript = value
		case "POST_DEPLOY_SCRIPT":
			cfg.PostDeployScript = value
		case "POST_START_SCRIPT":
			cfg.PostStartScript = value
		case "FIRST_INSTALL_SCRIPT":
			cfg.FirstInstallScript = value
		}
	}

	return scanner.Err()
}

// expandVariables expands environment variables and tildes in paths
func (c *Config) expandVariables() error {
	// Expand ${USER} in RemoteUser
	if c.RemoteUser == "${USER}" || c.RemoteUser == "$USER" {
		c.RemoteUser = os.Getenv("USER")
	}

	// Expand ~ in SSHKeyPath (local path)
	if strings.HasPrefix(c.SSHKeyPath, "~") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to expand ~ in SSH_KEY_PATH: %w", err)
		}
		c.SSHKeyPath = strings.Replace(c.SSHKeyPath, "~", homeDir, 1)
	}

	// Don't expand ~ in RemoteBaseDir - let the remote shell handle it
	// This allows ~/protohost to work correctly on remote servers

	// Don't set default SSL paths here - let nginx.go handle defaults
	// This allows nginx to use the correct public domain (protohost.xyz)

	return nil
}

// Validate checks that all required fields are set
func (c *Config) Validate() error {
	required := map[string]string{
		"PROJECT_PREFIX":   c.ProjectPrefix,
		"REPO_URL":         c.RepoURL,
		"REMOTE_HOST":      c.RemoteHost,
		"REMOTE_USER":      c.RemoteUser,
		"REMOTE_BASE_DIR":  c.RemoteBaseDir,
		"NGINX_PROXY_HOST": c.NginxProxyHost,
		"NGINX_SERVER":     c.NginxServer,
	}

	var missing []string
	for field, value := range required {
		if value == "" {
			missing = append(missing, field)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required configuration fields: %s", strings.Join(missing, ", "))
	}

	return nil
}
