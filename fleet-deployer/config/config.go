package config

import (
	"fmt"
	"os"
	"strings"
)

type AppConfig struct {
	ServiceName  string
	KafkaBrokers []string
	KafkaGroupID string
}

func LoadConfig() (*AppConfig, error) {
	brokers := os.Getenv("KAFKA_BROKERS")
	if brokers == "" {
		return nil, fmt.Errorf("KAFKA_BROKERS is not set")
	}

	return &AppConfig{
		ServiceName:  getEnv("SERVICE_NAME", "fleet-deployer"),
		KafkaBrokers: strings.Split(brokers, ","),
		KafkaGroupID: getEnv("KAFKA_GROUP_ID", "FLEET_DEPLOYER_GROUP"),
	}, nil
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}
