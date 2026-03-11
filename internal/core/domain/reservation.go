package domain

import (
	"time"

	"github.com/google/uuid"
)

type InstanceState string

const (
	InstanceStatePending   InstanceState = "Pending"
	InstanceStateDeploying InstanceState = "Deploying"
	InstanceStateReserved  InstanceState = "Reserved"
	InstanceStateError     InstanceState = "Error"
)

const UpdateReservationStateTopic = "update-reservation-instance-state-request"
const ReserveResourceRequestTopic = "reserve-resource-request"
const ReleaseInstancesRequestTopic = "release-instances-request"

type UpdateReservationInstanceStateRequestMessage struct {
	AssociationID uuid.UUID     `json:"association_id"`
	ResourceID    uuid.UUID     `json:"resource_id"`
	InstanceID    uuid.UUID     `json:"instance_id"`
	InstanceState InstanceState `json:"instance_state"`
}

type ReservationStatus string

const (
	ReservationStatusPending    ReservationStatus = "Pending"
	ReservationStatusReserving  ReservationStatus = "Reserving"
	ReservationStatusReserved   ReservationStatus = "Reserved"
	ReservationStatusFailed     ReservationStatus = "Failed"
	ReservationStatusReleasing  ReservationStatus = "Releasing"
	ReservationStatusClosed     ReservationStatus = "Closed"
	ReservationStatusCleaningUp ReservationStatus = "CleaningUp"
)

// TODO ArsenP : add instance state

// Resource represents a reservable resource.
type Resource struct {
	ID              uuid.UUID `json:"id"`
	Name            string    `json:"name"`
	OperatingSystem string    `json:"operating_system"`
}

// Instance represents a deployed compute instance.
type Instance struct {
	InstanceID uuid.UUID              `json:"instance_id"`
	ResourceID uuid.UUID              `json:"resource_id"`
	UserData   map[string]interface{} `json:"user_data"`
}

// ReserveResourceRequestMessage is sent from the reservation service to the worker to deploy instances.
type ReserveResourceRequestMessage struct {
	AssociationID uuid.UUID             `json:"association_id"`
	Duration      int64                 `json:"duration"`
	Resources     []ReservationResource `json:"resources"`
}

// ReleaseResourceRequestMessage is sent from the worker back to the reservation service
// to update reservation status when an instance is destroyed, released, or changes state.
type ReleaseResourceRequestMessage struct {
	AssociationID uuid.UUID     `json:"association_id"`
	ResourceID    uuid.UUID     `json:"resource_id"`
	InstanceID    uuid.UUID     `json:"instance_id"`
	InstanceState InstanceState `json:"instance_state"`
}

// ReleaseInstancesRequestMessage is sent from the reservation service to the worker to terminate instances.
type ReleaseInstancesRequestMessage struct {
	AssociationID uuid.UUID   `json:"association_id"`
	InstanceIDs   []uuid.UUID `json:"instance_ids"`
}

type Reservation struct {
	// ID of the reservation.
	ID uuid.UUID `json:"id"`

	// Status of the reservation.
	Status ReservationStatus `json:"status"`
	// Duration of the reservation in seconds.
	Duration int64 `json:"duration"`
	// Resources of the reservation.
	ReservationResources []ReservationResource `json:"reservation_resources"`
	// Timestamp of the reservation creation.
	CreatedAt int64 `json:"created_at"`
	// Timestamp of the reservation start time (when reservation is in reserved status).
	StartTime int64 `json:"start_time"`
}

type ReservationResource struct {
	// Index of the resource in the reservation.
	ReservationResourceIndex uint64 `json:"reservation_resource_index"`
	// ID of the reservation.
	ReservationID uuid.UUID `json:"reservation_id"`
	// ID of the resource.
	ResourceID uuid.UUID `json:"resource_id"`
	// ID of the instance. Can be empty if the instance is not yet deployed.
	InstanceID uuid.UUID `json:"instance_id"`
	// Status of the instance. Can be empty if the instance is not yet deployed.
	InstateState InstanceState `json:"instance_state"`
	// Map of User configuration parameters for the resource.
	UserConfig map[string]interface{} `json:"user_config"`
}

// Generate a new reservation with default parameters
func NewReservation(duration int64, resources []ReservationResource) Reservation {
	return Reservation{
		ID:                   uuid.New(),
		Status:               ReservationStatusPending,
		ReservationResources: resources,
		Duration:             duration,
		CreatedAt:            time.Now().Unix(),
		StartTime:            0, // default initialization since the reservation is not yet reserved
	}
}

// Generate a new requested resource with default parameters
func NewReservationResource(requestedResourceId uuid.UUID, userConfig map[string]string) ReservationResource {
	reservationResource := ReservationResource{
		ResourceID: requestedResourceId,
	}

	if len(userConfig) > 0 {
		reservationResource.UserConfig = make(map[string]interface{})
		for key, value := range userConfig {
			reservationResource.UserConfig[key] = value
		}
	}

	return reservationResource
}

func (r *Reservation) CanBeReleased() bool {
	return r.Status == ReservationStatusReserved || r.Status == ReservationStatusFailed
}
