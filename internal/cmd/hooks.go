package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/hooks"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewHooksCmd creates the hooks command
func NewHooksCmd() *cobra.Command {
	var remote bool
	var local bool
	var branch string

	cmd := &cobra.Command{
		Use:   "hooks [hook-name]",
		Short: "Manually run deployment hooks",
		Long: `Manually execute deployment hooks.

By default, hooks run on the remote server (where deployments are running).
Use --local to run hooks on your local machine instead.

Available hooks:
  pre-deploy     - Runs before deployment starts
  post-deploy    - Runs after deployment completes
  post-start     - Runs after containers start
  first-install  - Runs only on first deployment

Examples:
  protohost hooks post-start                    # Runs on remote (default)
  protohost hooks post-start --local            # Runs locally
  protohost hooks first-install --branch feature-x`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			hookName := args[0]

			// Validate hook name
			var hookType hooks.HookType
			switch hookName {
			case "pre-deploy":
				hookType = hooks.PreDeploy
			case "post-deploy":
				hookType = hooks.PostDeploy
			case "post-start":
				hookType = hooks.PostStart
			case "first-install":
				hookType = hooks.FirstInstall
			default:
				return fmt.Errorf("invalid hook name: %s. Valid options: pre-deploy, post-deploy, post-start, first-install", hookName)
			}

			// Default to remote unless --local is specified
			runRemote := !local

			return runHooks(hookType, runRemote, branch)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Run hook on remote server (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "Run hook locally instead of on remote server")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name (defaults to current branch)")

	return cmd
}

func runHooks(hookType hooks.HookType, remote bool, branchOverride string) error {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Detect branch if not specified
	branch := branchOverride
	if branch == "" {
		branch, err = git.GetCurrentBranch()
		if err != nil {
			return fmt.Errorf("failed to detect branch: %w", err)
		}
	}

	// Generate project name
	projectName := fmt.Sprintf("%s-%s", cfg.ProjectPrefix, branch)

	// Build environment variables for hook
	hookEnv := map[string]string{
		"PROJECT_NAME": projectName,
		"BRANCH":       branch,
		"REMOTE_HOST":  cfg.RemoteHost,
	}

	if remote {
		return runHookRemote(cfg, hookType, projectName, hookEnv)
	}

	return runHookLocal(cfg, hookType, hookEnv)
}

func runHookLocal(cfg *config.Config, hookType hooks.HookType, env map[string]string) error {
	fmt.Printf("🪝 Running %s hook locally...\n", hookType)

	// Get script from config based on hook type
	var scriptFromConfig string
	switch hookType {
	case hooks.PreDeploy:
		scriptFromConfig = cfg.PreDeployScript
	case hooks.PostDeploy:
		scriptFromConfig = cfg.PostDeployScript
	case hooks.PostStart:
		scriptFromConfig = cfg.PostStartScript
	case hooks.FirstInstall:
		scriptFromConfig = cfg.FirstInstallScript
	}

	if err := hooks.Execute(hookType, scriptFromConfig, env); err != nil {
		return fmt.Errorf("hook execution failed: %w", err)
	}

	fmt.Println("✅ Hook completed successfully!")
	return nil
}

func runHookRemote(cfg *config.Config, hookType hooks.HookType, projectName string, env map[string]string) error {
	fmt.Printf("🪝 Running %s hook on remote server %s...\n", hookType, cfg.RemoteHost)

	// Connect to remote
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost, cfg.SSHKeyPath, cfg.RemoteJumpUser, cfg.RemoteJumpHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Build remote command to run the hook
	// The hook will be executed in the context of the deployment directory
	// IMPORTANT: Use --local flag so the remote server runs the hook locally, not recursively remote
	script := fmt.Sprintf(`
set -e
cd %s/%s

# Check if protohost is installed
if ! command -v protohost &> /dev/null; then
    echo "Error: protohost not found on remote server"
    exit 1
fi

# Run the hook locally on the remote server (not recursively remote)
protohost hooks %s --local
`, cfg.RemoteBaseDir, projectName, hookType)

	if err := client.ExecuteInteractive(script); err != nil {
		return fmt.Errorf("remote hook execution failed: %w", err)
	}

	fmt.Println("✅ Remote hook completed successfully!")
	return nil
}
