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
	"github.com/nalej/grpc-inventory-manager-go"
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
		if metric.Tags == nil {
			metric.Tags = map[string]string{}
		}

		metrics = append(metrics, metric)
	}

	metricsData := &MetricsData{
		Timestamp: time.Unix(grpcMetricsData.GetTimestamp(), 0).UTC(),
		Metrics: metrics,
	}

	return metricsData, nil
}

type MetricValue struct {
	Timestamp time.Time
	Value int64
}

func (m *MetricValue) ToGRPC() *grpc_inventory_manager_go.QueryMetricsResult_Value {
	return &grpc_inventory_manager_go.QueryMetricsResult_Value{
		Timestamp: m.Timestamp.UTC().Unix(),
		Value: m.Value,
	}
}

type TimeRange struct {
	// Either Timestamp != 0 && (Start == End == Resolution == 0),
	// or Timestamp == 0 && (Start != 0 || End != 0)

	// Timestamp is set to request single data point
	Timestamp time.Time

	// Start indicates the start of the time range;
	// Start == 0 means starting from oldest data available
	Start time.Time

	// End indicates the end of the time range;
	// End == 0 means ending at newest data available
	End time.Time

	// Resolution indicates the duration between returned data points;
        // If Resolution == 0, return a single, aggregated (avg) data point
	Resolution time.Duration
}

func NewTimeRangeFromGRPC(timeRange *grpc_inventory_manager_go.QueryMetricsRequest_TimeRange) *TimeRange {
	return &TimeRange{
		Timestamp: time.Unix(timeRange.GetTimestamp(), 0).UTC(),
		Start: time.Unix(timeRange.GetTimeStart(), 0).UTC(),
		End: time.Unix(timeRange.GetTimeEnd(), 0).UTC(),
		Resolution: time.Duration(timeRange.GetResolution()) * time.Second,
	}
}

func ValidTimeRange(timeRange *grpc_inventory_manager_go.QueryMetricsRequest_TimeRange) derrors.Error {
	if !(timeRange.GetTimestamp() == 0) {
		if timeRange.GetTimeStart() != 0 || timeRange.GetTimeEnd() != 0 || timeRange.GetResolution() != 0 {
			return derrors.NewInvalidArgumentError("timestamp is set; start, end and resolution should be 0").
				WithParams(timeRange.GetTimestamp(), timeRange.GetTimeStart(),
				timeRange.GetTimeEnd(), timeRange.GetResolution())
		}
	} else {
		if timeRange.GetTimeStart() == 0 && timeRange.GetTimeEnd() == 0 {
			return derrors.NewInvalidArgumentError("timestamp is not set; either start, end or both should be set").
				WithParams(timeRange.GetTimestamp(), timeRange.GetTimeStart(),
				timeRange.GetTimeEnd(), timeRange.GetResolution())
		}
	}

	return nil
}

type AggregationMethod string
func (a AggregationMethod) String() string {
	return string(a)
}

const (
	AggregateNone AggregationMethod = "none"
	AggregateSum AggregationMethod = "_sum"
	AggregateAvg AggregationMethod = "_avg"
)

func AggregationMethodFromGRPC(method grpc_inventory_manager_go.QueryMetricsRequest_AggregationType) AggregationMethod {
	var aggrMap = map[grpc_inventory_manager_go.QueryMetricsRequest_AggregationType]AggregationMethod{
		grpc_inventory_manager_go.QueryMetricsRequest_NONE: AggregateNone,
		grpc_inventory_manager_go.QueryMetricsRequest_SUM: AggregateSum,
		grpc_inventory_manager_go.QueryMetricsRequest_AVG: AggregateAvg,
	}

	return aggrMap[method]
}

func ValidQueryMetricsRequest(request *grpc_inventory_manager_go.QueryMetricsRequest) derrors.Error {
	derr := ValidAssetSelector(request.GetAssets())
	if derr != nil {
		return derr
	}

	derr = ValidTimeRange(request.GetTimeRange())
	if derr != nil {
		return derr
	}

	if len(request.GetAssets().GetAssetIds()) != 1 && request.GetAggregation() != grpc_inventory_manager_go.QueryMetricsRequest_NONE {
		return derrors.NewInvalidArgumentError("metrics for more than one asset requested without aggregation method")
	}

	return nil
}
