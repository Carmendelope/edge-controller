/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/edge-controller/internal/pkg/server/helper"
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

	metricStorageProvider metricstorage.Provider
}

func NewManager(cfg config.Config, assetProvider asset.Provider, metricStorageProvider metricstorage.Provider) Manager{
	return Manager{cfg, assetProvider, metricStorageProvider}
}

func (m *Manager) deleteVPNAccount() {

	vpnHelper, err := helper.NewJoinHelper(m.config.JoinTokenPath, m.config.EicApiPort)
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error creating helper")
	}

	err = vpnHelper.DeleteLocalVPN()
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error removing vpn account")
	}

	err  = vpnHelper.RemoveCredentials()
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error deleting credentials")
	}

	// TODO: send a deletevpnUser to vpn-server

}

// Unlink the receiving EIC.
func (m * Manager)Unlink() (*grpc_common_go.Success, error) {

	go m.deleteVPNAccount()

	return &grpc_common_go.Success{}, nil
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
	// TODO: Potentially check if the Organization ID and Edge
	// Controller ID on the selector matches.

	metrics, derr := m.metricStorageProvider.ListMetrics(entities.NewTagSelectorFromGRPC(selector))
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
	tagSelector := entities.NewTagSelectorFromGRPC(request.GetAssets())
	timeRange := entities.NewTimeRangeFromGRPC(request.GetTimeRange())
	aggrMethod := entities.AggregationMethodFromGRPC(request.GetAggregation())

	metrics := request.GetMetrics()

	// If no metrics are requested, return all
	if len(metrics) == 0 {
		allMetrics, err := m.ListMetrics(request.GetAssets())
		if err != nil {
			return nil, err
		}
		metrics = allMetrics.GetMetrics()
	}

	// Create result for this asset or aggreagation of assets, for each metric
	grpcResults := make(map[string]*grpc_inventory_manager_go.QueryMetricsResult_AssetMetrics, len(metrics))
	for _, metric := range(metrics) {
		metricValues, derr := m.metricStorageProvider.QueryMetric(metric, tagSelector, timeRange, aggrMethod)
		if derr != nil {
			return nil, derr
		}

		// Convert the values
		grpcValues := make([]*grpc_inventory_manager_go.QueryMetricsResult_Value, 0, len(metricValues))
		for _, value := range(metricValues) {
			grpcValues = append(grpcValues, value.ToGRPC())
		}

		grpcResult := &grpc_inventory_manager_go.QueryMetricsResult_AssetMetricValues{
			Values: grpcValues,
		}

		// Set the correct asset or aggregation
		assets := request.GetAssets().GetAssetIds()
		if len(assets) == 1 {
			grpcResult.AssetId = assets[0]
		} else {
			grpcResult.AssetId = aggrMethod.String()
		}

		grpcResults[metric] = &grpc_inventory_manager_go.QueryMetricsResult_AssetMetrics{
			Metrics: []*grpc_inventory_manager_go.QueryMetricsResult_AssetMetricValues{
				grpcResult,
			},
		}
	}

	result := &grpc_inventory_manager_go.QueryMetricsResult{
		Metrics: grpcResults,
	}
	return result, nil
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
