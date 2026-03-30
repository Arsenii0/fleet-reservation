package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	sarama "github.com/IBM/sarama"
	"github.com/arsen/fleet-reservation/deployer/internal/core/ports"
	"github.com/arsen/fleet-reservation/shared/kafka"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
)

// RequestListenerAdapter consumes deploy and release requests from the reservation service.
type RequestListenerAdapter struct {
	consumer *kafka.KafkaConsumerAdapter
	handler  *deployRequestHandler
}

func NewRequestListenerAdapter(brokers []string, serviceName string, groupID string, manager ports.DeployerManagerPort) (*RequestListenerAdapter, error) {
	topics := []string{
		kafkaevents.ReserveResourceRequestTopic,
		kafkaevents.ReleaseInstancesRequestTopic,
	}

	handler := &deployRequestHandler{manager: manager}
	consumer, err := kafka.NewKafkaConsumerAdapter(handler, serviceName, brokers, topics, groupID)
	if err != nil {
		return nil, fmt.Errorf("failed to create request listener: %w", err)
	}

	return &RequestListenerAdapter{consumer: consumer, handler: handler}, nil
}

func (r *RequestListenerAdapter) Run(ctx context.Context) {
	r.consumer.StartObserving(ctx)
}

func (r *RequestListenerAdapter) Close() error {
	return r.consumer.Close()
}

type deployRequestHandler struct {
	manager ports.DeployerManagerPort
}

func (h *deployRequestHandler) HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error {
	switch msg.Topic {
	case kafkaevents.ReserveResourceRequestTopic:
		var req kafkaevents.ReserveResourceRequestMessage
		if err := json.Unmarshal(msg.Value, &req); err != nil {
			log.Printf("[listener] Failed to unmarshal ReserveResourceRequestMessage: %v", err)
			return err
		}
		log.Printf("[listener] Received deploy request: association=%s resources=%d", req.AssociationID, len(req.Resources))
		h.manager.HandleDeployRequest(ctx, req)

	case kafkaevents.ReleaseInstancesRequestTopic:
		var req kafkaevents.ReleaseInstancesRequestMessage
		if err := json.Unmarshal(msg.Value, &req); err != nil {
			log.Printf("[listener] Failed to unmarshal ReleaseInstancesRequestMessage: %v", err)
			return err
		}
		log.Printf("[listener] Received release request: association=%s instances=%d", req.AssociationID, len(req.Instances))
		h.manager.HandleReleaseRequest(ctx, req)

	default:
		log.Printf("[listener] Unknown topic: %s", msg.Topic)
	}

	return nil
}
