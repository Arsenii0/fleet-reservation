package kafka

import (
	"context"
	"encoding/json"

	"github.com/IBM/sarama"
)

// KafkaProducerAdapter wraps a sarama sync producer.
type KafkaProducerAdapter struct {
	producer sarama.SyncProducer
}

func NewKafkaProducerAdapter(brokers []string, serviceName string) (*KafkaProducerAdapter, error) {
	cfg := sarama.NewConfig()
	cfg.Version = sarama.V2_8_0_0
	cfg.Producer.Return.Successes = true
	cfg.Producer.Return.Errors = true
	cfg.Producer.RequiredAcks = sarama.WaitForAll
	cfg.ClientID = serviceName

	producer, err := sarama.NewSyncProducer(brokers, cfg)
	if err != nil {
		return nil, err
	}

	return &KafkaProducerAdapter{producer: producer}, nil
}

// SendMessage marshals msg to JSON and publishes it to topic.
// Returns the partition and offset of the produced message.
func (p *KafkaProducerAdapter) SendMessage(_ context.Context, topic string, msg interface{}) (int32, int64, error) {
	payload, err := json.Marshal(msg)
	if err != nil {
		return 0, 0, err
	}

	pm := &sarama.ProducerMessage{
		Topic: topic,
		Value: sarama.ByteEncoder(payload),
	}

	return p.producer.SendMessage(pm)
}

func (p *KafkaProducerAdapter) Close() error {
	return p.producer.Close()
}
