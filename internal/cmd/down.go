package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/docker"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/nginx"
	"github.com/thatjpcsguy/protohost/internal/registry"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewDownCmd creates the down command
func NewDownCmd() *cobra.Command {
	var (
		remote        bool
		local         bool
		removeVolumes bool
		branch        string
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop deployment",
		Long:  `Stops the remote deployment by default. Use --local to stop local deployment.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			// Detect branch if not specified
			if branch == "" {
				branch, err = git.GetCurrentBranch()
				if err != nil {
					return fmt.Errorf("failed to detect branch: %w", err)
				}
			}

			projectName := fmt.Sprintf("%s-%s", cfg.ProjectPrefix, branch)

			// Default to remote unless --local is specified
			if local {
				return downLocal(projectName, removeVolumes)
			}

			return downRemote(cfg, projectName, removeVolumes)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Stop remote deployment (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "Stop local deployment instead of remote")
	cmd.Flags().BoolVarP(&removeVolumes, "remove-volumes", "v", false, "Remove volumes")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name (defaults to current)")

	return cmd
}

func downLocal(projectName string, removeVolumes bool) error {
	// Determine deployment directory (same logic as deploy)
	var deployDir string
	if git.IsGitRepo() {
		// Use current directory if in a git repo
		cwd, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get current directory: %w", err)
		}
		deployDir = cwd
	} else {
		// Otherwise use deployments directory
		home, err := getUserHomeDir()
		if err != nil {
			return err
		}
		deployDir = fmt.Sprintf("%s/.protohost/deployments/%s", home, projectName)
	}

	// Load config for nginx removal
	cfg, err := config.Load()
	if err != nil {
		fmt.Printf("Warning: failed to load config: %v\n", err)
	}

	// Remove nginx configuration
	if cfg != nil && cfg.NginxServer != "" {
		fmt.Println("🌐 Removing nginx configuration...")
		if err := nginx.Remove(cfg, projectName); err != nil {
			fmt.Printf("Warning: failed to remove nginx config: %v\n", err)
		}
	}

	// Stop containers
	if err := docker.Down(projectName, deployDir, removeVolumes); err != nil {
		return err
	}

	// Update registry
	reg, err := registry.New()
	if err != nil {
		fmt.Printf("Warning: failed to update registry: %v\n", err)
	} else {
		defer func() { _ = reg.Close() }()
		if removeVolumes {
			// If volumes are removed, delete the registry entry so next deploy runs first-install
			if err := reg.ReleasePort(projectName); err != nil {
				fmt.Printf("Warning: failed to release port: %v\n", err)
			}
		} else {
			// Otherwise just mark as stopped
			if err := reg.UpdateStatus(projectName, "stopped"); err != nil {
				fmt.Printf("Warning: failed to update status: %v\n", err)
			}
		}
	}

	fmt.Println("✅ Deployment stopped")
	return nil
}

func downRemote(cfg *config.Config, projectName string, removeVolumes bool) error {
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost, cfg.SSHKeyPath)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	volumeFlag := ""
	if removeVolumes {
		volumeFlag = "-v"
	}

	// Use --local to avoid recursive remote execution
	cmd := fmt.Sprintf("cd %s/%s && protohost down --local %s",
		cfg.RemoteBaseDir, projectName, volumeFlag)

	return client.ExecuteInteractive(cmd)
}

func getUserHomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return home, nil
}
