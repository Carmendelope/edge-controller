/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metrics

// Edge Controller metrics storage plugin

import (
	"time"

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

	// Retention duration - we store this because we'll get it during
	// initialization but need it during plugin start, when we create
	// schema and set retention policy. This way, if we restart the
	// edge controller with a different retention, we alter the already
	// existing database to reflect that.
	retention time.Duration
}

func init() {
	metricsDescriptor.AddFlag(plugin.FlagDescriptor{
		Name: "retention",
		Description: "Default metrics data retention duration",
		Default: "inf",
	})
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
		retention: connConfig.Retention,
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

	derr = m.provider.SetRetention(m.retention)
	if derr != nil {
		return derr
	}

        return nil
}

func (m *Metrics) StopPlugin() {
	// Close database connection
	m.provider.Disconnect()
}

func (m *Metrics) HandleAgentData(assetId string, data *grpc_edge_controller_go.PluginData) (derrors.Error) {
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

	// Extra tags
	tags := map[string]string{
		"asset_id": assetId,
	}

	// Store metrics
	derr = m.provider.StoreMetricsData(metrics, tags)
	if derr != nil {
		return derr
	}

	return nil
}
