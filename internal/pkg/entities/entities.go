/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package entities

import (
	"github.com/nalej/derrors"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"time"
)

type AgentOpRequest struct {
	// Created with when this information was received
	Created int64 `json:"timestamp"`
	// OrganizationId with the organization identifier.
	OrganizationId string `json:"organization_id,omitempty"`
	// EdgeControllerId with the EIC identifier that will receive the operation.
	EdgeControllerId string `json:"edge_controller_id,omitempty"`
	// AssetId with the asset identifier.
	AssetId string `json:"asset_id,omitempty"`
	// Operation to be performed. No enum is used as to no constraint agent evolution.
	Operation string `json:"operation,omitempty"`
	// Plugin in charge of executing such operation.
	Plugin string `json:"plugin,omitempty"`
	// Params for the operation.
	Params map[string]string `json:"params,omitempty"`
}

func NewAgentOpRequestFromGRPC(request * grpc_inventory_manager_go.AgentOpRequest) * AgentOpRequest{
	return &AgentOpRequest{
		Created: time.Now().Unix(),
		OrganizationId:   request.OrganizationId,
		EdgeControllerId: request.EdgeControllerId,
		AssetId:          request.AssetId,
		Operation:        request.Operation,
		Plugin:           request.Plugin,
		Params:           request.Params,
	}
}

func (aor * AgentOpRequest) ToGRPC() *grpc_inventory_manager_go.AgentOpRequest{
	return &grpc_inventory_manager_go.AgentOpRequest{
		OrganizationId:       aor.OrganizationId,
		EdgeControllerId:     aor.EdgeControllerId,
		AssetId:              aor.AssetId,
		Operation:            aor.Operation,
		Plugin:               aor.Plugin,
		Params:               aor.Params,
	}
}

type AgentJoinInfo struct {
	// Created with when this information was received
	Created int64 `json:"timestamp"`
	// AssetId with the agent identifier as generated by the system model.
	AssetId string `json:"asset_id,omitempty"`
	// Token that the agent needs to send for further requests. The token should be added in the
	// authorization metadata of the gRPC context.
	Token                string   `json:"token,omitempty"`
}

func NewAgentJoinInfoFromGRPC(request * grpc_inventory_manager_go.AgentJoinResponse) * AgentJoinInfo{
	return &AgentJoinInfo{
		Created: time.Now().Unix(),
		AssetId: request.AssetId,
		Token:   request.Token,
	}
}

func ValidJoinRequest(request * grpc_edge_controller_go.AgentJoinRequest) derrors.Error{
	if request.AgentId == ""{
		return derrors.NewInvalidArgumentError("agent_id cannot be empty")
	}
	return nil
}

func ValidAgentStartInfo(info * grpc_inventory_manager_go.AgentStartInfo) derrors.Error{
	if info.AssetId == ""{
		return derrors.NewInvalidArgumentError("asset_id cannot be empty")
	}
	if info.Ip == ""{
		return derrors.NewInvalidArgumentError("ip cannot be empty")
	}
	return nil
}

func ValidAgentCheckRequest(request *grpc_edge_controller_go.AgentCheckRequest) derrors.Error{
	if request.AssetId == "" {
		return derrors.NewInvalidArgumentError("asset_id cannot be empty")
	}
	if request.Timestamp == 0 {
		return derrors.NewInvalidArgumentError("timestamp cannot be empty")
	}

	if len(request.PluginData) > 0 {
		for _, d := range(request.PluginData) {
			if d.Data == nil {
				return derrors.NewInvalidArgumentError("plugin data cannot be empty")
			}
		}
	}

	return nil
}

func ValidAgentOpResponse(response *grpc_inventory_manager_go.AgentOpResponse) derrors.Error{
	if response.AssetId == ""{
		return derrors.NewInvalidArgumentError("asset_id cannot be empty")
	}
	if response.OperationId == ""{
		return derrors.NewInvalidArgumentError("operation_id cannot be empty")
	}
	return nil
}

type AgentStartInfo struct{
	// Created with when this information was received
	Created int64 `json:"timestamp"`
	// AssetId with the asset identifier.
	AssetId string `json:"asset_id,omitempty"`
	// Ip that is visible from the EIC.
	Ip                   string   `json:"ip,omitempty"`
}

func NewAgentStartInfoFromGRPC(info * grpc_inventory_manager_go.AgentStartInfo) * AgentStartInfo{
	return &AgentStartInfo{
		Created: time.Now().Unix(),
		AssetId: info.AssetId,
		Ip:      info.Ip,
	}
}

func (asi * AgentStartInfo) ToGRPC() * grpc_inventory_manager_go.AgentStartInfo{
	return &grpc_inventory_manager_go.AgentStartInfo{
		AssetId:              asi.AssetId,
		Ip:                   asi.Ip,
	}
}

type AgentOpResponse struct{
	// Created with when this information was received
	Created int64 `json:"timestamp"`
	// OrganizationId with the organization identifier.
	OrganizationId string `json:"organization_id,omitempty"`
	// EdgeControllerId with the EIC identifier that facilitated the operation.
	EdgeControllerId string `json:"edge_controller_id,omitempty"`
	// AssetId with the asset identifier.
	AssetId string `json:"asset_id,omitempty"`
	// OperationId with the operation identifier.
	OperationId string `json:"operation_id,omitempty"`
	// Timestamp of the response.
	Timestamp int64 `json:"timestamp,omitempty"`
	// Status indicates if the operation was successfull
	Status string `json:"status,omitempty"`
	// Info with additional information for an operation.
	Info                 string   `json:"info,omitempty"`
}

func NewAgentOpResponseFromGRPC(response * grpc_inventory_manager_go.AgentOpResponse) * AgentOpResponse{
	return &AgentOpResponse{
		Created:          time.Now().Unix(),
		OrganizationId:   response.OrganizationId,
		EdgeControllerId: response.EdgeControllerId,
		AssetId:          response.AssetId,
		OperationId:      response.OperationId,
		Timestamp:        response.Timestamp,
		Status:           response.Status.String(),
		Info:             response.Info,
	}
}

func (aor * AgentOpResponse) ToGRPC() * grpc_inventory_manager_go.AgentOpResponse{
	return &grpc_inventory_manager_go.AgentOpResponse{
		OrganizationId:       aor.OrganizationId,
		EdgeControllerId:     aor.EdgeControllerId,
		AssetId:              aor.AssetId,
		OperationId:          aor.OperationId,
		Timestamp:            aor.Timestamp,
		Status:               grpc_inventory_manager_go.AgentOpStatus(grpc_inventory_manager_go.AgentOpStatus_value[aor.Status]),
		Info:                 aor.Info,
	}
}

type JoinToken struct{
	// Token that the agent needs to send for further requests.
	Token string   `json:"token,omitempty"`
	// ExpiredOn with information about when the token expires
	ExpiredOn int64 `json:"expired_on"`
	// OrganizationId with the organization identifier.
	// OrganizationId string `json:"organization_id,omitempty"`
	// EdgeControllerId with the EIC identifier that facilitated the operation.
	// EdgeControllerId string `json:"edge_controller_id,omitempty"`
}

func ValidEdgeControllerID(edge *grpc_inventory_go.EdgeControllerId) derrors.Error{
	if edge.OrganizationId == ""{
		return derrors.NewInvalidArgumentError("organization_id cannot be empty")
	}
	if edge.EdgeControllerId == ""{
		return derrors.NewInvalidArgumentError("edge_controller_id cannot be empty")
	}
	return nil
}

type TagSelector map[string][]string

func NewTagSelectorFromGRPC(selector *grpc_inventory_manager_go.AssetSelector) TagSelector {
	var tagSelector map[string][]string = nil
	assets := selector.GetAssetIds()
	if len(assets) > 0 {
		tagSelector = map[string][]string{
		"asset_id": assets,
		}
	}

	return tagSelector
}

func ValidAssetSelector(selector *grpc_inventory_manager_go.AssetSelector) derrors.Error {
	// Any selector is in theory valid. We could check organization id and
	// edge controller id, but that's easier done somewhere where we have
	// that info.

	// However, the edge controller likely doesn't know about user-defined
	// labels and group IDs, so the inventory manager has to translate from
	// those to asset IDs. To make sure that's done, we'll check if we don't
	// have those fields set anymore.

	if len(selector.GetGroupIds()) > 0 {
		return derrors.NewInvalidArgumentError("cannot select on group IDs at Edge Controller")
	}
	if len(selector.GetLabels()) > 0 {
		return derrors.NewInvalidArgumentError("cannot select on labels at Edge Controller")
	}

	return nil
}
