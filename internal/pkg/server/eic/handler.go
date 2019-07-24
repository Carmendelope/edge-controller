/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"context"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
)

// Handler structure for the cluster requests.
type Handler struct {
	Manager Manager
}

// NewHandler creates a new Handler with a linked manager.
func NewHandler(manager Manager) *Handler{
	return &Handler{manager}
}

// Unlink the receiving EIC.
func (h *Handler)Unlink(_ context.Context, in *grpc_common_go.Empty) (*grpc_common_go.Success, error) {
	log.Info().Msg("unlink message received")
	return h.Manager.Unlink()
}
// TriggerAgentOperation registers the operation in the EIC so that the agent will be notified on the
// next connection.
func (h *Handler)TriggerAgentOperation(_ context.Context, request *grpc_inventory_manager_go.AgentOpRequest) (*grpc_inventory_manager_go.AgentOpResponse, error){

	vErr := entities.ValidAgentOpRequest(request)
	if vErr != nil {
		return nil, conversions.ToGRPCError(vErr)
	}
	return h.Manager.TriggerAgentOperation(request)

}
// Configure changes specific configuration options of the Edge Controller
// and/or Edge Controller plugins
func (h *Handler)Configure(_ context.Context, request *grpc_inventory_manager_go.ConfigureEICRequest) (*grpc_common_go.Success, error) {
	return nil, nil
}
// ListMetrics returns available metrics for a certain selection of assets
func (h *Handler)ListMetrics(_ context.Context, selector *grpc_inventory_manager_go.AssetSelector) (*grpc_inventory_manager_go.MetricsList, error) {
	log.Debug().Interface("selector", selector).Msg("listing available metrics")
	derr := entities.ValidAssetSelector(selector)
	if derr != nil {
		return nil, conversions.ToGRPCError(derr)
	}

	return h.Manager.ListMetrics(selector)
}
// QueryMetrics retrieves the monitoring data of assets local to this
// Edge Controller
func (h *Handler)QueryMetrics(_ context.Context, request *grpc_inventory_manager_go.QueryMetricsRequest) (*grpc_inventory_manager_go.QueryMetricsResult, error){
	log.Debug().Interface("request", request).Msg("executing metrics query")
	derr := entities.ValidQueryMetricsRequest(request)
	if derr != nil {
		return nil, conversions.ToGRPCError(derr)
	}

	return h.Manager.QueryMetrics(request)
}
// CreateAgentJoinToken generates a JoinToken to allow an agent to join to a controller
func (h *Handler)CreateAgentJoinToken(_ context.Context, edgeControllerID *grpc_inventory_go.EdgeControllerId) (*grpc_inventory_manager_go.AgentJoinToken, error){

	log.Debug().Interface("edgeControllerID", edgeControllerID).Msg("creating agent join token")
	vErr := entities.ValidEdgeControllerID(edgeControllerID)
	if vErr != nil {
		return nil, conversions.ToGRPCError(vErr)
	}

	return h.Manager.CreateAgentJoinToken(edgeControllerID)

}

// UninstallAgent operation to uninstall an agent
func (h *Handler) UninstallAgent(_ context.Context, request *grpc_inventory_manager_go.FullUninstallAgentRequest) (*grpc_inventory_manager_go.EdgeControllerOpResponse, error) {
	log.Debug().Interface("edgeControllerID", request.EdgeControllerId).Str("assetID", request.AssetId).Bool("force", request.Force).Msg("uninstall agent")

	vErr := entities.ValidFullUninstallAgentRequest(request)
	if vErr != nil {
		return nil, conversions.ToGRPCError(vErr)
	}

	return h.Manager.UninstallAgent(request)
}

// InstallAgent triggers the installation of an agent.
func (h *Handler) InstallAgent(ctx context.Context, request *grpc_inventory_manager_go.InstallAgentRequest) (*grpc_inventory_manager_go.EdgeControllerOpResponse, error){
	vErr := entities.ValidInstallAgentRequest(request)
	if vErr != nil {
		return nil, conversions.ToGRPCError(vErr)
	}
	return h.Manager.InstallAgent(request)
}