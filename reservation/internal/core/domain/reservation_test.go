package domain

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestReservationCanBeReleased(t *testing.T) {
	tests := []struct {
		name        string
		reservation *Reservation
		expected    bool
	}{
		{
			name: "pending",
			reservation: &Reservation{
				Status: ReservationStatusPending,
			},
			expected: false,
		},
		{
			name: "reserved",
			reservation: &Reservation{
				Status: ReservationStatusReserved,
			},
			expected: true,
		},
		{
			name: "closed",
			reservation: &Reservation{
				Status: ReservationStatusClosed,
			},
			expected: false,
		},
		{
			name: "failed",
			reservation: &Reservation{
				Status: ReservationStatusFailed,
			},
			expected: true,
		},
		{
			name: "releasing",
			reservation: &Reservation{
				Status: ReservationStatusReleasing,
			},
			expected: false,
		},
		{
			name: "reserving",
			reservation: &Reservation{
				Status: ReservationStatusReserving,
			},
			expected: false,
		},
		{
			name: "cleaning up",
			reservation: &Reservation{
				Status: ReservationStatusCleaningUp,
			},
			expected: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.reservation.CanBeReleased())
		})
	}
}
