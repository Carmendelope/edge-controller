/*
 * Copyright (C)  2019 Nalej - All Rights Reserved
 */

package entities

// Metrics data for storage
// cf. github.com/nalej/service-net-agent/internal/pkg/agentplugin/metrics/metricsdata.go

import (
	"time"

	"github.com/nalej/derrors"

	"github.com/nalej/grpc-edge-controller-go"
)

type MetricsData struct {
	Timestamp time.Time
	Metrics []*Metric
}

type Metric struct {
	Name string
	Tags map[string]string
	Fields map[string]uint64
}

func NewMetricsDataFromGRPC(data *grpc_edge_controller_go.PluginData) (*MetricsData, derrors.Error) {
	// Type cast and check
	grpcMetricsData := data.GetMetricsData()
	if grpcMetricsData == nil {
		return nil, derrors.NewInvalidArgumentError("invalid plugin data for metrics")
	}

	grpcMetrics := grpcMetricsData.GetMetrics()
	metrics := make([]*Metric, 0, len(grpcMetrics))
	for _, grpcMetric := range(grpcMetrics) {
		metric := &Metric{
			Name: grpcMetric.GetName(),
			Tags: grpcMetric.GetTags(),
			Fields: grpcMetric.GetFields(),
		}
		metrics = append(metrics, metric)
	}

	metricsData := &MetricsData{
		Timestamp: time.Unix(grpcMetricsData.GetTimestamp(), 0).UTC(),
		Metrics: metrics,
	}

	return metricsData, nil
}
