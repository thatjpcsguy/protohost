package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/registry"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewListCmd creates the list command
func NewListCmd() *cobra.Command {
	var remote bool
	var local bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all deployments",
		Long:  `Lists remote deployments by default. Use --local to list local deployments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to remote unless --local is specified
			if local {
				return listLocal()
			}
			return listRemote()
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "List remote deployments (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "List local deployments instead of remote")

	return cmd
}

func listLocal() error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer func() { _ = reg.Close() }()

	allocations, err := reg.ListAllocations()
	if err != nil {
		return fmt.Errorf("failed to list allocations: %w", err)
	}

	if len(allocations) == 0 {
		fmt.Println("No local deployments found")
		return nil
	}

	green := color.New(color.FgGreen).SprintFunc()
	yellow := color.New(color.FgYellow).SprintFunc()
	red := color.New(color.FgRed).SprintFunc()

	fmt.Println("Local Deployments")
	fmt.Println("=================")
	fmt.Println()

	for _, alloc := range allocations {
		// Color-code status
		statusStr := alloc.Status
		switch alloc.Status {
		case "running":
			statusStr = green(alloc.Status)
		case "stopped":
			statusStr = yellow(alloc.Status)
		case "expired":
			statusStr = red(alloc.Status)
		}

		fmt.Printf("%s (%s)\n", alloc.ProjectName, statusStr)
		fmt.Printf("  Branch:   %s\n", alloc.Branch)
		fmt.Printf("  Port:     %d\n", alloc.WebPort)
		fmt.Printf("  URL:      http://localhost:%d\n", alloc.WebPort)
		fmt.Printf("  Created:  %s\n", alloc.CreatedAt.Format("2006-01-02 15:04:05"))

		// Show expiration
		if time.Now().After(alloc.ExpiresAt) {
			daysAgo := int(time.Since(alloc.ExpiresAt).Hours() / 24)
			fmt.Printf("  Expires:  %s\n", red(fmt.Sprintf("expired %d days ago", daysAgo)))
		} else {
			daysLeft := int(time.Until(alloc.ExpiresAt).Hours() / 24)
			fmt.Printf("  Expires:  in %d days\n", daysLeft)
		}

		fmt.Println()
	}

	return nil
}

func listRemote() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	fmt.Printf("Connecting to %s@%s...\n", cfg.RemoteUser, cfg.RemoteHost)

	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	// Run protohost list on remote (with --local to avoid recursive remote execution)
	if err := client.ExecuteInteractive("cd " + cfg.RemoteBaseDir + " && protohost list --local"); err != nil {
		return fmt.Errorf("failed to list remote deployments: %w", err)
	}

	return nil
}
