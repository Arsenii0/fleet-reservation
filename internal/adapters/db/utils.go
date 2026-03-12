package db

import (
	"encoding/json"

	"github.com/arsen/fleet-reservation/internal/core/domain"
)

func (sa Adapter) toDomainReservation(reservation Reservation) domain.Reservation {
	var resources []domain.ReservationResource
	for _, resource := range reservation.ReservationResources {
		userConfig := make(map[string]interface{})
		if resource.UserConfig != nil {
			if err := json.Unmarshal(resource.UserConfig, &userConfig); err != nil {
				userConfig = nil
			}
		}

		// Convert the ReservationResource to the domain model
		resources = append(resources, domain.ReservationResource{
			ReservationResourceIndex: resource.ReservationResourceID,
			ReservationID:            resource.ReservationID,
			ResourceID:               resource.ResourceID,
			InstanceID:               resource.InstanceID,
			InstanceState:            domain.InstanceState(resource.InstanceState),
			UserConfig:               userConfig,
		})
	}

	return domain.Reservation{
		ID:                   reservation.ReservationID,
		Status:               reservation.Status,
		ReservationResources: resources,
		Duration:             reservation.Duration,
		CreatedAt:            reservation.CreatedAt.Unix(),
		StartTime:            reservation.StartTime.Unix(),
	}
}
