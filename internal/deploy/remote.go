package deploy

import (
	"fmt"
	"os"
	"strings"

	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/hooks"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// RemoteOptions contains options for remote deployment
type RemoteOptions struct {
	Branch       string
	Clean        bool
	Build        bool
	AutoBootstrap bool
}

// Remote performs a remote deployment
func Remote(opts RemoteOptions) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Detect branch if not specified
	branch := opts.Branch
	if branch == "" {
		branch, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to detect branch: %w", err)
		}
	}

	// Generate project name
	projectName := fmt.Sprintf("%s-%s", cfg.ProjectPrefix, branch)

	fmt.Printf("🚀 Deploying %s to %s@%s...\n", projectName, cfg.RemoteUser, cfg.RemoteHost)
	fmt.Println()

	// Execute pre-deploy hook locally
	hookEnv := map[string]string{
		"PROJECT_NAME": projectName,
		"BRANCH":       branch,
		"REMOTE_HOST":  cfg.RemoteHost,
	}
	if err := hooks.Execute(hooks.PreDeploy, cfg.PreDeployScript, hookEnv); err != nil {
		return fmt.Errorf("pre-deploy hook failed: %w", err)
	}

	// Connect to remote
	fmt.Printf("🔌 Connecting to %s@%s...\n", cfg.RemoteUser, cfg.RemoteHost)
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Check if protohost is installed on remote
	installed, err := client.CheckProtohostInstalled()
	if err != nil {
		return fmt.Errorf("failed to check protohost installation: %w", err)
	}

	if !installed {
		if opts.AutoBootstrap {
			fmt.Println("⚠️  Protohost not found on remote, installing...")
			if err := bootstrapRemote(client); err != nil {
				return fmt.Errorf("failed to bootstrap remote: %w", err)
			}
		} else {
			msg := fmt.Sprintf(`protohost not found on remote server

Please install protohost on the remote server first:

  Option 1: Run bootstrap command
    protohost bootstrap-remote

  Option 2: Add --auto-bootstrap flag
    protohost deploy --remote --auto-bootstrap

  Option 3: Install manually on remote
    ssh %s@%s "curl -sSL https://raw.githubusercontent.com/thatjpcsguy/protohost/main/install.sh | bash"`, cfg.RemoteUser, cfg.RemoteHost)
			return fmt.Errorf("%s", msg)
		}
	}

	// Build remote deployment script
	script := buildRemoteDeployScript(cfg, projectName, branch, opts)

	// Execute deployment on remote
	fmt.Println("🚀 Executing remote deployment...")
	fmt.Println()

	if err := client.ExecuteInteractive(script); err != nil {
		return fmt.Errorf("remote deployment failed: %w", err)
	}

	fmt.Println()
	fmt.Println("✅ Remote deployment complete!")
	fmt.Printf("🌐 URL: https://%s.protohost.xyz\n", projectName)
	fmt.Println()

	// Execute post-deploy hook locally
	if err := hooks.Execute(hooks.PostDeploy, cfg.PostDeployScript, hookEnv); err != nil {
		fmt.Printf("Warning: post-deploy hook failed: %v\n", err)
	}

	return nil
}

// buildRemoteDeployScript builds the bash script to run on remote
func buildRemoteDeployScript(cfg *config.Config, projectName, branch string, opts RemoteOptions) string {
	var script strings.Builder

	script.WriteString("set -e\n\n")

	// Ensure base directory exists
	script.WriteString(fmt.Sprintf("mkdir -p %s\n", cfg.RemoteBaseDir))
	script.WriteString(fmt.Sprintf("cd %s\n\n", cfg.RemoteBaseDir))

	// Clone or pull repository
	script.WriteString(fmt.Sprintf("if [ ! -d %s ]; then\n", projectName))
	script.WriteString(fmt.Sprintf("    echo '📦 Cloning repository (branch: %s)...'\n", branch))
	script.WriteString(fmt.Sprintf("    git clone -b %s %s %s\n", branch, cfg.RepoURL, projectName))
	script.WriteString("else\n")
	script.WriteString(fmt.Sprintf("    echo '🔄 Updating repository (branch: %s)...'\n", branch))
	script.WriteString(fmt.Sprintf("    cd %s\n", projectName))
	script.WriteString("    git fetch origin\n")
	script.WriteString(fmt.Sprintf("    git reset --hard origin/%s\n", branch))
	script.WriteString(fmt.Sprintf("    git pull origin %s\n", branch))
	script.WriteString(fmt.Sprintf("    cd %s\n", cfg.RemoteBaseDir))
	script.WriteString("fi\n\n")

	// Change to project directory
	script.WriteString(fmt.Sprintf("cd %s/%s\n\n", cfg.RemoteBaseDir, projectName))

	// Build protohost deploy command (use --local to avoid recursive remote execution)
	deployCmd := "protohost deploy --local"
	if opts.Clean {
		deployCmd += " --clean"
	}
	if opts.Build {
		deployCmd += " --build"
	}

	script.WriteString("# Run protohost deploy locally on remote server\n")
	script.WriteString(fmt.Sprintf("%s\n", deployCmd))

	return script.String()
}

// bootstrapRemote installs protohost on remote server
func bootstrapRemote(client *ssh.Client) error {
	script := `
set -e

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
esac

# Download and install protohost
echo "Installing protohost for $OS/$ARCH..."
INSTALL_DIR="$HOME/.local/bin"
mkdir -p "$INSTALL_DIR"

# For now, tell user to install manually
# TODO: Download from GitHub releases when available
echo "Please build and install protohost manually on this server"
exit 1
`

	return client.ExecuteInteractive(script)
}

// BootstrapRemote installs protohost on remote server (command implementation)
func BootstrapRemote() error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		// If no config, try to read from command line
		if !fileExists(".protohost.config") {
			return fmt.Errorf("no .protohost.config found. Run 'protohost init' first or specify --host and --user")
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("🚀 Installing protohost on %s@%s...\n", cfg.RemoteUser, cfg.RemoteHost)

	// Connect to remote
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Check if already installed
	installed, err := client.CheckProtohostInstalled()
	if err != nil {
		return fmt.Errorf("failed to check installation: %w", err)
	}

	if installed {
		fmt.Println("✓ Protohost is already installed on remote")
		return nil
	}

	// Install
	if err := bootstrapRemote(client); err != nil {
		return fmt.Errorf("failed to install: %w", err)
	}

	fmt.Println("✅ Protohost installed successfully!")
	return nil
}

// fileExists checks if a file exists
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
