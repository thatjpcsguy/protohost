package registry

import "time"

// PortAllocation represents a port allocation record
type PortAllocation struct {
	ID          int
	ProjectName string
	WebPort     int
	Branch      string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Status      string // "running", "stopped", "expired"
	RepoURL     string
}
