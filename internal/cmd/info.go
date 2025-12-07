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

	cmd := &cobra.Command{
		Use:   "info",
		Short: "Show deployment info",
		Long:  `Shows information about the current branch deployment.`,
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

			if remote {
				return infoRemote(cfg, projectName)
			}

			return infoLocal(projectName)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Show remote deployment info")

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
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	cmd := fmt.Sprintf("cd %s/%s && protohost info", cfg.RemoteBaseDir, projectName)
	return client.ExecuteInteractive(cmd)
}
