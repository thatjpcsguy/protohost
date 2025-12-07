package cmd

import (
	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/deploy"
)

// NewDeployCmd creates the deploy command
func NewDeployCmd() *cobra.Command {
	var (
		remote        bool
		clean         bool
		build         bool
		branch        string
		autoBootstrap bool
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy current branch",
		Long:  `Deploys the current branch locally or to a remote server.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if remote {
				return deploy.Remote(deploy.RemoteOptions{
					Branch:        branch,
					Clean:         clean,
					Build:         build,
					AutoBootstrap: autoBootstrap,
				})
			}

			return deploy.Local(deploy.LocalOptions{
				Branch: branch,
				Clean:  clean,
				Build:  build,
			})
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Deploy to remote server")
	cmd.Flags().BoolVar(&clean, "clean", false, "Remove everything before deploying")
	cmd.Flags().BoolVar(&build, "build", false, "Force rebuild containers")
	cmd.Flags().StringVar(&branch, "branch", "", "Override branch name")
	cmd.Flags().BoolVar(&autoBootstrap, "auto-bootstrap", false, "Automatically install protohost on remote if missing")

	return cmd
}
