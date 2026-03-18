package application

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/reservation/internal/core/ports"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
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
		err := app.producer.PostMessage(ctx, kafkaevents.ReserveResourceRequestTopic, reserveRequestMessage)
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

	err := app.producer.PostMessage(ctx, kafkaevents.ReleaseInstancesRequestTopic, releaseRequestMessage)
	if err != nil {
		return err
	}

	if reservation.StartTime > 0 {
		duration := time.Since(time.Unix(reservation.StartTime, 0))
		hours := int(duration.Hours())
		minutes := int(duration.Minutes()) % 60
		seconds := int(duration.Seconds()) % 60
		log.Printf("Sent release request for reservation %s, duration: %dh%dm%ds",
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

func (app CoreApplication) UpdateReservationStatusRequest(ctx context.Context, request kafkaevents.InstanceStatusUpdate) error {
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

func GetUpdatedReservationResources(reservationResources []domain.ReservationResource, request kafkaevents.InstanceStatusUpdate) ([]domain.ReservationResource, error) {
	var emptyInstanceResource *domain.ReservationResource

	for i, res := range reservationResources {
		// Both nil: update the first resource whose state differs (generic broadcast)
		if request.ResourceID == uuid.Nil && request.InstanceID == uuid.Nil {
			if res.InstanceState != request.InstanceState {
				reservationResources[i].InstanceState = request.InstanceState
				reservationResources[i].IPAddress = request.IPAddress
				return reservationResources, nil
			}
			continue
		}

		// ResourceID nil but InstanceID set (e.g. release response): match by InstanceID only
		if request.ResourceID == uuid.Nil && request.InstanceID != uuid.Nil {
			if res.InstanceID == request.InstanceID {
				reservationResources[i].InstanceState = request.InstanceState
				reservationResources[i].IPAddress = request.IPAddress
				return reservationResources, nil
			}
			continue
		}

		// Normal case: match by ResourceID + InstanceID
		if res.ResourceID == request.ResourceID {
			if res.InstanceID == request.InstanceID {
				reservationResources[i].InstanceState = request.InstanceState
				reservationResources[i].IPAddress = request.IPAddress
				return reservationResources, nil
			}
			if res.InstanceID == uuid.Nil && emptyInstanceResource == nil {
				emptyInstanceResource = &reservationResources[i]
			}
		}
	}

	if emptyInstanceResource != nil {
		emptyInstanceResource.InstanceState = request.InstanceState
		emptyInstanceResource.InstanceID = request.InstanceID
		emptyInstanceResource.IPAddress = request.IPAddress
		return reservationResources, nil
	}

	return nil, fmt.Errorf("no suitable ReservationResource found for updating")
}

func GetUpdatedReservationStatus(resources []domain.ReservationResource) (domain.ReservationStatus, error) {
	if len(resources) == 0 {
		return domain.ReservationStatusClosed, nil
	}

	allReserved := true
	allReleased := true
	hasAnyReleased := false

	for _, res := range resources {
		switch res.InstanceState {
		case kafkaevents.InstanceStateError:
			return domain.ReservationStatusFailed, nil
		case kafkaevents.InstanceStateReserved:
			allReleased = false
		case kafkaevents.InstanceStateReleased:
			allReserved = false
			hasAnyReleased = true
		default: // Pending, Deploying
			allReserved = false
			allReleased = false
		}
	}

	if allReserved {
		return domain.ReservationStatusReserved, nil
	}
	if allReleased {
		return domain.ReservationStatusClosed, nil
	}
	if hasAnyReleased {
		return domain.ReservationStatusReleasing, nil
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

func (app CoreApplication) ListResources(ctx context.Context) ([]domain.Resource, error) {
	return app.db.ListResources(ctx)
}
