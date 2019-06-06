/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/rs/zerolog/log"
	"github.com/satori/go.uuid"
)

type Manager struct{
	config config.Config
	provider asset.Provider

	metricStorageProvider metricstorage.Provider
}

func NewManager(cfg config.Config, assetProvider asset.Provider, metricStorageProvider metricstorage.Provider) Manager{
	return Manager{cfg, assetProvider, metricStorageProvider}
}

// Unlink the receiving EIC.
func (m * Manager)Unlink() (*grpc_common_go.Success, error) {
	return nil, nil
}
// TriggerAgentOperation registers the operation in the EIC so that the agent will be notified on the
// next connection.
func (m * Manager)TriggerAgentOperation(request *grpc_inventory_manager_go.AgentOpRequest) (*grpc_inventory_manager_go.AgentOpResponse, error){
	return nil, nil
}
// Configure changes specific configuration options of the Edge Controller
// and/or Edge Controller plugins
func (m * Manager)Configure(request *grpc_inventory_manager_go.ConfigureEICRequest) (*grpc_common_go.Success, error) {
	return nil, nil
}
// ListMetrics returns available metrics for a certain selection of assets
func (m * Manager)ListMetrics(selector *grpc_inventory_manager_go.AssetSelector) (*grpc_inventory_manager_go.MetricsList, error) {
	// TODO: Potentially check if the Organization ID and Edge
	// Controller ID on the selector matches.

	// Create tag selector from assets
	var tagSelector map[string][]string = nil
	assets := selector.GetAssetIds()
	if len(assets) > 0 {
		tagSelector = map[string][]string{
			"asset_id": assets,
		}
	}

	metrics, derr := m.metricStorageProvider.ListMetrics(tagSelector)
	if derr != nil {
		return nil, derr
	}

	metricsList := &grpc_inventory_manager_go.MetricsList{
		Metrics: metrics,
	}

	return metricsList, nil
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
