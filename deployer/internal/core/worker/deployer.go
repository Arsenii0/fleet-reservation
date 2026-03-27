package worker

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"
)

// DeployResult holds the outcome of a deploy operation.
type DeployResult struct {
	InstanceID uuid.UUID
	ResourceID uuid.UUID
	IPAddress  string
	Err        error
}

// ReleaseResult holds the outcome of a release operation.
type ReleaseResult struct {
	InstanceID uuid.UUID
	Err        error
}

// Deployer is the interface for deploying and releasing instances.
// TODO: replace stub implementation with Terraform-based provisioner.
type Deployer interface {
	Deploy(ctx context.Context, instanceID uuid.UUID, resourceID uuid.UUID, userConfig map[string]interface{}) DeployResult
	Release(ctx context.Context, instanceID uuid.UUID) ReleaseResult
}

// StubDeployer is a no-op deployer that simulates success.
// It logs the operation and returns a fake IP address.
type StubDeployer struct{}

func NewStubDeployer() *StubDeployer {
	return &StubDeployer{}
}

func (d *StubDeployer) Deploy(ctx context.Context, instanceID uuid.UUID, resourceID uuid.UUID, userConfig map[string]interface{}) DeployResult {
	log.Printf("[deployer] Deploy: instance=%s resource=%s config=%v", instanceID, resourceID, userConfig)

	// TODO: replace with actual provisioning (e.g. Terraform)
	fakeIP := fmt.Sprintf("10.0.%d.%d", instanceID[0], instanceID[1])
	log.Printf("[deployer] Deploy success: instance=%s ip=%s", instanceID, fakeIP)

	return DeployResult{
		InstanceID: instanceID,
		ResourceID: resourceID,
		IPAddress:  fakeIP,
	}
}

func (d *StubDeployer) Release(ctx context.Context, instanceID uuid.UUID) ReleaseResult {
	log.Printf("[deployer] Release: instance=%s", instanceID)

	// TODO: replace with actual teardown (e.g. Terraform destroy)
	log.Printf("[deployer] Release success: instance=%s", instanceID)

	return ReleaseResult{InstanceID: instanceID}
}
