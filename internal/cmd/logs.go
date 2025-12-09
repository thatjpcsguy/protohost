package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/docker"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewLogsCmd creates the logs command
func NewLogsCmd() *cobra.Command {
	var (
		remote bool
		local  bool
		follow bool
		branch string
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View logs for deployment",
		Long:  `Views remote logs by default. Use --local to view local logs.`,
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
				return logsLocal(projectName, follow)
			}

			return logsRemote(cfg, projectName, follow)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "View remote logs (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "View local logs instead of remote")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Follow log output")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name (defaults to current)")

	return cmd
}

func logsLocal(projectName string, follow bool) error {
	// Get deployment directory
	home, err := getUserHomeDir()
	if err != nil {
		return err
	}

	deployDir := fmt.Sprintf("%s/.protohost/deployments/%s", home, projectName)

	return docker.Logs(projectName, deployDir, follow)
}

func logsRemote(cfg *config.Config, projectName string, follow bool) error {
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost, cfg.SSHKeyPath, cfg.RemoteJumpUser, cfg.RemoteJumpHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	followFlag := ""
	if follow {
		followFlag = "-f"
	}

	cmd := fmt.Sprintf("cd %s/%s && docker compose -p %s logs %s",
		cfg.RemoteBaseDir, projectName, projectName, followFlag)

	return client.ExecuteInteractive(cmd)
}
