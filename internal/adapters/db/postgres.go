package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/arsen/fleet-reservation/internal/core/domain"
)

type Adapter struct {
	orm *gorm.DB
}

type Reservation struct {
	ReservationID        uuid.UUID                `gorm:"type:uuid;primary_key;default:uuid_generate_v4()"`
	Status               domain.ReservationStatus `gorm:"not null"`
	Duration             int64                    `gorm:"not null"`
	CreatedAt            time.Time                `gorm:"type:timestamptz;not null"`
	ReservationResources []ReservationResource    `gorm:"foreignkey:ReservationID"`
	StartTime            time.Time                `gorm:"type:timestamptz;not null"`
}

type ReservationResource struct {
	ReservationResourceID uint64          `gorm:"primaryKey;autoIncrement"`
	ReservationID         uuid.UUID       `gorm:"type:uuid;not null;foreignkey:ReservationID"`
	ResourceID            uuid.UUID       `gorm:"type:uuid;not null"`
	InstanceID            uuid.UUID       `gorm:"type:uuid"`
	InstateState          string          `gorm:"not null"`
	UserConfig            json.RawMessage `gorm:"type:jsonb"`
}

func NewDBAdapter(dsn string) (*Adapter, error) {
	db, openErr := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if openErr != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDBConnection, openErr)
	}
	errExt := db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";").Error
	if errExt != nil {
		return nil, fmt.Errorf("%w: %v", domain.ErrDBExtension, errExt)
	}

	return &Adapter{orm: db}, nil
}

func (sa Adapter) Get(ctx context.Context, reservationId uuid.UUID) (domain.Reservation, error) {
	var reservationEntity Reservation
	res := sa.orm.WithContext(ctx).Preload("ReservationResources").First(&reservationEntity, reservationId)
	reservation := sa.toDomainReservation(reservationEntity)
	return reservation, res.Error
}

func (sa Adapter) Add(ctx context.Context, reservation *domain.Reservation) error {
	var resources []ReservationResource
	for _, resource := range reservation.ReservationResources {
		userConfigJSON, err := json.Marshal(resource.UserConfig)
		if err != nil {
			return err
		}

		resources = append(resources, ReservationResource{
			ResourceID:   resource.ResourceID,
			InstateState: string(resource.InstateState),
			UserConfig:   userConfigJSON,
		})
	}
	reservationModel := Reservation{
		ReservationID:        reservation.ID,
		Status:               reservation.Status,
		ReservationResources: resources,
		Duration:             reservation.Duration,
		CreatedAt:            time.Unix(reservation.CreatedAt, 0),
		StartTime:            time.Unix(reservation.StartTime, 0),
	}
	res := sa.orm.WithContext(ctx).Create(&reservationModel)
	if res.Error == nil {
		reservation.ID = reservationModel.ReservationID
	}
	return res.Error
}

func (sa Adapter) Update(ctx context.Context, reservation *domain.Reservation) error {
	_, notFoundErr := sa.Get(ctx, reservation.ID)

	if notFoundErr != nil {
		return notFoundErr
	}

	var resources []ReservationResource
	for _, resource := range reservation.ReservationResources {
		// Marshal UserConfig to JSON
		userConfigJSON, err := json.Marshal(resource.UserConfig)
		if err != nil {
			return err
		}

		// Append the updated resource to the resources slice
		resources = append(resources, ReservationResource{
			ReservationResourceID: resource.ReservationResourceIndex,
			ReservationID:         resource.ReservationID,
			ResourceID:            resource.ResourceID,
			InstanceID:            resource.InstanceID,
			InstateState:          string(resource.InstateState),
			UserConfig:            userConfigJSON,
		})

		// Update the database
		query := sa.orm.WithContext(ctx).Model(&ReservationResource{}).Where("reservation_resource_id = ?", resource.ReservationResourceIndex).Updates(resources[len(resources)-1])

		if query.RowsAffected == 0 {
			return fmt.Errorf("no matching reservation resources found")
		} else if query.Error != nil {
			return query.Error
		}
	}

	reservationModel := Reservation{
		ReservationID:        reservation.ID,
		Status:               reservation.Status,
		ReservationResources: resources,
		Duration:             reservation.Duration,
		CreatedAt:            time.Unix(reservation.CreatedAt, 0),
		StartTime:            time.Unix(reservation.StartTime, 0),
	}
	updatedReservation := sa.orm.WithContext(ctx).Model(&Reservation{}).Where("reservation_id = ?", reservation.ID).Updates(reservationModel)

	if updatedReservation.Error == nil {
		reservation.ID = reservationModel.ReservationID
	}
	return updatedReservation.Error
}

func (sa Adapter) List(ctx context.Context) ([]*domain.Reservation, error) {
	var reservationEntities []Reservation
	res := sa.orm.WithContext(ctx).Preload("ReservationResources").Find(&reservationEntities)
	var reservations []*domain.Reservation

	for _, reservationEntity := range reservationEntities {
		var resources []domain.ReservationResource
		for _, resource := range reservationEntity.ReservationResources {
			resources = append(resources, domain.ReservationResource{
				ReservationResourceIndex: resource.ReservationResourceID,
				ReservationID:            resource.ReservationID,
				ResourceID:               resource.ResourceID,
				InstanceID:               resource.InstanceID,
				InstateState:             domain.InstanceState(resource.InstateState),
			})
		}
		reservation := domain.Reservation{
			ID:                   reservationEntity.ReservationID,
			Status:               reservationEntity.Status,
			ReservationResources: resources,
			Duration:             reservationEntity.Duration,
			CreatedAt:            reservationEntity.CreatedAt.Unix(),
			StartTime:            reservationEntity.StartTime.Unix(),
		}
		reservations = append(reservations, &reservation)
	}
	return reservations, res.Error
}
