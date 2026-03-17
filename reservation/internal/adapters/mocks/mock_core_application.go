package mocks

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/stretchr/testify/mock"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
)

type MockCoreApplicationPort struct {
	mu sync.Mutex // Mutex to protect calls to the mock from concurrent calls
	mock.Mock
}

func (m *MockCoreApplicationPort) CreateReservation(ctx context.Context, reservation domain.Reservation) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx, reservation)
	return ret.Get(0).(domain.Reservation), ret.Error(1)
}

func (m *MockCoreApplicationPort) GetReservation(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx, reservationId)
	return ret.Get(0).(domain.Reservation), ret.Error(1)
}

func (m *MockCoreApplicationPort) ReleaseReservation(ctx context.Context, reservationId uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx, reservationId)
	return ret.Error(0)
}

func (m *MockCoreApplicationPort) UpdateReservationStatusRequest(ctx context.Context, request kafkaevents.InstanceStatusUpdate) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx, request)
	return ret.Error(0)
}

func (m *MockCoreApplicationPort) ListAllReservations(ctx context.Context) ([]*domain.Reservation, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx)
	return ret.Get(0).([]*domain.Reservation), ret.Error(1)
}

func (m *MockCoreApplicationPort) CleanUpReservation(ctx context.Context, reservationId uuid.UUID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	ret := m.Called(ctx, reservationId)
	return ret.Error(0)
}
