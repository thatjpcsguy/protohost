package docker

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Build builds Docker Compose containers
func Build(projectName, dir string) error {
	fmt.Println("🔨 Building Docker containers...")

	cmd := exec.Command("docker", "compose", "-p", projectName, "build")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to build containers: %w", err)
	}

	return nil
}

// Up starts Docker Compose containers
func Up(projectName, dir string, env map[string]string) error {
	fmt.Println("🚀 Starting containers...")

	// Create .env file with environment variables
	if err := writeEnvFile(dir, env); err != nil {
		return err
	}

	cmd := exec.Command("docker", "compose", "-p", projectName, "up", "-d")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start containers: %w", err)
	}

	return nil
}

// Down stops and removes Docker Compose containers
func Down(projectName, dir string, removeVolumes bool) error {
	fmt.Println("🛑 Stopping containers...")

	args := []string{"compose", "-p", projectName, "down"}
	if removeVolumes {
		args = append(args, "-v")
		fmt.Println("   Removing volumes...")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to stop containers: %w", err)
	}

	return nil
}

// Logs streams logs from Docker Compose containers
func Logs(projectName, dir string, follow bool) error {
	args := []string{"compose", "-p", projectName, "logs"}
	if follow {
		args = append(args, "-f")
	}

	cmd := exec.Command("docker", args...)
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// IsRunning checks if containers are running
func IsRunning(projectName string) (bool, error) {
	cmd := exec.Command("docker", "compose", "-p", projectName, "ps", "--quiet")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// If there's output, containers are running
	return len(strings.TrimSpace(string(output))) > 0, nil
}

// Status returns the status of containers
func Status(projectName, dir string) (string, error) {
	cmd := exec.Command("docker", "compose", "-p", projectName, "ps")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w", err)
	}

	return string(output), nil
}

// writeEnvFile writes environment variables to a .env file
func writeEnvFile(dir string, env map[string]string) error {
	envPath := filepath.Join(dir, ".env")

	// Read existing .env if it exists
	existingVars := make(map[string]string)
	if content, err := os.ReadFile(envPath); err == nil {
		for _, line := range strings.Split(string(content), "\n") {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				existingVars[parts[0]] = parts[1]
			}
		}
	}

	// Merge with new env vars (new vars take precedence)
	for k, v := range env {
		existingVars[k] = v
	}

	// Write to .env file
	file, err := os.Create(envPath)
	if err != nil {
		return fmt.Errorf("failed to create .env file: %w", err)
	}
	defer file.Close()

	// Write sorted vars
	for k, v := range existingVars {
		if _, err := io.WriteString(file, fmt.Sprintf("%s=%s\n", k, v)); err != nil {
			return fmt.Errorf("failed to write .env file: %w", err)
		}
	}

	return nil
}
