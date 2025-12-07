package ssh

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// Client represents an SSH client
type Client struct {
	Host   string
	User   string
	client *ssh.Client
}

// NewClient creates a new SSH client
func NewClient(user, host string) (*Client, error) {
	// Get SSH key path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	keyPath := filepath.Join(home, ".ssh", "id_rsa")
	key, err := os.ReadFile(keyPath)
	if err != nil {
		keyPath = filepath.Join(home, ".ssh", "id_ed25519")
		key, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read SSH key: %w", err)
		}
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Load known_hosts
	knownHostsPath := filepath.Join(home, ".ssh", "known_hosts")
	hostKeyCallback, err := knownhosts.New(knownHostsPath)
	if err != nil {
		// If known_hosts doesn't exist, use insecure (not recommended for production)
		hostKeyCallback = ssh.InsecureIgnoreHostKey()
	}

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: hostKeyCallback,
	}

	// Connect
	client, err := ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s@%s: %w", user, host, err)
	}

	return &Client{
		Host:   host,
		User:   user,
		client: client,
	}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// Execute runs a command and returns the output
func (c *Client) Execute(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	var stdout bytes.Buffer
	session.Stdout = &stdout

	if err := session.Run(command); err != nil {
		return "", fmt.Errorf("failed to execute command: %w", err)
	}

	return stdout.String(), nil
}

// ExecuteInteractive runs a command and streams output to terminal
func (c *Client) ExecuteInteractive(command string) error {
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	if err := session.Run(command); err != nil {
		return fmt.Errorf("command failed: %w", err)
	}

	return nil
}

// SCP copies a file to the remote host
func (c *Client) SCP(localPath, remotePath string) error {
	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return fmt.Errorf("failed to read local file: %w", err)
	}

	// Create remote file
	session, err := c.client.NewSession()
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}
	defer session.Close()

	// Use cat to write file
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer stdin.Close()
		io.WriteString(stdin, string(content))
	}()

	if err := session.Run(fmt.Sprintf("cat > %s", remotePath)); err != nil {
		return fmt.Errorf("failed to write remote file: %w", err)
	}

	return nil
}

// CheckProtohostInstalled checks if protohost is installed on remote
func (c *Client) CheckProtohostInstalled() (bool, error) {
	output, err := c.Execute("which protohost")
	if err != nil {
		return false, nil
	}

	return strings.TrimSpace(output) != "", nil
}

// SimpleExecute is a simpler way to execute commands via SSH using the ssh binary
// This is useful when we don't need the full SSH client
func SimpleExecute(user, host, command string) error {
	cmd := exec.Command("ssh", fmt.Sprintf("%s@%s", user, host), command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
