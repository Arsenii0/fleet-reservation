package application

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/arsen/fleet-reservation/deployer/internal/core/ports"
	"github.com/arsen/fleet-reservation/shared/kafkaevents"
	"github.com/google/uuid"
)

// DeployerManager is the core application for the deployer service.
// It receives deploy/release requests and spawns per-instance worker goroutines.
// Workers are agnostic to the underlying technology: they call deploy or destroy
// on the appropriate DeploymentPlugin and send status updates back.
type DeployerManager struct {
	// plugins maps plugin type key (e.g. "terraform") to its implementation.
	// To add a new backend: implement ports.DeploymentPlugin and register it here.
	plugins      map[string]ports.DeploymentPlugin
	statusSender ports.StatusSenderPort
}

func NewDeployerManager(plugins map[string]ports.DeploymentPlugin, statusSender ports.StatusSenderPort) *DeployerManager {
	return &DeployerManager{plugins: plugins, statusSender: statusSender}
}

// HandleDeployRequest spawns one goroutine per resource in the reserve request.
func (m *DeployerManager) HandleDeployRequest(ctx context.Context, msg kafkaevents.ReserveResourceRequestMessage) {
	for _, resource := range msg.Resources {
		resource := resource
		go m.runDeploy(ctx, msg.AssociationID, resource)
	}
}

// HandleReleaseRequest spawns one goroutine per instance in the release request.
func (m *DeployerManager) HandleReleaseRequest(ctx context.Context, msg kafkaevents.ReleaseInstancesRequestMessage) {
	for _, instance := range msg.Instances {
		instance := instance
		go m.runRelease(ctx, msg.AssociationID, instance)
	}
}

func (m *DeployerManager) runDeploy(ctx context.Context, associationID uuid.UUID, resource kafkaevents.ResourceDeployRequest) {
	instanceID := resource.InstanceID
	if instanceID == uuid.Nil {
		instanceID = uuid.New()
	}

	plugin, module, err := m.resolvePlugin(resource.Plugin)
	if err != nil {
		log.Printf("[manager] Deploy failed for instance=%s: %v", instanceID, err)
		m.sendUpdate(ctx, kafkaevents.InstanceStatusUpdate{
			AssociationID: associationID,
			ResourceID:    resource.ResourceID,
			InstanceID:    instanceID,
			InstanceState: kafkaevents.InstanceStateError,
		})
		return
	}

	// Notify reservation that deployment is in progress.
	m.sendUpdate(ctx, kafkaevents.InstanceStatusUpdate{
		AssociationID: associationID,
		ResourceID:    resource.ResourceID,
		InstanceID:    instanceID,
		InstanceState: kafkaevents.InstanceStateDeploying,
	})

	result, err := plugin.Deploy(ctx, instanceID, module)

	update := kafkaevents.InstanceStatusUpdate{
		AssociationID: associationID,
		ResourceID:    resource.ResourceID,
		InstanceID:    instanceID,
		InstanceState: kafkaevents.InstanceStateReserved,
		IPAddress:     result.IPAddress,
		Username:      result.Username,
		Password:      result.Password,
	}
	if err != nil {
		log.Printf("[manager] Deploy failed for instance=%s: %v", instanceID, err)
		update.InstanceState = kafkaevents.InstanceStateError
		update.IPAddress, update.Username, update.Password = "", "", ""
	}

	m.sendUpdate(ctx, update)
}

func (m *DeployerManager) runRelease(ctx context.Context, associationID uuid.UUID, instance kafkaevents.InstanceReleaseInfo) {
	plugin, module, err := m.resolvePlugin(instance.Plugin)
	if err != nil {
		log.Printf("[manager] Release failed for instance=%s: %v", instance.InstanceID, err)
		m.sendUpdate(ctx, kafkaevents.InstanceStatusUpdate{
			AssociationID: associationID,
			InstanceID:    instance.InstanceID,
			InstanceState: kafkaevents.InstanceStateError,
		})
		return
	}

	err = plugin.Destroy(ctx, instance.InstanceID, module)

	update := kafkaevents.InstanceStatusUpdate{
		AssociationID: associationID,
		InstanceID:    instance.InstanceID,
		InstanceState: kafkaevents.InstanceStateReleased,
	}
	if err != nil {
		log.Printf("[manager] Release failed for instance=%s: %v", instance.InstanceID, err)
		update.InstanceState = kafkaevents.InstanceStateError
	}

	m.sendUpdate(ctx, update)
}

// resolvePlugin splits a "type/module" plugin string, looks up the registered implementation,
// and returns both the implementation and the module name.
func (m *DeployerManager) resolvePlugin(plugin string) (ports.DeploymentPlugin, string, error) {
	parts := strings.SplitN(plugin, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return nil, "", fmt.Errorf("invalid plugin string %q", plugin)
	}
	impl, ok := m.plugins[parts[0]]
	if !ok {
		return nil, "", fmt.Errorf("no plugin registered for type %q", parts[0])
	}
	module := ""
	if len(parts) == 2 {
		module = parts[1]
	}
	return impl, module, nil
}

func (m *DeployerManager) sendUpdate(ctx context.Context, update kafkaevents.InstanceStatusUpdate) {
	if err := m.statusSender.SendStatusUpdate(ctx, update); err != nil {
		log.Printf("[manager] Failed to send status update for instance=%s: %v", update.InstanceID, err)
	}
}
