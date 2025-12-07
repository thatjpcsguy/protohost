package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/docker"
	"github.com/thatjpcsguy/protohost/internal/git"
	"github.com/thatjpcsguy/protohost/internal/registry"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewDownCmd creates the down command
func NewDownCmd() *cobra.Command {
	var (
		remote        bool
		removeVolumes bool
		branch        string
	)

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Stop deployment",
		Long:  `Stops the deployment for the current branch.`,
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

			if remote {
				return downRemote(cfg, projectName, removeVolumes)
			}

			return downLocal(projectName, removeVolumes)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Stop remote deployment")
	cmd.Flags().BoolVarP(&removeVolumes, "remove-volumes", "v", false, "Remove volumes")
	cmd.Flags().StringVar(&branch, "branch", "", "Branch name (defaults to current)")

	return cmd
}

func downLocal(projectName string, removeVolumes bool) error {
	// Get deployment directory
	home, err := getUserHomeDir()
	if err != nil {
		return err
	}

	deployDir := fmt.Sprintf("%s/.protohost/deployments/%s", home, projectName)

	// Stop containers
	if err := docker.Down(projectName, deployDir, removeVolumes); err != nil {
		return err
	}

	// Update registry
	reg, err := registry.New()
	if err != nil {
		fmt.Printf("Warning: failed to update registry: %v\n", err)
	} else {
		defer reg.Close()
		if err := reg.UpdateStatus(projectName, "stopped"); err != nil {
			fmt.Printf("Warning: failed to update status: %v\n", err)
		}
	}

	fmt.Println("✅ Deployment stopped")
	return nil
}

func downRemote(cfg *config.Config, projectName string, removeVolumes bool) error {
	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer client.Close()

	volumeFlag := ""
	if removeVolumes {
		volumeFlag = "-v"
	}

	cmd := fmt.Sprintf("cd %s/%s && protohost down %s",
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
