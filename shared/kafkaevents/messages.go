package kafkaevents

import "github.com/google/uuid"

// InstanceState is the lifecycle state of a deployed instance.
type InstanceState string

const (
	InstanceStatePending   InstanceState = "Pending"
	InstanceStateDeploying InstanceState = "Deploying"
	InstanceStateReserved  InstanceState = "Reserved"
	InstanceStateError     InstanceState = "Error"
	InstanceStateReleased  InstanceState = "Released"
)

// ResourceDeployRequest describes a single resource within a deploy request.
type ResourceDeployRequest struct {
	ReservationResourceIndex uint64                 `json:"reservation_resource_index"`
	ReservationID            uuid.UUID              `json:"reservation_id"`
	ResourceID               uuid.UUID              `json:"resource_id"`
	InstanceID               uuid.UUID              `json:"instance_id"`
	InstanceState            InstanceState          `json:"instance_state"`
	UserConfig               map[string]interface{} `json:"user_config"`
}

// ReserveResourceRequestMessage is sent from reservation to fleet-deployer to provision instances.
type ReserveResourceRequestMessage struct {
	AssociationID uuid.UUID               `json:"association_id"`
	Duration      int64                   `json:"duration"`
	Resources     []ResourceDeployRequest `json:"resources"`
}

// ReleaseInstancesRequestMessage is sent from reservation to fleet-deployer to terminate instances.
type ReleaseInstancesRequestMessage struct {
	AssociationID uuid.UUID   `json:"association_id"`
	InstanceIDs   []uuid.UUID `json:"instance_ids"`
}

// InstanceStatusUpdate is sent from fleet-deployer back to reservation with the result of a deploy or release.
type InstanceStatusUpdate struct {
	AssociationID uuid.UUID     `json:"association_id"`
	ResourceID    uuid.UUID     `json:"resource_id"`
	InstanceID    uuid.UUID     `json:"instance_id"`
	InstanceState InstanceState `json:"instance_state"`
	IPAddress     string        `json:"ip_address"`
}
