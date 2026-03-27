package ports

import (
	"context"

	"github.com/google/uuid"
)

// DeployResult holds the connection details returned after a successful deploy.
type DeployResult struct {
	IPAddress string
	Username  string
	Password  string
}

// DeploymentPlugin is the port that all deployment backend implementations must satisfy.
// The module string is the second component of the plugin identifier (e.g. "openclaw-guardian"
// from "terraform/openclaw-guardian"). Each implementation uses it to locate the right
// infrastructure definition internally.
type DeploymentPlugin interface {
	Deploy(ctx context.Context, instanceID uuid.UUID, module string) (DeployResult, error)
	Destroy(ctx context.Context, instanceID uuid.UUID, module string) error
}
