package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// HookType represents the type of hook
type HookType string

const (
	PreDeploy    HookType = "pre-deploy"
	PostDeploy   HookType = "post-deploy"
	PostStart    HookType = "post-start"
	FirstInstall HookType = "first-install"
)

// Execute runs a hook if it exists
// Priority: file-based hook > script from config
func Execute(hookType HookType, scriptFromConfig string, env map[string]string) error {
	// Check for file-based hook first
	hookPath := filepath.Join(".protohost", "hooks", string(hookType)+".sh")
	if _, err := os.Stat(hookPath); err == nil {
		fmt.Printf("🪝 Running %s hook (file-based)...\n", hookType)
		return execHookFile(hookPath, env)
	}

	// Fallback to script from config
	if scriptFromConfig != "" {
		fmt.Printf("🪝 Running %s script (from config)...\n", hookType)
		return execHookScript(scriptFromConfig, env)
	}

	// No hook defined
	return nil
}

// execHookFile executes a hook script file
func execHookFile(path string, env map[string]string) error {
	cmd := exec.Command("bash", path)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook failed: %w", err)
	}

	return nil
}

// execHookScript executes a hook script from config
func execHookScript(script string, env map[string]string) error {
	cmd := exec.Command("bash", "-c", script)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Set environment variables
	cmd.Env = os.Environ()
	for k, v := range env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("hook script failed: %w", err)
	}

	return nil
}
