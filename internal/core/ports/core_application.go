package ports

import (
	"context"

	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/google/uuid"
)

// Enum with values are now in domain package

type CoreApplicationPort interface {
	CreateReservation(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error)
	GetReservation(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error)
	ReleaseReservation(ctx context.Context, reservationId uuid.UUID) error
	UpdateReservationStatusRequest(ctx context.Context, request domain.UpdateReservationInstanceStateRequestMessage) error
	ListAllReservations(ctx context.Context) ([]*domain.Reservation, error)
	CleanUpReservation(ctx context.Context, reservationId uuid.UUID) error
}
