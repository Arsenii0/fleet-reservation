package worker

import (
	"context"
	"log"

	"github.com/arsen/fleet-reservation/fleet-deployer/internal/core/ports"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
	"github.com/google/uuid"
)

// DeploymentCoordinator spawns workers to handle deploy and release requests.
// Each task runs in its own goroutine (ephemeral worker).
// TODO: replace goroutines with ephemeral Docker containers per task.
type DeploymentCoordinator struct {
	deployer     Deployer
	statusSender ports.StatusSenderPort
}

func NewDeploymentCoordinator(deployer Deployer, statusSender ports.StatusSenderPort) *DeploymentCoordinator {
	return &DeploymentCoordinator{
		deployer:     deployer,
		statusSender: statusSender,
	}
}

// HandleDeployRequest spawns one worker goroutine per resource in the reserve request.
func (c *DeploymentCoordinator) HandleDeployRequest(ctx context.Context, msg kafkaevents.ReserveResourceRequestMessage) {
	for _, resource := range msg.Resources {
		resource := resource // capture for goroutine
		go c.runDeploy(ctx, msg.AssociationID, resource)
	}
}

// HandleReleaseRequest spawns one worker goroutine per instance in the release request.
func (c *DeploymentCoordinator) HandleReleaseRequest(ctx context.Context, msg kafkaevents.ReleaseInstancesRequestMessage) {
	for _, instanceID := range msg.InstanceIDs {
		instanceID := instanceID // capture for goroutine
		go c.runRelease(ctx, msg.AssociationID, instanceID)
	}
}

func (c *DeploymentCoordinator) runDeploy(ctx context.Context, associationID uuid.UUID, resource kafkaevents.ResourceDeployRequest) {
	// Ensure instance ID is set; if the reservation service didn't pre-generate one, create it here.
	instanceID := resource.InstanceID
	if instanceID == uuid.Nil {
		instanceID = uuid.New()
	}

	result := c.deployer.Deploy(ctx, instanceID, resource.ResourceID, resource.UserConfig)

	update := kafkaevents.InstanceStatusUpdate{
		AssociationID: associationID,
		ResourceID:    resource.ResourceID,
		InstanceID:    instanceID,
		InstanceState: kafkaevents.InstanceStateReserved,
		IPAddress:     result.IPAddress,
	}

	if result.Err != nil {
		log.Printf("[coordinator] Deploy failed: instance=%s err=%v", instanceID, result.Err)
		update.InstanceState = kafkaevents.InstanceStateError
		update.IPAddress = ""
	}

	if err := c.statusSender.SendStatusUpdate(ctx, update); err != nil {
		log.Printf("[coordinator] Failed to send status update for instance=%s: %v", instanceID, err)
	}
}

func (c *DeploymentCoordinator) runRelease(ctx context.Context, associationID uuid.UUID, instanceID uuid.UUID) {
	result := c.deployer.Release(ctx, instanceID)

	update := kafkaevents.InstanceStatusUpdate{
		AssociationID: associationID,
		InstanceID:    instanceID,
		InstanceState: kafkaevents.InstanceStateReleased,
	}

	if result.Err != nil {
		log.Printf("[coordinator] Release failed: instance=%s err=%v", instanceID, result.Err)
		update.InstanceState = kafkaevents.InstanceStateError
	}

	if err := c.statusSender.SendStatusUpdate(ctx, update); err != nil {
		log.Printf("[coordinator] Failed to send status update for instance=%s: %v", instanceID, err)
	}
}
