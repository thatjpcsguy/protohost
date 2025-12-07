package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/cmd"
)

var version = "0.1.2"

func main() {
	rootCmd := &cobra.Command{
		Use:   "protohost",
		Short: "Multi-branch Docker Compose deployment tool",
		Long: `Protohost is a deployment tool for managing multiple branches of Docker Compose
applications with automatic port allocation and nginx configuration.`,
		Version: version,
	}

	// Add subcommands
	rootCmd.AddCommand(cmd.NewInitCmd())
	rootCmd.AddCommand(cmd.NewDeployCmd())
	rootCmd.AddCommand(cmd.NewListCmd())
	rootCmd.AddCommand(cmd.NewLogsCmd())
	rootCmd.AddCommand(cmd.NewDownCmd())
	rootCmd.AddCommand(cmd.NewInfoCmd())
	rootCmd.AddCommand(cmd.NewCleanupCmd())
	rootCmd.AddCommand(cmd.NewBootstrapRemoteCmd())

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
