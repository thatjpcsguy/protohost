package registry

import (
	"database/sql"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Registry manages port allocations
type Registry struct {
	db *sql.DB
}

// New creates or opens a registry database
func New() (*Registry, error) {
	// Get registry path
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	protohostDir := filepath.Join(home, ".protohost")
	if err := os.MkdirAll(protohostDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create .protohost directory: %w", err)
	}

	dbPath := filepath.Join(protohostDir, "registry.db")

	// Open database
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	r := &Registry{db: db}

	// Initialize schema
	if err := r.initSchema(); err != nil {
		_ = db.Close()
		return nil, err
	}

	return r, nil
}

// Close closes the database connection
func (r *Registry) Close() error {
	return r.db.Close()
}

// initSchema creates the port_allocations table if it doesn't exist
func (r *Registry) initSchema() error {
	schema := `
	CREATE TABLE IF NOT EXISTS port_allocations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		project_name TEXT NOT NULL UNIQUE,
		web_port INTEGER NOT NULL UNIQUE,
		branch TEXT NOT NULL,
		created_at TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		status TEXT NOT NULL,
		repo_url TEXT
	);

	CREATE INDEX IF NOT EXISTS idx_status ON port_allocations(status);
	CREATE INDEX IF NOT EXISTS idx_expires ON port_allocations(expires_at);
	`

	_, err := r.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to initialize schema: %w", err)
	}

	return nil
}

// AllocatePort allocates a port for a project, or returns existing allocation
func (r *Registry) AllocatePort(projectName, branch, repoURL string, ttlDays, basePort int) (int, error) {
	// Check if project already has a port
	var existingPort int
	err := r.db.QueryRow(
		"SELECT web_port FROM port_allocations WHERE project_name = ?",
		projectName,
	).Scan(&existingPort)

	if err == nil {
		// Port already allocated, update expiration and status
		expiresAt := time.Now().UTC().AddDate(0, 0, ttlDays).Format(time.RFC3339)
		_, err = r.db.Exec(
			"UPDATE port_allocations SET expires_at = ?, status = 'running' WHERE project_name = ?",
			expiresAt, projectName,
		)
		if err != nil {
			return 0, fmt.Errorf("failed to update expiration: %w", err)
		}
		return existingPort, nil
	}

	if err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to check existing port: %w", err)
	}

	// Find next available port
	port, err := r.findAvailablePort(basePort)
	if err != nil {
		return 0, err
	}

	// Insert new allocation
	createdAt := time.Now().UTC().Format(time.RFC3339)
	expiresAt := time.Now().UTC().AddDate(0, 0, ttlDays).Format(time.RFC3339)

	_, err = r.db.Exec(`
		INSERT INTO port_allocations (project_name, web_port, branch, created_at, expires_at, status, repo_url)
		VALUES (?, ?, ?, ?, ?, 'running', ?)
	`, projectName, port, branch, createdAt, expiresAt, repoURL)

	if err != nil {
		return 0, fmt.Errorf("failed to insert allocation: %w", err)
	}

	return port, nil
}

// findAvailablePort finds the first available port starting from basePort
func (r *Registry) findAvailablePort(basePort int) (int, error) {
	// Get all allocated ports from registry
	rows, err := r.db.Query("SELECT web_port FROM port_allocations")
	if err != nil {
		return 0, fmt.Errorf("failed to query ports: %w", err)
	}
	defer func() { _ = rows.Close() }()

	usedPorts := make(map[int]bool)
	for rows.Next() {
		var port int
		if err := rows.Scan(&port); err != nil {
			return 0, err
		}
		usedPorts[port] = true
	}

	// Find first available port
	for offset := 0; offset < 100; offset++ {
		port := basePort + offset
		if usedPorts[port] {
			continue
		}

		// Check if port is actually available by attempting to bind
		if r.isPortAvailable(port) {
			return port, nil
		}
	}

	return 0, fmt.Errorf("no available ports in range %d-%d", basePort, basePort+99)
}

// isPortAvailable checks if a port is available by attempting to listen on it
func (r *Registry) isPortAvailable(port int) bool {
	addr := fmt.Sprintf(":%d", port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}

// ReleasePort removes a port allocation
func (r *Registry) ReleasePort(projectName string) error {
	_, err := r.db.Exec("DELETE FROM port_allocations WHERE project_name = ?", projectName)
	if err != nil {
		return fmt.Errorf("failed to release port: %w", err)
	}
	return nil
}

// UpdateStatus updates the status of a deployment
func (r *Registry) UpdateStatus(projectName, status string) error {
	_, err := r.db.Exec(
		"UPDATE port_allocations SET status = ? WHERE project_name = ?",
		status, projectName,
	)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}
	return nil
}

// ListAllocations returns all port allocations
func (r *Registry) ListAllocations() ([]PortAllocation, error) {
	rows, err := r.db.Query(`
		SELECT id, project_name, web_port, branch, created_at, expires_at, status, COALESCE(repo_url, '')
		FROM port_allocations
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to query allocations: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var allocations []PortAllocation
	for rows.Next() {
		var a PortAllocation
		var createdAt, expiresAt string

		err := rows.Scan(
			&a.ID, &a.ProjectName, &a.WebPort, &a.Branch,
			&createdAt, &expiresAt, &a.Status, &a.RepoURL,
		)
		if err != nil {
			return nil, err
		}

		// Parse timestamps
		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		a.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

		allocations = append(allocations, a)
	}

	return allocations, nil
}

// MarkExpired marks deployments as expired if they're past their TTL
func (r *Registry) MarkExpired() ([]PortAllocation, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Get expired deployments
	rows, err := r.db.Query(`
		SELECT id, project_name, web_port, branch, created_at, expires_at, status, COALESCE(repo_url, '')
		FROM port_allocations
		WHERE expires_at < ? AND status != 'expired'
	`, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var expired []PortAllocation
	for rows.Next() {
		var a PortAllocation
		var createdAt, expiresAt string

		err := rows.Scan(
			&a.ID, &a.ProjectName, &a.WebPort, &a.Branch,
			&createdAt, &expiresAt, &a.Status, &a.RepoURL,
		)
		if err != nil {
			return nil, err
		}

		a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
		a.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

		expired = append(expired, a)
	}

	// Mark as expired
	if len(expired) > 0 {
		_, err = r.db.Exec("UPDATE port_allocations SET status = 'expired' WHERE expires_at < ?", now)
		if err != nil {
			return nil, fmt.Errorf("failed to mark expired: %w", err)
		}
	}

	return expired, nil
}

// GetAllocation returns the allocation for a project
func (r *Registry) GetAllocation(projectName string) (*PortAllocation, error) {
	var a PortAllocation
	var createdAt, expiresAt string

	err := r.db.QueryRow(`
		SELECT id, project_name, web_port, branch, created_at, expires_at, status, COALESCE(repo_url, '')
		FROM port_allocations
		WHERE project_name = ?
	`, projectName).Scan(
		&a.ID, &a.ProjectName, &a.WebPort, &a.Branch,
		&createdAt, &expiresAt, &a.Status, &a.RepoURL,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("no allocation found for %s", projectName)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get allocation: %w", err)
	}

	a.CreatedAt, _ = time.Parse(time.RFC3339, createdAt)
	a.ExpiresAt, _ = time.Parse(time.RFC3339, expiresAt)

	return &a, nil
}
