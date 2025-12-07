package git

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// GetCurrentBranch returns the current git branch name
func GetCurrentBranch() (string, error) {
	cmd := exec.Command("git", "branch", "--show-current")
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	branch := strings.TrimSpace(string(output))
	if branch == "" {
		return "", fmt.Errorf("not on a branch")
	}

	return branch, nil
}

// CloneOrPull clones a repository or pulls updates if it already exists
// Returns true if this is a new clone, false if it was an update
func CloneOrPull(repoURL, branch, targetDir string) (bool, error) {
	// Check if directory exists
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		// Clone repository
		fmt.Printf("📦 Cloning repository (branch: %s)...\n", branch)
		cmd := exec.Command("git", "clone", "-b", branch, repoURL, targetDir)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return false, fmt.Errorf("failed to clone repository: %w", err)
		}
		return true, nil
	}

	// Pull updates
	fmt.Printf("🔄 Updating repository (branch: %s)...\n", branch)

	// Fetch
	cmd := exec.Command("git", "fetch", "origin")
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to fetch: %w", err)
	}

	// Reset to remote branch
	cmd = exec.Command("git", "reset", "--hard", fmt.Sprintf("origin/%s", branch))
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to reset: %w", err)
	}

	// Pull
	cmd = exec.Command("git", "pull", "origin", branch)
	cmd.Dir = targetDir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return false, fmt.Errorf("failed to pull: %w", err)
	}

	return false, nil
}

// IsGitRepo checks if the current directory is a git repository
func IsGitRepo() bool {
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	return cmd.Run() == nil
}
