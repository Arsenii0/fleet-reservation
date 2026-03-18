package domain

import "github.com/google/uuid"

// ConvertDomainReservationToReserveRequest builds the outgoing message sent to workers to deploy instances.
func ConvertDomainReservationToReserveRequest(reservation *Reservation) ReserveResourceRequestMessage {
	return ReserveResourceRequestMessage{
		AssociationID: reservation.ID,
		Duration:      reservation.Duration,
		Resources:     reservation.ReservationResources,
	}
}

// ConvertToReleaseRequest builds the outgoing message sent to workers to terminate instances.
func ConvertToReleaseRequest(reservation *Reservation) ReleaseInstancesRequestMessage {
	var instanceIDs []uuid.UUID
	for _, res := range reservation.ReservationResources {
		instanceIDs = append(instanceIDs, res.InstanceID)
	}
	return ReleaseInstancesRequestMessage{
		AssociationID: reservation.ID,
		InstanceIDs:   instanceIDs,
	}
}
