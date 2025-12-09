package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/registry"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewInfoCmd creates the info command
func NewInfoCmd() *cobra.Command {
	var remote bool
	var local bool

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show deployment info",
		Long:  `Shows remote deployment info by default. Use --local to show local deployment info.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			branch, err := git.GetCurrentBranch()
			if err != nil {
				return fmt.Errorf("failed to detect branch: %w", err)
			}

			projectName := fmt.Sprintf("%s-%s", cfg.ProjectPrefix, branch)

			// Default to remote unless --local is specified
			if local {
				return infoLocal(projectName)
			}

			return infoRemote(cfg, projectName)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Show remote deployment info (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "Show local deployment info instead of remote")

	return cmd
}

func infoLocal(projectName string) error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer func() { _ = reg.Close() }()

	alloc, err := reg.GetAllocation(projectName)
	if err != nil {
		return fmt.Errorf("no deployment found for %s", projectName)
	}

	fmt.Printf("Project: %s\n", alloc.ProjectName)
	fmt.Printf("Branch:  %s\n", alloc.Branch)
	fmt.Printf("Status:  %s\n", alloc.Status)
	fmt.Printf("Port:    %d\n", alloc.WebPort)
	fmt.Printf("URL:     http://localhost:%d\n", alloc.WebPort)
	fmt.Printf("Created: %s\n", alloc.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("Expires: %s\n", alloc.ExpiresAt.Format("2006-01-02 15:04:05"))

	return nil
}

func infoRemote(cfg *config.Config, projectName string) error {
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost, cfg.SSHKeyPath, cfg.RemoteJumpUser, cfg.RemoteJumpHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Use --local to avoid recursive remote execution
	cmd := fmt.Sprintf("cd %s/%s && protohost info --local", cfg.RemoteBaseDir, projectName)
	return client.ExecuteInteractive(cmd)
}
