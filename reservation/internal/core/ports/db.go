package ports

import (
	"context"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
	"github.com/google/uuid"
)

type DBPort interface {
	Get(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error)
	Add(ctx context.Context, reservation *domain.Reservation) error
	Update(ctx context.Context, reservation *domain.Reservation) error
	List(ctx context.Context) ([]*domain.Reservation, error)
	ListResources(ctx context.Context) ([]domain.Resource, error)
}
