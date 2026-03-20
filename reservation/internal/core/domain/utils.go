package domain

import (
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
	"github.com/google/uuid"
)

// ConvertDomainReservationToReserveRequest builds the outgoing message sent to workers to deploy instances.
func ConvertDomainReservationToReserveRequest(reservation *Reservation) kafkaevents.ReserveResourceRequestMessage {
	resources := make([]kafkaevents.ResourceDeployRequest, len(reservation.ReservationResources))
	for i, r := range reservation.ReservationResources {
		resources[i] = kafkaevents.ResourceDeployRequest{
			ReservationResourceIndex: r.ReservationResourceIndex,
			ReservationID:            r.ReservationID,
			ResourceID:               r.ResourceID,
			InstanceID:               r.InstanceID,
			InstanceState:            r.InstanceState,
			UserConfig:               r.UserConfig,
		}
	}
	return kafkaevents.ReserveResourceRequestMessage{
		AssociationID: reservation.ID,
		Duration:      reservation.Duration,
		Resources:     resources,
	}
}

// ConvertToReleaseRequest builds the outgoing message sent to workers to terminate instances.
func ConvertToReleaseRequest(reservation *Reservation) kafkaevents.ReleaseInstancesRequestMessage {
	var instanceIDs []uuid.UUID
	for _, res := range reservation.ReservationResources {
		instanceIDs = append(instanceIDs, res.InstanceID)
	}
	return kafkaevents.ReleaseInstancesRequestMessage{
		AssociationID: reservation.ID,
		InstanceIDs:   instanceIDs,
	}
}
