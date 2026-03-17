package ports

import (
	"context"

	"github.com/arsen/fleet-reservation/shared/kafkaevents"
)

// StatusSenderPort sends instance status updates back to the reservation service.
type StatusSenderPort interface {
	SendStatusUpdate(ctx context.Context, update kafkaevents.InstanceStatusUpdate) error
}
