/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package agent

import (
	"context"
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-edge-inventory-proxy-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"time"
)

const DefaultTimeout = 30 * time.Second

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
	m.provider.AddJoinToken(response.Token)
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

func (m * Manager) AgentCheck(request *grpc_edge_controller_go.AgentCheckRequest) (*grpc_edge_controller_go.CheckResult, derrors.Error) {
	// TODO: Verify clock sync
	// TODO: Handle plugin data
	log.Info().Str("assetID", request.AssetId).Msg("agent check")

	m.notifier.AgentAlive(request.AssetId)
	pending, err := m.provider.GetPendingOperations(request.AssetId, true)
	if err != nil{
		log.Error().Str("trace", err.DebugReport()).Msg("cannot retrieve pending operations for an agent")
		// In this case the error is not returned to the agent as it cannot do anything.
		return &grpc_edge_controller_go.CheckResult{}, nil
	}
	// Return empty message
	if len(pending) == 0{
		return &grpc_edge_controller_go.CheckResult{}, nil
	}
	// Transform the result into gRPC structures.
	result := make([]*grpc_inventory_manager_go.AgentOpRequest, 0, len(pending))
	for _ , p := range pending{
		result = append(result, p.ToGRPC())
	}
	return &grpc_edge_controller_go.CheckResult{
		PendingRequests:      result,
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
