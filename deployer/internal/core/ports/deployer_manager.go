package ports

import (
	"context"

	"github.com/arsen/fleet-reservation/shared/kafkaevents"
)

// DeployerManagerPort is the port through which the message adapter drives
// the core application.  The concrete implementation is application.DeployerManager.
type DeployerManagerPort interface {
	HandleDeployRequest(ctx context.Context, msg kafkaevents.ReserveResourceRequestMessage)
	HandleReleaseRequest(ctx context.Context, msg kafkaevents.ReleaseInstancesRequestMessage)
}
