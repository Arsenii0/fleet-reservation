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
	ReservationResourceIndex uint64    `json:"reservation_resource_index"`
	ReservationID            uuid.UUID `json:"reservation_id"`
	ResourceID               uuid.UUID `json:"resource_id"`
	// ResourceName is the human-readable name of the resource (e.g. "OpenClaw").
	ResourceName string `json:"resource_name"`
	// Plugin identifies the deployment method and module, e.g. "terraform/openclaw-guardian".
	Plugin        string                 `json:"plugin"`
	InstanceID    uuid.UUID              `json:"instance_id"`
	InstanceState InstanceState          `json:"instance_state"`
	UserConfig    map[string]interface{} `json:"user_config"`
}

// ReserveResourceRequestMessage is sent from reservation to deployer to provision instances.
type ReserveResourceRequestMessage struct {
	AssociationID uuid.UUID               `json:"association_id"`
	Duration      int64                   `json:"duration"`
	Resources     []ResourceDeployRequest `json:"resources"`
}

// InstanceReleaseInfo carries per-instance information needed for teardown.
type InstanceReleaseInfo struct {
	InstanceID uuid.UUID `json:"instance_id"`
	// Plugin mirrors the value stored at deploy time so the deployer knows how to destroy.
	Plugin string `json:"plugin"`
}

// ReleaseInstancesRequestMessage is sent from reservation to deployer to terminate instances.
type ReleaseInstancesRequestMessage struct {
	AssociationID uuid.UUID             `json:"association_id"`
	Instances     []InstanceReleaseInfo `json:"instances"`
}

// InstanceStatusUpdate is sent from deployer back to reservation with the result of a deploy or release.
type InstanceStatusUpdate struct {
	AssociationID uuid.UUID     `json:"association_id"`
	ResourceID    uuid.UUID     `json:"resource_id"`
	InstanceID    uuid.UUID     `json:"instance_id"`
	InstanceState InstanceState `json:"instance_state"`
	IPAddress     string        `json:"ip_address"`
	Username      string        `json:"username"`
	Password      string        `json:"password"`
}
