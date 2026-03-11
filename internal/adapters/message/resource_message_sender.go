package message

import (
	"context"
	"fmt"
	"log"

	"github.com/arsen/fleet-reservation/internal/adapters/message/kafka"
	"github.com/arsen/fleet-reservation/internal/core/domain"
)

type ResourceMessageSenderAdaptor struct {
	producer *kafka.KafkaProducerAdapter
}

func NewResourceMessageSenderAdaptor(brokers []string, serviceName string) (*ResourceMessageSenderAdaptor, error) {
	log.Print("creating resource message sender adaptor")
	kafkaProd, err := kafka.NewKafkaProducerAdapter(brokers, serviceName)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create kafka producer with err: %s", domain.AdaptorInitializationError, err)
	}
	return &ResourceMessageSenderAdaptor{producer: kafkaProd}, nil
}

// Implements the ReservationMessageSenderPort interface
func (r *ResourceMessageSenderAdaptor) PostMessage(ctx context.Context, topic string, msg interface{}) error {
	_, _, err := r.producer.SendMessage(ctx, topic, msg)
	if err != nil {
		return fmt.Errorf("Failed to send message with err: %s", err)
	}
	return nil
}

func (r *ResourceMessageSenderAdaptor) Close() error {
	return r.producer.Close()
}
