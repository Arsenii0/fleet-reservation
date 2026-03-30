package db

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/arsen/fleet-reservation/reservation/internal/core/domain"
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
	ResourceName          string          `gorm:"default:null"`
	Plugin                string          `gorm:"default:null"`
	InstanceID            uuid.UUID       `gorm:"type:uuid"`
	InstanceState         string          `gorm:"not null"`
	IPAddress             string          `gorm:"default:null"`
	Username              string          `gorm:"default:null"`
	Password              string          `gorm:"default:null"` // TODO ArsenP : store as secret
	UserConfig            json.RawMessage `gorm:"type:jsonb"`
}

type Resource struct {
	ResourceID      uuid.UUID `gorm:"type:uuid;primaryKey;default:uuid_generate_v4()"`
	Name            string    `gorm:"uniqueIndex;not null"`
	OperatingSystem string    `gorm:"not null"`
	Plugin          string    `gorm:"default:null"`
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
	if err := db.AutoMigrate(&Reservation{}, &ReservationResource{}, &Resource{}); err != nil {
		return nil, fmt.Errorf("%w: auto-migrate failed: %v", domain.ErrDBConnection, err)
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
			ResourceID:    resource.ResourceID,
			ResourceName:  resource.ResourceName,
			Plugin:        resource.Plugin,
			InstanceState: string(resource.InstanceState),
			IPAddress:     resource.IPAddress,
			Username:      resource.Username,
			Password:      resource.Password,
			UserConfig:    userConfigJSON,
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
			ResourceName:          resource.ResourceName,
			Plugin:                resource.Plugin,
			InstanceID:            resource.InstanceID,
			InstanceState:         string(resource.InstanceState),
			IPAddress:             resource.IPAddress,
			Username:              resource.Username,
			Password:              resource.Password,
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
		reservation := sa.toDomainReservation(reservationEntity)
		reservations = append(reservations, &reservation)
	}
	return reservations, res.Error
}

func (sa Adapter) ListResources(ctx context.Context) ([]domain.Resource, error) {
	var entities []Resource
	if err := sa.orm.WithContext(ctx).Find(&entities).Error; err != nil {
		return nil, err
	}
	resources := make([]domain.Resource, len(entities))
	for i, e := range entities {
		resources[i] = domain.Resource{ID: e.ResourceID, Name: e.Name, OperatingSystem: e.OperatingSystem, Plugin: e.Plugin}
	}
	return resources, nil
}

// EnsureResource upserts a resource by name, setting/updating the plugin field.
func (sa Adapter) EnsureResource(ctx context.Context, name, operatingSystem, plugin string) (domain.Resource, error) {
	var entity Resource
	err := sa.orm.WithContext(ctx).Where("name = ?", name).First(&entity).Error
	if err == nil {
		if entity.Plugin != plugin {
			sa.orm.WithContext(ctx).Model(&entity).Update("plugin", plugin) //nolint:errcheck
			entity.Plugin = plugin
		}
		return domain.Resource{ID: entity.ResourceID, Name: entity.Name, OperatingSystem: entity.OperatingSystem, Plugin: entity.Plugin}, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return domain.Resource{}, err
	}
	entity = Resource{ResourceID: uuid.New(), Name: name, OperatingSystem: operatingSystem, Plugin: plugin}
	if err := sa.orm.WithContext(ctx).Create(&entity).Error; err != nil {
		return domain.Resource{}, err
	}
	return domain.Resource{ID: entity.ResourceID, Name: entity.Name, OperatingSystem: entity.OperatingSystem, Plugin: entity.Plugin}, nil
}
