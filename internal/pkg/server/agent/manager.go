/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package agent

import (
	"context"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/edgeplugin"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-edge-inventory-proxy-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"github.com/satori/go.uuid"
	"time"
)

const DefaultTimeout = 30 * time.Second
const UninstallOp = "uninstall"
const CorePluging = "core"

type Manager struct{
	config config.Config
	provider asset.Provider
	notifier Notifier
	// managementClient that connects with the proxy on the management cluster. Notice that this will be an async
	// proxy for most operations as the inventory manager is not directly exposed.
	managementClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient
}

func NewManager(cfg config.Config, assetProvider asset.Provider, notifier Notifier, managementClient grpc_edge_inventory_proxy_go.EdgeInventoryProxyClient) Manager{
	return Manager{cfg,assetProvider, notifier, managementClient}
}

func (m * Manager) AgentJoin(request *grpc_edge_controller_go.AgentJoinRequest) (*grpc_inventory_manager_go.AgentJoinResponse, derrors.Error) {
	log.Debug().Str("agentID", request.AgentId).Msg("agent request join")
	ctx, cancel := context.WithTimeout(context.Background(), DefaultTimeout)
	defer cancel()
	toSend := &grpc_inventory_manager_go.AgentJoinRequest{
		OrganizationId:       m.config.OrganizationId,
		EdgeControllerId:     m.config.EdgeControllerId,
		AgentId:              request.AgentId,
		Labels:               request.Labels,
		Os:                   request.Os,
		Hardware:             request.Hardware,
		Storage:              request.Storage,
	}
	response, err := m.managementClient.AgentJoin(ctx, toSend)
	if err != nil{
		log.Warn().Str("agentID", request.AgentId).Str("trace", conversions.ToDerror(err).DebugReport()).Msg("cannot join agent")
		return nil, conversions.ToDerror(err)
	}

	// add agent
	err = m.provider.AddManagedAsset(entities.AgentJoinInfo{
		Created: time.Now().Unix(),
		AssetId: response.AssetId,
		Token: response.Token,
	})
	if err != nil{
		log.Warn().Str("agentID", request.AgentId).Str("assetId", response.AssetId).Str("trace", conversions.ToDerror(err).DebugReport()).Msg("cannot add Asset")
		return nil, conversions.ToDerror(err)
	}


	// add Token
	log.Debug().Str("agentID", request.AgentId).Str("assetID", response.AssetId).Msg("Agent joined successfully")
	return response, nil


}

func (m * Manager) AgentStart(info *grpc_inventory_manager_go.AgentStartInfo) derrors.Error {
	log.Debug().Str("assetID", info.AssetId).Msg("agent started")
	err := m.notifier.NotifyAgentStart(info)
	if err != nil{
		log.Warn().Str("trace", err.DebugReport()).Msg("error notifying agent start event")
		return err
	}
	return nil
}

func (m * Manager) AgentCheck(request *grpc_edge_controller_go.AgentCheckRequest, ip string) (*grpc_edge_controller_go.CheckResult, derrors.Error) {
	// TODO: Verify clock sync
	log.Info().Str("assetID", request.AssetId).Str("ip", ip).Msg("agent check")

	exists, asset := m.notifier.PendingUnInstall(request.AssetId)
	// verify if this agent is pending to be uninstalled
	if exists{

		err := m.provider.RemoveManagedAsset(request.AssetId)
		if err != nil {
			log.Warn().Str("assetID", request.AssetId).Str("trace", err.DebugReport()).Msg("error removing agent")
		}

		m.notifier.RemovePendingUninstall(request.AssetId)

		// TODO: proccess the information received
		return m.SendUninstallMessageToAgent(asset)
	}

	m.notifier.AgentAlive(request.AssetId, ip)

	// Handle plugin data
	for _, data := range (request.GetPluginData()) {
		derr := edgeplugin.HandleAgentData(request.GetAssetId(), data)
		// TODO: Think about failure modes - do we want to collect
		// errors and handle what we can, handle nothing if there is
		// an error (which requires rollback or commit) or just
		// stop on the first error (current mode of operation).
		if derr != nil {
			return nil, derr
		}
	}

	pending, err := m.provider.GetPendingOperations(request.AssetId, true)
	if err != nil {
		log.Error().Str("trace", err.DebugReport()).Msg("cannot retrieve pending operations for an agent")
		// In this case the error is not returned to the agent as it cannot do anything.
		return &grpc_edge_controller_go.CheckResult{}, nil
	}
	log.Info().Str("assetID", request.AssetId).Int("pending operation", len(pending)).Msg("sending pending operation to the agent")

	// Return empty message
	if len(pending) == 0 {
		return &grpc_edge_controller_go.CheckResult{}, nil
	}
	// Transform the result into gRPC structures.
	result := make([]*grpc_inventory_manager_go.AgentOpRequest, 0, len(pending))
	for _, p := range pending {
		result = append(result, p.ToGRPC())
	}
	return &grpc_edge_controller_go.CheckResult{
		PendingRequests: result,
	}, nil


}

func (m * Manager) CallbackAgentOperation(response *grpc_inventory_manager_go.AgentOpResponse) derrors.Error {
	log.Debug().Str("assetID", response.AssetId).Str("status", response.Status.String()).Msg("agent callback")
	err := m.notifier.NotifyCallback(response)
	if err != nil{
		log.Warn().Str("trace", err.DebugReport()).Msg("error notifying agent callback")
		return err
	}
	return nil
}

// removes all the agent information (edge-controller -> agent and agent -> token)
func (m *Manager) RemoveAgent(request entities.FullAssetId) derrors.Error{

	err := m.provider.RemoveManagedAsset(request.AssetId)
	if err != nil {
		return err
	}
	return nil
}

// SendUninstallMessageToAgent operation to send a message to an agent to inform it is going to be uninstalled
func (m *Manager) SendUninstallMessageToAgent (request entities.FullAssetId) (*grpc_edge_controller_go.CheckResult, derrors.Error){

	result := []*grpc_inventory_manager_go.AgentOpRequest{{
		OrganizationId: request.OrganizationId,
		EdgeControllerId: request.EdgeControllerId,
		AssetId: request.AssetId,
		OperationId: uuid.NewV4().String(),
		Operation: UninstallOp,
		Plugin: CorePluging,
	}}
	return &grpc_edge_controller_go.CheckResult{
		PendingRequests: result,
	}, nil
}