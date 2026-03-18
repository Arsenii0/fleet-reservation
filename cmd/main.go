package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arsen/fleet-reservation/config"
	"github.com/arsen/fleet-reservation/internal/adapters/api"
	"github.com/arsen/fleet-reservation/internal/adapters/db"
	"github.com/arsen/fleet-reservation/internal/adapters/message"
	"github.com/arsen/fleet-reservation/internal/adapters/timer"
	"github.com/arsen/fleet-reservation/internal/core/application"
	"github.com/arsen/fleet-reservation/internal/core/domain"
)

func main() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// Load Configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load configuration. Error: %v", err)
	}

	dbAdapter, err := connectToDatabase()
	if err != nil {
		log.Fatal(err)
	}
	log.Println("connected to database")

	messageSender, err := message.NewResourceMessageSenderAdaptor(cfg.KafkaBrokers, cfg.ServiceName)
	if err != nil {
		log.Fatalf("Failed to create kafka producer. Error: %v", err)
	}
	defer func() {
		if err := messageSender.Close(); err != nil {
			log.Printf("Error closing ReservationMessageSenderAdaptor: %v", err)
		}
	}()

	// Core application
	apiApp := application.NewCoreApplication(dbAdapter, messageSender)

	grpcAdapter := api.NewGrpcAdapter(apiApp, cfg.ApplicationPort)
	if cfg.Env == "dev" || cfg.Env == "test" {
		grpcAdapter.RegisterReflection = true
	}
	log.Println("starting reservation api service")

	timerAdapter := timer.NewTimerAdapter(apiApp, cfg.TimerInterval, cfg.CleanupSize)
	log.Println("Starting timer service")

	kafkaConsumer := createKafkaConsumer(cfg, apiApp)
	// Initialize context for Kafka consumer
	ctx, cancel := context.WithCancel(context.Background())
	// Start observing Kafka topics
	go kafkaConsumer.Run(ctx)

	// Start application
	go func() {
		grpcAdapter.Run()
	}()

	log.Println("Reservation service started")

	go timerAdapter.Start(ctx)
	log.Println("Timer service started")

	// Wait for termination signal
	<-stopChan
	log.Println("Shutting down...")
	// Cancel the context to stop the Kafka consumer

	cancel()
	if err := kafkaConsumer.Close(); err != nil {
		log.Printf("Error closing Kafka consumer: %v", err)
	}

	grpcAdapter.Stop()
	log.Println("reservation service stopped")

	timerAdapter.Stop()
	log.Println("timer service stopped")

	signal.Stop(stopChan)
	close(stopChan)
}

func connectToDatabase() (*db.Adapter, error) {

	dbAdapter, err := db.NewDBAdapter(config.GetDBEnv())

	if err != nil {
		switch {
		case errors.Is(err, domain.ErrDBConnection):
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		case errors.Is(err, domain.ErrDBExtension):
			return nil, fmt.Errorf("failed to create database extension: %w", err)
		default:
			return nil, fmt.Errorf("failed to connect to database: %w", err)
		}
	}

	return dbAdapter, nil
}

func createKafkaConsumer(cfg *config.AppConfig, apiApp *application.CoreApplication) *message.MessageListenerAdaptor {

	kafkaConsumer, err := message.NewMessageListenerAdaptor(cfg.KafkaBrokers, []string{domain.UpdateReservationStateTopic},
		cfg.ServiceName, cfg.KafkaGroupID, apiApp)
	if err != nil {
		log.Fatalf("Failed to create Kafka consumer. Error: %v", err)
	}

	return kafkaConsumer
}
