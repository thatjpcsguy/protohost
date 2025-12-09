package ssh

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
	"golang.org/x/term"
)

// Client represents an SSH client
type Client struct {
	Host       string
	User       string
	client     *ssh.Client
	jumpClient *ssh.Client // Optional jump host client
}

// NewClient creates a new SSH client
func NewClient(user, host, configKeyPath string, jumpUser, jumpHost string) (*Client, error) {
	// Get SSH key path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	var keyPath string
	var key []byte

	// If a specific key path is configured, try that first
	if configKeyPath != "" {
		keyPath = configKeyPath
		key, err = os.ReadFile(keyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read configured SSH key at %s: %w", keyPath, err)
		}
	} else {
		// Fall back to default key paths
		keyPath = filepath.Join(home, ".ssh", "id_rsa")
		key, err = os.ReadFile(keyPath)
		if err != nil {
			keyPath = filepath.Join(home, ".ssh", "id_ed25519")
			key, err = os.ReadFile(keyPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read SSH key: %w", err)
			}
		}
	}

	// Parse private key
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		// Check if the error is due to passphrase protection
		if strings.Contains(err.Error(), "passphrase") ||
		   strings.Contains(err.Error(), "encrypted") ||
		   strings.Contains(err.Error(), "cannot decode") {
			// Prompt for passphrase
			fmt.Printf("Enter passphrase for %s: ", keyPath)
			passphrase, passphraseErr := term.ReadPassword(int(syscall.Stdin))
			fmt.Println() // Add newline after password input
			if passphraseErr != nil {
				return nil, fmt.Errorf("failed to read passphrase: %w", passphraseErr)
			}

			// Try parsing with passphrase
			signer, err = ssh.ParsePrivateKeyWithPassphrase(key, passphrase)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key with passphrase: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
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

	var client *ssh.Client
	var jumpClient *ssh.Client

	// If jump host is specified, connect through it
	if jumpHost != "" {
		// Connect to jump host first
		jumpConfig := &ssh.ClientConfig{
			User: jumpUser,
			Auth: []ssh.AuthMethod{
				ssh.PublicKeys(signer),
			},
			HostKeyCallback: hostKeyCallback,
		}

		jumpClient, err = ssh.Dial("tcp", fmt.Sprintf("%s:22", jumpHost), jumpConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to jump host %s@%s: %w", jumpUser, jumpHost, err)
		}

		// Connect to target host through jump host
		conn, err := jumpClient.Dial("tcp", fmt.Sprintf("%s:22", host))
		if err != nil {
			_ = jumpClient.Close()
			return nil, fmt.Errorf("failed to dial %s through jump host: %w", host, err)
		}

		// Create SSH connection over the jump host connection
		ncc, chans, reqs, err := ssh.NewClientConn(conn, fmt.Sprintf("%s:22", host), config)
		if err != nil {
			_ = conn.Close()
			_ = jumpClient.Close()
			return nil, fmt.Errorf("failed to create SSH connection through jump host: %w", err)
		}

		client = ssh.NewClient(ncc, chans, reqs)
	} else {
		// Direct connection (no jump host)
		client, err = ssh.Dial("tcp", fmt.Sprintf("%s:22", host), config)
		if err != nil {
			return nil, fmt.Errorf("failed to connect to %s@%s: %w", user, host, err)
		}
	}

	return &Client{
		Host:       host,
		User:       user,
		client:     client,
		jumpClient: jumpClient,
	}, nil
}

// Close closes the SSH connection
func (c *Client) Close() error {
	var err error
	if c.client != nil {
		err = c.client.Close()
	}
	if c.jumpClient != nil {
		if jumpErr := c.jumpClient.Close(); jumpErr != nil && err == nil {
			err = jumpErr
		}
	}
	return err
}

// Execute runs a command and returns the output
func (c *Client) Execute(command string) (string, error) {
	session, err := c.client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create session: %w", err)
	}
	defer func() { _ = session.Close() }()

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
	defer func() { _ = session.Close() }()

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
	defer func() { _ = session.Close() }()

	// Use cat to write file
	stdin, err := session.StdinPipe()
	if err != nil {
		return fmt.Errorf("failed to create stdin pipe: %w", err)
	}

	go func() {
		defer func() { _ = stdin.Close() }()
		_, _ = stdin.Write(content)
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
