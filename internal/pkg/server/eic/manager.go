/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"github.com/satori/go.uuid"
)

type Manager struct{
	config config.Config
	provider asset.Provider
}

func NewManager(cfg config.Config, assetProvider asset.Provider,) Manager{
	return Manager{cfg, assetProvider}
}

// Unlink the receiving EIC.
func (m * Manager)Unlink() (*grpc_common_go.Success, error) {
	return nil, nil
}
// TriggerAgentOperation registers the operation in the EIC so that the agent will be notified on the
// next connection.
func (m * Manager)TriggerAgentOperation(request *grpc_inventory_manager_go.AgentOpRequest) (*grpc_inventory_manager_go.AgentOpResponse, error){

	log.Info().Interface("request", request).Msg("Triggering agent operation")

	operation := entities.NewAgentOpRequestFromGRPC(request)

	// adds the operation
	err := m.provider.AddPendingOperation(*operation)
	if err != nil {
		return nil, conversions.ToDerror(err)
	}

	return &grpc_inventory_manager_go.AgentOpResponse{
		OrganizationId:		request.OrganizationId,
		EdgeControllerId:	request.EdgeControllerId,
		AssetId:			request.AssetId,
		OperationId: 		request.OperationId,
		Timestamp:			operation.Created,
		Status:				grpc_inventory_manager_go.AgentOpStatus_SCHEDULED,
		Info: "",
	}, nil
}
// Configure changes specific configuration options of the Edge Controller
// and/or Edge Controller plugins
func (m * Manager)Configure(request *grpc_inventory_manager_go.ConfigureEICRequest) (*grpc_common_go.Success, error) {
	return nil, nil
}
// ListMetrics returns available metrics for a certain selection of assets
func (m * Manager)ListMetrics(selector *grpc_inventory_manager_go.AssetSelector) (*grpc_inventory_manager_go.MetricsList, error) {
	return nil, nil
}
// QueryMetrics retrieves the monitoring data of assets local to this
// Edge Controller
func (m * Manager)QueryMetrics(request *grpc_inventory_manager_go.QueryMetricsRequest) (*grpc_inventory_manager_go.QueryMetricsResult, error){
	return nil, nil
}
// CreateAgentJoinToken generates a JoinToken to allow an agent to join to a controller
func (m * Manager)CreateAgentJoinToken(edgeControllerID *grpc_inventory_go.EdgeControllerId) (*grpc_inventory_manager_go.AgentJoinToken, error){
	token := uuid.NewV4().String()

	tokenInfo, err := m.provider.AddJoinToken(token)
	if err != nil {
		return nil, err
	}

	log.Info().Interface("token", token).Msg("agent join token added")

	return &grpc_inventory_manager_go.AgentJoinToken{
		OrganizationId: edgeControllerID.OrganizationId,
		EdgeControllerId: edgeControllerID.EdgeControllerId,
		Token: token,
		ExpiresOn: tokenInfo.ExpiredOn,
	}, nil

}
