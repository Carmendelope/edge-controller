/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package eic

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/asset"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
	"github.com/nalej/edge-controller/internal/pkg/server/agent"
	"github.com/nalej/edge-controller/internal/pkg/server/config"
	"github.com/nalej/edge-controller/internal/pkg/server/helper"
	"github.com/nalej/grpc-common-go"
	"github.com/nalej/grpc-inventory-go"
	"github.com/nalej/grpc-inventory-manager-go"
	"github.com/nalej/grpc-monitoring-go"
	"github.com/nalej/grpc-utils/pkg/conversions"
	"github.com/rs/zerolog/log"
	"github.com/satori/go.uuid"
	"time"
)

const CanceledResponseInfo = "Canceled by the System. Agent Uninstalled"
const InstallResponseInfo  = "Agent Install"
const UninstallResponseInfo = "Agent Uninstall"

type Manager struct {
	config   config.Config
	provider asset.Provider

	metricStorageProvider metricstorage.Provider
	agentInstaller        *AgentInstaller
	notifier              *agent.Notifier
}

func NewManager(cfg config.Config, assetProvider asset.Provider, metricStorageProvider metricstorage.Provider, notifier *agent.Notifier) Manager {
	installer := NewAgentInstaller(cfg, notifier)
	return Manager{cfg, assetProvider, metricStorageProvider, installer, notifier}
}

// unlinkEC removes VPN Client and credentials file
func (m *Manager) unlinkEC() {

	vpnHelper, err := helper.NewJoinHelper(m.config.JoinTokenPath, m.config.EicApiPort)
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error creating helper")
	}

	err = vpnHelper.DeleteLocalVPN()
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error removing vpn account")
	}

	err = vpnHelper.RemoveCredentials()
	if err != nil {
		log.Warn().Str("error", conversions.ToDerror(err).DebugReport()).Msg("error deleting credentials")
	}

	m.notifier.StopNotifierLoop()

	log.Info().Str("EC", m.config.EdgeControllerId).Msg("unlinked")

	// TODO: send a deletevpnUser to vpn-server

}

// Unlink the receiving EIC.
func (m *Manager) Unlink() (*grpc_common_go.Success, error) {

	err := m.provider.Clear()
	if err != nil {
		log.Warn().Str("err", err.DebugReport()).Msg("error clearing database")
	}

	go m.unlinkEC()

	return &grpc_common_go.Success{}, nil
}

// TriggerAgentOperation registers the operation in the EIC so that the agent will be notified on the
// next connection.
func (m *Manager) TriggerAgentOperation(request *grpc_inventory_manager_go.AgentOpRequest) (*grpc_inventory_manager_go.AgentOpResponse, error) {

	log.Info().Interface("request", request).Msg("Triggering agent operation")

	// NP-1506. Limit agent operation pending queue on EC
	ops, err := m.provider.GetPendingOperations(request.AssetId, false)
	if err != nil {
		return nil, conversions.ToGRPCError(err)
	}
	if len(ops) > 0 {
		return nil, conversions.ToGRPCError(derrors.NewFailedPreconditionError("unable to queue this operation, there is already one"))
	}

	operation := entities.NewAgentOpRequestFromGRPC(request)

	// adds the operation
	err = m.provider.AddPendingOperation(*operation)
	if err != nil {
		return nil, conversions.ToDerror(err)
	}

	return &grpc_inventory_manager_go.AgentOpResponse{
		OrganizationId:   request.OrganizationId,
		EdgeControllerId: request.EdgeControllerId,
		AssetId:          request.AssetId,
		OperationId:      request.OperationId,
		Timestamp:        operation.Created,
		Status:           grpc_inventory_go.OpStatus_SCHEDULED,
		Info:             "",
	}, nil
}

// Configure changes specific configuration options of the Edge Controller
// and/or Edge Controller plugins
func (m *Manager) Configure(request *grpc_inventory_manager_go.ConfigureEICRequest) (*grpc_common_go.Success, error) {
	return nil, nil
}

// ListMetrics returns available metrics for a certain selection of assets
func (m *Manager) ListMetrics(selector *grpc_inventory_go.AssetSelector) (*grpc_monitoring_go.MetricsList, error) {
	// TODO: Potentially check if the Organization ID and Edge
	// Controller ID on the selector matches.

	metrics, derr := m.metricStorageProvider.ListMetrics(entities.NewTagSelectorFromGRPC(selector))
	if derr != nil {
		return nil, derr
	}

	metricsList := &grpc_monitoring_go.MetricsList{
		Metrics: metrics,
	}

	return metricsList, nil
}

// QueryMetrics retrieves the monitoring data of assets local to this
// Edge Controller
func (m *Manager) QueryMetrics(request *grpc_monitoring_go.QueryMetricsRequest) (*grpc_monitoring_go.QueryMetricsResult, error) {
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
	grpcResults := make(map[string]*grpc_monitoring_go.QueryMetricsResult_AssetMetrics, len(metrics))
	for _, metric := range metrics {
		metricValues, derr := m.metricStorageProvider.QueryMetric(metric, tagSelector, timeRange, aggrMethod)
		if derr != nil {
			return nil, derr
		}

		// Convert the values
		grpcValues := make([]*grpc_monitoring_go.QueryMetricsResult_Value, 0, len(metricValues))
		for _, value := range metricValues {
			grpcValues = append(grpcValues, value.ToGRPC())
		}

		grpcResult := &grpc_monitoring_go.QueryMetricsResult_AssetMetricValues{
			Values: grpcValues,
		}

		// Set the correct asset or aggregation
		assets := request.GetAssets().GetAssetIds()
		if len(assets) == 1 {
			grpcResult.AssetId = assets[0]
		} else {
			grpcResult.Aggregation = request.GetAggregation()
		}

		grpcResults[metric] = &grpc_monitoring_go.QueryMetricsResult_AssetMetrics{
			Metrics: []*grpc_monitoring_go.QueryMetricsResult_AssetMetricValues{
				grpcResult,
			},
		}
	}

	result := &grpc_monitoring_go.QueryMetricsResult{
		Metrics: grpcResults,
	}
	return result, nil
}

// CreateAgentJoinToken generates a JoinToken to allow an agent to join to a controller
func (m *Manager) CreateAgentJoinToken(edgeControllerID *grpc_inventory_go.EdgeControllerId) (*grpc_inventory_manager_go.AgentJoinToken, error) {
	token := uuid.NewV4().String()

	tokenInfo, err := m.provider.AddJoinToken(token)
	if err != nil {
		return nil, err
	}

	log.Info().Interface("token", token).Msg("agent join token added")

	return &grpc_inventory_manager_go.AgentJoinToken{
		OrganizationId:   edgeControllerID.OrganizationId,
		EdgeControllerId: edgeControllerID.EdgeControllerId,
		Token:            token,
		ExpiresOn:        tokenInfo.ExpiredOn,
	}, nil

}

// UninstallAgent operation to uninstall an agent
func (m *Manager) UninstallAgent(assetID *grpc_inventory_manager_go.FullUninstallAgentRequest) (*grpc_inventory_manager_go.EdgeControllerOpResponse, error) {

	// TODO: check what happens if a 'forced uninstall' message is received and before the token is deleted the agent connects
	// send the message to the notifier
	operationID := uuid.NewV4().String()
	m.notifier.UninstallAgent(assetID, operationID)

	//  remove pending operations and send them as cancelled to the IM.
	pending, err := m.provider.GetPendingOperations(assetID.AssetId, true)
	if err != nil {
		log.Error().Str("trace", err.DebugReport()).Msg("cannot retrieve pending operations for an agent uninstalling agent")
		// In this case the error is not returned to the agent as it cannot do anything.
		return nil, err
	}

	for _, operation := range pending {
		err = m.provider.AddOpResponse(entities.AgentOpResponse{
			Created:          time.Now().Unix(),
			OrganizationId:   assetID.OrganizationId,
			EdgeControllerId: assetID.EdgeControllerId,
			AssetId:          assetID.AssetId,
			OperationId:      operation.OperationId,
			Timestamp:        time.Now().Unix(),
			Status:           grpc_inventory_go.OpStatus_CANCELED.String(),
			Info:             CanceledResponseInfo,
		})
		if err != nil {
			log.Error().Str("trace", err.DebugReport()).Str("edge_controller_id", operation.EdgeControllerId).
				Str("asset_id", operation.AssetId).Str("operation_id", operation.OperationId).Msg("cannot add canceled operation")
		}
	}

	// TODO: maybe it would be better to erase the asset first of all

	// if the uninstalling is forced, the agent is deleted directly,
	// the edge controller does not wait for it to send a message
	if assetID.Force{
		err := m.provider.RemoveManagedAsset(assetID.AssetId)
		if err != nil {
			log.Warn().Str("assetID", assetID.AssetId).Str("trace", err.DebugReport()).Msg("error removing agent")
		}
	}

	return &grpc_inventory_manager_go.EdgeControllerOpResponse{
		OrganizationId: assetID.OrganizationId,
		EdgeControllerId: assetID.EdgeControllerId,
		OperationId: operationID,
		Status: grpc_inventory_go.OpStatus_SCHEDULED,
		Timestamp: time.Now().Unix(),
		Info: UninstallResponseInfo,

	}, nil
}

func (m *Manager) InstallAgent(request *grpc_inventory_manager_go.InstallAgentRequest) (*grpc_inventory_manager_go.EdgeControllerOpResponse, error) {
	// Prepare the data to trigger the async install
	opID := uuid.NewV4().String()
	token := uuid.NewV4().String()
	tokenInfo, err := m.provider.AddJoinToken(token)
	if err != nil {
		return nil, err
	}
	go m.agentInstaller.InstallAgent(opID, tokenInfo.Token, request)
	response := &grpc_inventory_manager_go.EdgeControllerOpResponse{
		OrganizationId:		request.OrganizationId,
		EdgeControllerId:	request.EdgeControllerId,
		OperationId:      	opID,
		Status:           	grpc_inventory_go.OpStatus_INPROGRESS,
		Timestamp: 		 	time.Now().Unix(),
		Info: 				InstallResponseInfo,
	}
	return response, nil
}
