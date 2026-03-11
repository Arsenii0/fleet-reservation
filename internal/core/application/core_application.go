package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/internal/core/ports"
	"github.com/google/uuid"
)

type CoreApplication struct {
	db       ports.DBPort
	producer ports.ResourceMessageSenderPort
}

func NewCoreApplication(dbPort ports.DBPort, producerPort ports.ResourceMessageSenderPort) *CoreApplication {
	return &CoreApplication{
		db:       dbPort,
		producer: producerPort,
	}
}

func (app CoreApplication) CreateReservation(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error) {
	// Add the reservation to the database
	err := app.db.Add(ctx, &reservation)
	if err != nil {
		return domain.Reservation{}, err
	}

	if len(reservation.ReservationResources) > 0 {
		reserveRequestMessage := domain.ConvertDomainReservationToReserveRequest(&reservation)
		err := app.producer.PostMessage(ctx, domain.ReserveResourceRequestTopic, reserveRequestMessage)
		if err != nil {
			// If the message fails to send, update the reservation status to Failed
			reservation.Status = domain.ReservationStatusFailed
			updateError := app.db.Update(ctx, &reservation)
			if updateError != nil {
				return domain.Reservation{}, updateError
			}

			return domain.Reservation{}, err
		}
	}

	return reservation, nil
}

func (app CoreApplication) GetReservation(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error) {
	reservation, err := app.db.Get(ctx, reservationId)
	if err != nil {
		return domain.Reservation{}, err
	}
	return reservation, nil
}

func (app CoreApplication) ReleaseReservation(ctx context.Context, reservationId uuid.UUID) error {
	reservation, err := app.db.Get(ctx, reservationId)
	if err != nil {
		log.Printf("Error getting reservation %s: %v", reservationId, err)
		return domain.ErrReservationNotFound
	}

	if !reservation.CanBeReleased() {
		log.Printf("Reservation %s cannot be released", reservationId)
		return domain.ErrReservationCannotBeReleased
	}

	reservation.Status = domain.ReservationStatusReleasing

	err = app.db.Update(ctx, &reservation)
	if err != nil {
		return err
	}

	err = app.sendReleaseRequest(ctx, &reservation)
	if err != nil {
		return err
	}

	return nil
}

func (app CoreApplication) sendReleaseRequest(ctx context.Context, reservation *domain.Reservation) error {
	releaseRequestMessage := domain.ConvertToReleaseRequest(reservation)

	err := app.producer.PostMessage(ctx, domain.ReleaseInstancesRequestTopic, releaseRequestMessage)
	if err != nil {
		return err
	}

	if reservation.StartTime > 0 {
		duration := time.Since(time.Unix(reservation.StartTime, 0))
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60
		log.Printf("Sent release request for reservation %s by %s, duration: %dh%dm%ds",
			reservation.ID, hours, minutes, seconds)
	}

	return nil
}

func (app CoreApplication) CleanUpReservation(ctx context.Context, reservationId uuid.UUID) error {
	reservation, err := app.db.Get(ctx, reservationId)
	if err != nil {
		return err
	}

	reservation.Status = domain.ReservationStatusCleaningUp

	err = app.db.Update(ctx, &reservation)
	if err != nil {
		return err
	}

	err = app.sendReleaseRequest(ctx, &reservation)
	if err != nil {
		return err
	}

	return nil
}

func (app CoreApplication) UpdateReservationStatusRequest(ctx context.Context, request domain.UpdateReservationInstanceStateRequestMessage) error {
	reservation, err := app.db.Get(ctx, request.AssociationID) // use association id as reservation id
	if err != nil {
		return err
	}

	updatedReservationResources, err := GetUpdatedReservationResources(reservation.ReservationResources, request)
	if updatedReservationResources == nil {
		return err
	}

	newReservationStatus, err := GetUpdatedReservationStatus(updatedReservationResources)
	if err != nil {
		return err
	}

	reservation.Status = newReservationStatus
	reservation.ReservationResources = updatedReservationResources

	// Update the start_time if Reservation is in Reserved status
	if newReservationStatus == domain.ReservationStatusReserved {
		reservation.StartTime = time.Now().Unix()
	}

	err = app.db.Update(ctx, &reservation)
	if err != nil {
		return err
	}

	return nil
}

func GetUpdatedReservationResources(reservationResources []domain.ReservationResource, request domain.UpdateReservationInstanceStateRequestMessage) ([]domain.ReservationResource, error) {
	// Keep track of a ReservationResource with an empty InstanceID
	var emptyInstanceResource *domain.ReservationResource

	// Look for a ReservationResource with the InstanceID from the message
	for i, res := range reservationResources {
		// Handle the case where the ResourceID is zero
		// If there is no specific instance assigned to the reservation request, .
		// This means the resource manager does not know the specific ResourceID.
		// It returns ResourceID and InstanceID might be zero (Nil UUID)
		// We need to handle this case appropriately by updating the reservation
		// state based on the provided instance state.
		if request.ResourceID == uuid.Nil {
			if request.InstanceID == uuid.Nil {
				if res.InstateState != request.InstanceState {
					reservationResources[i].InstateState = request.InstanceState
					return reservationResources, nil
				}
			}
		}

		if res.ResourceID == request.ResourceID {
			if res.InstanceID == request.InstanceID {
				reservationResources[i].InstateState = request.InstanceState
				return reservationResources, nil
			}

			// Record the first ReservationResource with an empty InstanceID
			if res.InstanceID == uuid.Nil && emptyInstanceResource == nil {
				emptyInstanceResource = &reservationResources[i]
			}
		}
	}

	// If no InstanceID from the message exists, update the ReservationResource with an empty InstanceID
	if emptyInstanceResource != nil {
		emptyInstanceResource.InstateState = request.InstanceState
		emptyInstanceResource.InstanceID = request.InstanceID
		return reservationResources, nil
	}

	// If there are no empty InstanceIDs and the InstanceID from the message doesn't exist, return an error
	return nil, fmt.Errorf("no suitable ReservationResource found for updating")
}

func GetUpdatedReservationStatus(resources []domain.ReservationResource) (domain.ReservationStatus, error) {
	if len(resources) == 0 {
		return domain.ReservationStatusClosed, nil
	}

	allReserved := true
	for _, res := range resources {
		if res.InstateState == domain.InstanceStateError {
			return domain.ReservationStatusFailed, nil
		}
		if res.InstateState != domain.InstanceStateReserved {
			allReserved = false
		}
	}

	if allReserved {
		return domain.ReservationStatusReserved, nil
	}

	return domain.ReservationStatusReserving, nil
}

func (app CoreApplication) ListAllReservations(ctx context.Context) ([]*domain.Reservation, error) {
	reservations, err := app.db.List(ctx)
	if err != nil {
		return nil, err
	}
	return reservations, nil
}
