package ports

import (
	"context"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
	"github.com/google/uuid"
)

type CoreApplicationPort interface {
	CreateReservation(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error)
	GetReservation(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error)
	ReleaseReservation(ctx context.Context, reservationId uuid.UUID) error
	UpdateReservationStatusRequest(ctx context.Context, request kafkaevents.InstanceStatusUpdate) error
	ListAllReservations(ctx context.Context) ([]*domain.Reservation, error)
	ListResources(ctx context.Context) ([]domain.Resource, error)
	CleanUpReservation(ctx context.Context, reservationId uuid.UUID) error
}
