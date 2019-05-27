/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package agent

import (
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"golang.org/x/net/context"
)

// Handler structure for the cluster requests.
type Handler struct {
	Manager Manager
}

// NewHandler creates a new Handler with a linked manager.
func NewHandler(manager Manager) *Handler{
	return &Handler{manager}
}

func (h *Handler) AgentJoin(ctx context.Context, request *grpc_edge_controller_go.AgentJoinRequest) (*grpc_inventory_manager_go.AgentJoinResponse, error) {
	err := entities.ValidJoinRequest(request)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	response, err := h.Manager.AgentJoin(request)
	if err != nil{
		log.Warn().Str("trace", err.DebugReport()).Msg("agent join failed")
		return nil, conversions.ToGRPCError(err)
	}
	log.Debug().Str("assetID", response.AssetId).Msg("agent joined successfully")
	return response, nil
}

func (h *Handler) AgentStart(ctx context.Context, info *grpc_inventory_manager_go.AgentStartInfo) (*grpc_common_go.Success, error) {
	err := entities.ValidAgentStartInfo(info)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	err = h.Manager.AgentStart(info)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return &grpc_common_go.Success{}, nil
}

func (h *Handler) AgentCheck(ctx context.Context, request *grpc_edge_controller_go.AgentCheckRequest) (*grpc_edge_controller_go.CheckResult, error) {
	err := entities.ValidAgentCheckRequest(request)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	result, err := h.Manager.AgentCheck(request)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return result, nil
}

func (h *Handler) CallbackAgentOperation(ctx context.Context, response *grpc_inventory_manager_go.AgentOpResponse) (*grpc_common_go.Success, error) {
	err := entities.ValidAgentOpResponse(response)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	err = h.Manager.CallbackAgentOperation(response)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	return &grpc_common_go.Success{}, nil
}

func (h *Handler) CreateAgentJoinToken(ctx context.Context, edge *grpc_inventory_go.EdgeControllerId) (*grpc_inventory_manager_go.AgentJoinToken, error) {
	err := entities.ValidEdgeControllerID(edge)
	if err != nil{
		return nil, conversions.ToGRPCError(err)
	}
	token, err := h.Manager.CreateAgentJoinToken(edge)
	if err != nil {
		return nil, conversions.ToGRPCError(err)
	}

	return token, nil
}