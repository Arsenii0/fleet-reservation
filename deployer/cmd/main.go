package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/arsen/fleet-reservation/deployer/config"
	"github.com/arsen/fleet-reservation/deployer/internal/adapters/message"
	"github.com/arsen/fleet-reservation/deployer/internal/core/application"
	"github.com/arsen/fleet-reservation/deployer/internal/core/ports"
	tfplugin "github.com/arsen/fleet-reservation/deployer/internal/plugins/terraform"
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

	// Build the plugin registry.
	// To add a new deployment backend (e.g. CloudFormation), implement ports.DeploymentPlugin
	// and register it here under a unique type key (the part before "/" in the plugin string).
	pluginRegistry := map[string]ports.DeploymentPlugin{
		"terraform": &tfplugin.TerraformDeployer{},
	}

	manager := application.NewDeployerManager(pluginRegistry, statusSender)

	listener, err := message.NewRequestListenerAdapter(cfg.KafkaBrokers, cfg.ServiceName, cfg.KafkaGroupID, manager)
	if err != nil {
		log.Fatalf("Failed to create request listener: %v", err)
	}
	defer listener.Close()

	ctx, cancel := context.WithCancel(context.Background())

	go listener.Run(ctx)

	log.Printf("deployer started, listening on brokers: %v", cfg.KafkaBrokers)

	<-stopChan
	log.Println("Shutting down deployer...")
	cancel()
}
