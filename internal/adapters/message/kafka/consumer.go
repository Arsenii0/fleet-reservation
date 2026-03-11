package kafka

import (
	"context"
	"log"

	"github.com/IBM/sarama"
)

// MessageHandler is implemented by the caller to process consumed messages.
type MessageHandler interface {
	HandleMessage(ctx context.Context, msg *sarama.ConsumerMessage) error
}

// KafkaConsumerAdapter wraps a sarama consumer group.
type KafkaConsumerAdapter struct {
	group   sarama.ConsumerGroup
	topics  []string
	handler *consumerGroupHandler
}

func NewKafkaConsumerAdapter(handler MessageHandler, serviceName string, brokers []string, topics []string, groupID string) (*KafkaConsumerAdapter, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	cfg.Consumer.Group.Rebalance.GroupStrategies = []sarama.BalanceStrategy{sarama.NewBalanceStrategyRoundRobin()}
	cfg.Consumer.Offsets.Initial = sarama.OffsetNewest
	cfg.ClientID = serviceName

	group, err := sarama.NewConsumerGroup(brokers, groupID, cfg)
	if err != nil {
		return nil, err
	}

	return &KafkaConsumerAdapter{
		group:   group,
		topics:  topics,
		handler: &consumerGroupHandler{handler: handler},
	}, nil
}

// StartObserving blocks and consumes messages until ctx is cancelled.
func (c *KafkaConsumerAdapter) StartObserving(ctx context.Context) {
	for {
		if err := c.group.Consume(ctx, c.topics, c.handler); err != nil {
			log.Printf("kafka consumer error: %v", err)
		}
		if ctx.Err() != nil {
			return
		}
	}
}

func (c *KafkaConsumerAdapter) Close() error {
	return c.group.Close()
}

// consumerGroupHandler implements sarama.ConsumerGroupHandler.
type consumerGroupHandler struct {
	handler MessageHandler
}

func (h *consumerGroupHandler) Setup(_ sarama.ConsumerGroupSession) error   { return nil }
func (h *consumerGroupHandler) Cleanup(_ sarama.ConsumerGroupSession) error { return nil }

func (h *consumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for {
		select {
		case msg, ok := <-claim.Messages():
			if !ok {
				return nil
			}
			if err := h.handler.HandleMessage(session.Context(), msg); err != nil {
				log.Printf("error handling kafka message on topic %s: %v", msg.Topic, err)
			}
			session.MarkMessage(msg, "")
		case <-session.Context().Done():
			return nil
		}
	}
}
