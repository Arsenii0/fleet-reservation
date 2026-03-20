package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arsen/fleet-reservation/fleet-deployer/config"
	"github.com/arsen/fleet-reservation/fleet-deployer/internal/adapters/message"
	"github.com/arsen/fleet-reservation/fleet-deployer/internal/core/worker"
)

func main() {
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	statusSender, err := message.NewStatusSenderAdapter(cfg.KafkaBrokers, cfg.ServiceName)
	if err != nil {
		log.Fatalf("Failed to create status sender: %v", err)
	}
	defer statusSender.Close()

	deployer := worker.NewStubDeployer()
	coordinator := worker.NewDeploymentCoordinator(deployer, statusSender)

	listener, err := message.NewRequestListenerAdapter(cfg.KafkaBrokers, cfg.ServiceName, cfg.KafkaGroupID, coordinator)
	if err != nil {
		log.Fatalf("Failed to create request listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go listener.Run(ctx)

	log.Printf("fleet-deployer started, listening on brokers: %v", cfg.KafkaBrokers)

	<-stopChan
	log.Println("Shutting down fleet-deployer...")
	cancel()
}
