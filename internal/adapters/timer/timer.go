package timer

import (
	"context"
	"log"
	"math/rand"
	"time"

	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/internal/core/ports"
)

type TimerAdapter struct {
	api         ports.CoreApplicationPort
	interval    time.Duration
	ticker      *time.Ticker
	done        chan bool
	cleanupSize int
}

func NewTimerAdapter(coreApplicationPort ports.CoreApplicationPort, interval time.Duration, cleanupSize int) *TimerAdapter {
	return &TimerAdapter{
		api:         coreApplicationPort,
		interval:    interval,
		ticker:      time.NewTicker(interval),
		done:        make(chan bool),
		cleanupSize: cleanupSize,
	}
}

func (t *TimerAdapter) Start(ctx context.Context) {
	for {
		select {
		case <-t.ticker.C:
			t.cleanupReservations(ctx, t.cleanupSize)
		case <-t.done:
			return
		}
	}
}

func (t *TimerAdapter) Stop() {
	t.done <- true  // stop the channel
	t.ticker.Stop() // stop the ticker
}

func (t TimerAdapter) getExpiredReservations(ctx context.Context) ([]*domain.Reservation, error) {
	var expiredReservations []*domain.Reservation
	reservations, err := t.api.ListAllReservations(ctx)
	if err != nil {
		log.Println("error getting all reservations: ", err)
		return nil, err
	}

	for _, reservation := range reservations {
		if reservation.CanBeReleased() {
			var expiredTime time.Time

			// if the reservation is not started (aka not in Reserved status)
			if reservation.StartTime == 0 {
				expiredTime = time.Unix(reservation.CreatedAt+reservation.Duration, 0)
			} else {
				expiredTime = time.Unix(reservation.StartTime+reservation.Duration, 0)
			}

			if time.Now().After(expiredTime) {
				expiredReservations = append(expiredReservations, reservation)
			}
		}
	}

	log.Printf("found %d/%d expired reservations that can be cleaned up", len(expiredReservations), len(reservations))
	return expiredReservations, nil
}

func selectRandomReservations(reservations []*domain.Reservation, selectSize int) []*domain.Reservation {
	if len(reservations) <= selectSize {
		return reservations
	}

	// Create a new random source and generator
	source := rand.NewSource(time.Now().UnixNano())
	r := rand.New(source)
	// Generate a random permutation of indices
	perm := r.Perm(len(reservations))

	// Select the first selectSize indices from the permutation
	selectedReservations := make([]*domain.Reservation, selectSize)
	for i := 0; i < selectSize; i++ {
		selectedReservations[i] = reservations[perm[i]]
	}

	return selectedReservations
}

func (t TimerAdapter) cleanupReservations(ctx context.Context, cleanupSize int) {
	expiredReservations, err := t.getExpiredReservations(ctx)
	if err != nil {
		log.Println("error getting expired reservations: ", err)
	}

	// Randomly select a portion of the reservations for cleanup
	expiredReservations = selectRandomReservations(expiredReservations, cleanupSize)
	log.Printf("cleaning up %d reservations", len(expiredReservations))

	count := 0
	for _, reservation := range expiredReservations {
		if count >= cleanupSize {
			break
		}
		log.Println("cleaning up reservation: ", reservation.ID)
		// the error is ignored since this function only updates db + send message to kafka
		err = t.api.CleanUpReservation(ctx, reservation.ID)
		if err != nil {
			log.Println("Error cleaning up reservation: ", err)
		}
		count++
	}
	log.Printf("next clean up in %v", t.interval)
}
