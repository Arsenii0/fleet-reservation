package config

import (
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

type AppConfig struct {
	Env             string
	DBHost          string
	DBPort          int
	DBUser          string
	DBPassword      string
	DBName          string
	ApplicationPort int
	HTTPPort        int
	KafkaBrokers    []string
	KafkaGroupID    string
	TimerInterval   time.Duration
	CleanupSize     int
	ServiceName     string
}

func getEnv(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

func getEnvRequired(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is not set", key)
	}
	return v, nil
}

func getEnvInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(strings.Trim(v, "\""))
	if err != nil {
		return 0, fmt.Errorf("invalid %s value %q: %v", key, v, err)
	}
	return n, nil
}

// LoadConfig loads application config from the environment
func LoadConfig() (*AppConfig, error) {
	var err error
	var cfg AppConfig

	cfg.Env = getEnv("ENV", "local")
	cfg.ServiceName = getEnv("SERVICE_NAME", "ReservationService")
	cfg.KafkaGroupID = getEnv("KAFKA_GROUP_ID", "UPDATE_RESOURCE_GROUP_ID")

	if cfg.DBHost, err = getEnvRequired("DB_HOST"); err != nil {
		return nil, err
	}
	if cfg.DBUser, err = getEnvRequired("DB_USER"); err != nil {
		return nil, err
	}
	if cfg.DBPassword, err = getEnvRequired("DB_PASSWORD"); err != nil {
		return nil, err
	}
	if cfg.DBName, err = getEnvRequired("DB_NAME"); err != nil {
		return nil, err
	}

	kafkaBrokers, err := getEnvRequired("KAFKA_BROKERS")
	if err != nil {
		return nil, err
	}
	cfg.KafkaBrokers = strings.Split(kafkaBrokers, ",")

	if cfg.DBPort, err = getEnvInt("DB_PORT", 5432); err != nil {
		return nil, err
	}
	if cfg.ApplicationPort, err = getEnvInt("APPLICATION_PORT", 50051); err != nil {
		return nil, err
	}
	if cfg.HTTPPort, err = getEnvInt("HTTP_PORT", 8080); err != nil {
		return nil, err
	}
	if cfg.CleanupSize, err = getEnvInt("CLEANUP_SIZE", 20); err != nil {
		return nil, err
	}

	timerSecs, err := getEnvInt("TIMER_INTERVAL_IN_SECONDS", 1200)
	if err != nil {
		return nil, err
	}
	cfg.TimerInterval = time.Duration(timerSecs) * time.Second

	return &cfg, nil
}

func GetDBEnv() string {
	cfg, err := LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}
	return fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPassword, cfg.DBName)
}
