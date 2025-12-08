package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/thatjpcsguy/protohost/internal/config"
	"github.com/thatjpcsguy/protohost/internal/docker"
	"github.com/thatjpcsguy/protohost/internal/registry"
	"github.com/thatjpcsguy/protohost/internal/ssh"
)

// NewCleanupCmd creates the cleanup command
func NewCleanupCmd() *cobra.Command {
	var (
		remote bool
		local  bool
		dryRun bool
	)

	cmd := &cobra.Command{
		Use:   "cleanup",
		Short: "Remove expired deployments",
		Long:  `Removes remote expired deployments by default. Use --local to cleanup local deployments.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default to remote unless --local is specified
			if local {
				return cleanupLocal(dryRun)
			}
			return cleanupRemote(dryRun)
		},
	}

	cmd.Flags().BoolVar(&remote, "remote", false, "Cleanup remote deployments (default, kept for backwards compatibility)")
	cmd.Flags().BoolVar(&local, "local", false, "Cleanup local deployments instead of remote")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be removed")

	return cmd
}

func cleanupLocal(dryRun bool) error {
	reg, err := registry.New()
	if err != nil {
		return fmt.Errorf("failed to open registry: %w", err)
	}
	defer func() { _ = reg.Close() }()

	// Mark expired deployments
	expired, err := reg.MarkExpired()
	if err != nil {
		return fmt.Errorf("failed to mark expired: %w", err)
	}

	if len(expired) == 0 {
		fmt.Println("No expired deployments found")
		return nil
	}

	red := color.New(color.FgRed).SprintFunc()

	fmt.Println("Found expired deployments:")
	for _, alloc := range expired {
		daysAgo := int(time.Since(alloc.ExpiresAt).Hours() / 24)
		fmt.Printf("  - %s %s\n", alloc.ProjectName, red(fmt.Sprintf("(expired %d days ago)", daysAgo)))
	}
	fmt.Println()

	if dryRun {
		fmt.Println("Dry run - no changes made")
		return nil
	}

	// Clean up each expired deployment
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	for _, alloc := range expired {
		fmt.Printf("Removing %s...\n", alloc.ProjectName)

		deployDir := filepath.Join(home, ".protohost", "deployments", alloc.ProjectName)

		// Stop containers
		if err := docker.Down(alloc.ProjectName, deployDir, true); err != nil {
			fmt.Printf("  Warning: failed to stop containers: %v\n", err)
		} else {
			fmt.Println("  ✓ Stopped containers")
		}

		// Remove directory
		if err := os.RemoveAll(deployDir); err != nil {
			fmt.Printf("  Warning: failed to remove directory: %v\n", err)
		} else {
			fmt.Println("  ✓ Removed directory")
		}

		// Release port
		if err := reg.ReleasePort(alloc.ProjectName); err != nil {
			fmt.Printf("  Warning: failed to release port: %v\n", err)
		} else {
			fmt.Printf("  ✓ Released port %d\n", alloc.WebPort)
		}

		fmt.Println()
	}

	fmt.Printf("✅ Cleanup complete! Removed %d deployment(s)\n", len(expired))
	return nil
}

func cleanupRemote(dryRun bool) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	client, err := ssh.NewClient(cfg.RemoteUser, cfg.RemoteHost)
	if err != nil {
		return fmt.Errorf("failed to connect: %w", err)
	}
	defer func() { _ = client.Close() }()

	dryRunFlag := ""
	if dryRun {
		dryRunFlag = "--dry-run"
	}

	// Use --local to avoid recursive remote execution
	cmd := fmt.Sprintf("cd %s && protohost cleanup --local %s", cfg.RemoteBaseDir, dryRunFlag)
	return client.ExecuteInteractive(cmd)
}
