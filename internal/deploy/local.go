package deploy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/docker"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/hooks"
	"github.com/thatjpcsguy/protohost/internal/registry"
)

// LocalOptions contains options for local deployment
type LocalOptions struct {
	Branch string
	Clean  bool
	Build  bool
}

// Local performs a local deployment
func Local(opts LocalOptions) error {
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

	fmt.Printf("🚀 Deploying %s locally...\n", projectName)
	fmt.Println()

	// Execute pre-deploy hook
	hookEnv := map[string]string{
		"PROJECT_NAME": projectName,
		"BRANCH":       branch,
	}
	if err := hooks.Execute(hooks.PreDeploy, cfg.PreDeployScript, hookEnv); err != nil {
		return fmt.Errorf("pre-deploy hook failed: %w", err)
	}

	// Open registry
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer func() { _ = reg.Close() }()

	// Allocate port and determine if this is a new deployment
	port, isNew, err := reg.AllocatePort(projectName, branch, cfg.RepoURL, cfg.TTLDays, cfg.BaseWebPort)
	if err != nil {
		return fmt.Errorf("failed to allocate port: %w", err)
	}

	fmt.Printf("📍 Allocated port: %d\n", port)
	hookEnv["WEB_PORT"] = fmt.Sprintf("%d", port)

	// For local deployment, use current directory if in a git repo
	var deployDir string

	if git.IsGitRepo() {
		// Use current directory
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		deployDir = cwd
		fmt.Println("📂 Using current directory for deployment")
	} else {
		// Not in a git repo, clone to deployment directory
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		deployDir = filepath.Join(home, ".protohost", "deployments", projectName)

		// Clone or pull repository
		_, err = git.CloneOrPull(cfg.RepoURL, branch, deployDir)
		if err != nil {
			return fmt.Errorf("failed to update repository: %w", err)
		}
	}

	// Handle --clean flag
	if opts.Clean {
		fmt.Println("🧹 Cleaning existing deployment...")
		if err := docker.Down(projectName, deployDir, true); err != nil {
			fmt.Printf("Warning: failed to clean deployment: %v\n", err)
		}
	}

	// Build containers if requested or if this is a new deployment
	if opts.Build || isNew {
		if err := docker.Build(projectName, deployDir); err != nil {
			return err
		}
	}

	// Start containers
	env := map[string]string{
		"WEB_PORT":              fmt.Sprintf("%d", port),
		"COMPOSE_PROJECT_NAME":  projectName,
	}

	if err := docker.Up(projectName, deployDir, env); err != nil {
		return err
	}

	// Update registry status
	if err := reg.UpdateStatus(projectName, "running"); err != nil {
		fmt.Printf("Warning: failed to update registry status: %v\n", err)
	}

	// Execute post-start hook
	if err := hooks.Execute(hooks.PostStart, cfg.PostStartScript, hookEnv); err != nil {
		fmt.Printf("Warning: post-start hook failed: %v\n", err)
	}

	// Execute first-install hook if this is a new deployment
	if isNew {
		if err := hooks.Execute(hooks.FirstInstall, cfg.FirstInstallScript, hookEnv); err != nil {
			fmt.Printf("Warning: first-install hook failed: %v\n", err)
		}
	}

	// Display deployment info
	fmt.Println()
	fmt.Println("✅ Deployment complete!")
	fmt.Println()
	fmt.Printf("🌐 URL: http://localhost:%d\n", port)
	fmt.Printf("📋 Project: %s\n", projectName)
	fmt.Printf("📂 Directory: %s\n", deployDir)
	fmt.Println()

	// Execute post-deploy hook
	if err := hooks.Execute(hooks.PostDeploy, cfg.PostDeployScript, hookEnv); err != nil {
		fmt.Printf("Warning: post-deploy hook failed: %v\n", err)
	}

	return nil
}
