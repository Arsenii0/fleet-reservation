package application

import (
	"context"

	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
)

type mockDBPort struct {
	mock.Mock
}

func (m *mockDBPort) Add(ctx context.Context, reservation *domain.Reservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *mockDBPort) Get(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error) {
	args := m.Called(ctx, reservationId)
	return args.Get(0).(domain.Reservation), args.Error(1)
}

func (m *mockDBPort) Update(ctx context.Context, reservation *domain.Reservation) error {
	args := m.Called(ctx, reservation)
	return args.Error(0)
}

func (m *mockDBPort) List(ctx context.Context) ([]*domain.Reservation, error) {
	args := m.Called(ctx)
	return args.Get(0).([]*domain.Reservation), args.Error(1)
}

func (m *mockDBPort) Filter(ctx context.Context, userIDs []uuid.UUID, statuses []domain.ReservationStatus, pageSize uint, cursor uuid.UUID) (
	[]domain.Reservation, uuid.UUID, error) {
	args := m.Called(ctx, userIDs, statuses, pageSize, cursor)
	return args.Get(0).([]domain.Reservation), args.Get(1).(uuid.UUID), args.Error(2)
}

type mockResourceMessageSenderPort struct {
	mock.Mock
}

func (m *mockResourceMessageSenderPort) PostMessage(ctx context.Context, topic string, msg interface{}) error {
	args := m.Called(ctx, topic, msg)
	return args.Error(0)
}
