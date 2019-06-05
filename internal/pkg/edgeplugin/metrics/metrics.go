/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metrics

// Edge Controller metrics storage plugin

import (
	"github.com/nalej/derrors"

	plugin "github.com/nalej/infra-net-plugin"
	"github.com/nalej/edge-controller/internal/pkg/edgeplugin"
	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
	"github.com/nalej/grpc-edge-controller-go"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

var metricsDescriptor = plugin.PluginDescriptor{
        Name: "metrics",
        Description: "System metrics storage plugin",
        NewFunc: NewMetrics,
}

type Metrics struct {
        edgeplugin.BaseEdgePlugin

	// Storage provider
	provider metricstorage.Provider
}

func init() {
	metricsDescriptor.AddFlag(plugin.FlagDescriptor{
		Name: "influxdb.address",
		Description: "InfluxDB address",
		Default: "http://localhost:8086",
	})
	metricsDescriptor.AddFlag(plugin.FlagDescriptor{
		Name: "influxdb.database",
		Description: "InfluxDB database name",
		Default: "metrics",
	})
        plugin.Register(&metricsDescriptor)
}

func NewMetrics(config *viper.Viper) (plugin.Plugin, derrors.Error) {
	// Validate configuration
	connConfig, derr := metricstorage.NewConnectionConfig(config)
	if derr != nil {
		return nil, derr
	}

	// Create storage provider based on config. For now, just InfluxDB
	provider, derr := metricstorage.NewProvider(connConfig)
	if derr != nil {
		return nil, derr
	}

	m := &Metrics{
		provider: provider,
	}

	return m, nil
}

func (m *Metrics) GetPluginDescriptor() (*plugin.PluginDescriptor) {
	return &metricsDescriptor
}

func (m *Metrics) StartPlugin() (derrors.Error) {
	// Open database connection and create database and table if necessary
	derr := m.provider.Connect()
	if derr != nil {
		return derr
	}

	derr = m.provider.CreateSchema(true)
	if derr != nil {
		return derr
	}

        return nil
}

func (m *Metrics) StopPlugin() {
	// Close database connection
	m.provider.Disconnect()
}

func (m *Metrics) HandleAgentData(data *grpc_edge_controller_go.PluginData) (derrors.Error) {
	log.Debug().Msg("metrics data received")
	// Check if started
	if !m.provider.Connected() {
		return derrors.NewUnavailableError("metrics plugin not started")
	}

	// Retrieve, type cast, check and convert
	metrics, derr := entities.NewMetricsDataFromGRPC(data)
	if derr != nil {
		return derr
	}

	// Store metrics
	derr = m.provider.StoreMetricsData(metrics)
	if derr != nil {
		return derr
	}

	return nil
}
