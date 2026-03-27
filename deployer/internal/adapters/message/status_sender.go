package message

import (
	"context"
	"fmt"
	"log"

	"github.com/arsen/fleet-reservation/shared/kafka"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
)

// StatusSenderAdapter sends instance status updates back to the reservation service.
type StatusSenderAdapter struct {
	producer *kafka.KafkaProducerAdapter
}

func NewStatusSenderAdapter(brokers []string, serviceName string) (*StatusSenderAdapter, error) {
	producer, err := kafka.NewKafkaProducerAdapter(brokers, serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to create status sender: %w", err)
	}
	return &StatusSenderAdapter{producer: producer}, nil
}

func (s *StatusSenderAdapter) SendStatusUpdate(ctx context.Context, update kafkaevents.InstanceStatusUpdate) error {
	log.Printf("[status-sender] Sending status update: association=%s instance=%s state=%s ip=%s",
		update.AssociationID, update.InstanceID, update.InstanceState, update.IPAddress)

	_, _, err := s.producer.SendMessage(ctx, kafkaevents.UpdateReservationInstanceStateTopic, update)
	if err != nil {
		return fmt.Errorf("failed to send status update: %w", err)
	}
	return nil
}

func (s *StatusSenderAdapter) Close() error {
	return s.producer.Close()
}
