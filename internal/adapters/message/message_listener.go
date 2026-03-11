package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sarama "github.com/IBM/sarama"
	"github.com/arsen/fleet-reservation/internal/adapters/message/kafka"
	"github.com/arsen/fleet-reservation/internal/core/domain"
	"github.com/arsen/fleet-reservation/internal/core/ports"
)

type MessageListenerAdaptor struct {
	consumer *kafka.KafkaConsumerAdapter
	handler  *ReservationConsumerHandler
}

func NewMessageListenerAdaptor(brokers []string, topics []string, serviceName string, groupID string, coreApp ports.CoreApplicationPort) (*MessageListenerAdaptor, error) {
	log.Print("creating reservation message listener")
	handler := NewResourceConsumerHandler(coreApp)
	kafkaConsumer, err := kafka.NewKafkaConsumerAdapter(handler, serviceName, brokers, topics, groupID)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create reservation message listener adaptor with err: %s", domain.AdaptorInitializationError, err)
	}
	return &MessageListenerAdaptor{consumer: kafkaConsumer, handler: handler}, nil
}

func (r *MessageListenerAdaptor) Run(ctx context.Context) {
	r.consumer.StartObserving(ctx)
}

func (r *MessageListenerAdaptor) Close() error {
	return r.consumer.Close()
}

type ReservationConsumerHandler struct {
	coreApp ports.CoreApplicationPort
}

func NewResourceConsumerHandler(app ports.CoreApplicationPort) *ReservationConsumerHandler {
	return &ReservationConsumerHandler{
		coreApp: app,
	}
}

func (h *ReservationConsumerHandler) HandleMessage(ctx context.Context, message *sarama.ConsumerMessage) error {
	var instanceUpdateMsg domain.UpdateReservationInstanceStateRequestMessage
	if err := json.Unmarshal(message.Value, &instanceUpdateMsg); err != nil {
		log.Printf("Failed to unmarshal message: %v", err)
		return err
	}

	return h.coreApp.UpdateReservationStatusRequest(ctx, instanceUpdateMsg)
}
