package cmd

import (
	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/deploy"
)

// NewBootstrapRemoteCmd creates the bootstrap-remote command
func NewBootstrapRemoteCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "bootstrap-remote",
		Short: "Install protohost on remote server",
		Long:  `Installs protohost on the remote server specified in .protohost.config`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return deploy.BootstrapRemote()
		},
	}
}
