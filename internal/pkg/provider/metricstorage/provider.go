/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metricstorage

// Metric storage provider interfaces and creation

import (
	"time"

	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"

	"github.com/rs/zerolog/log"
)

// Provider interface to store and retrieve metrics data. Before any storager
// or retrieval actions can take place, a connection has to be made
type Provider interface {
	// Create a connection to the storage system. All relevant information
	// should be passed when creating the provider instance
	Connect() derrors.Error
	// Disconnect from the storage system
	Disconnect() derrors.Error
	// Check if there is a connection
	Connected() bool

	// Create the schema needed to store metrics data. Returns an error if
	// any of the entities already exist, unless `ifNeeded` is set.
	CreateSchema(ifNeeded bool) derrors.Error

	// Store metrics
	StoreMetricsData(data *entities.MetricsData, extraTags map[string]string) derrors.Error

	// List available metrics. If tagSelector is empty, return all available,
	// if tagSelector contains key-value pairs, return metrics available
	// for the union of those tags
	ListMetrics(tagSelector map[string][]string) ([]string, derrors.Error)

	// Query specific metric. If tagSelector is empty, return all values
	// available, aggregated with aggr. If tagSelector is contains
	// key-value pairs, return values for the union of those tags,
	// aggregated with aggr. If tagSelector contains a single entry,
	// values for that specific tag are returned and aggr is ignored.
	QueryMetric(metric string, tagSelector map[string][]string, timeRange TimeRange, aggr AggregationMethod) ([]Value, derrors.Error)
}

type Value struct {
	Timestamp time.Time
	Value int64
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

type AggregationMethod string
const (
	AggregateNone AggregationMethod = "none"
	AggregateSum AggregationMethod = "_sum"
	AggregateAvg AggregationMethod = "_avg"
)

type ProviderType string
func (t ProviderType) String() string {
	return string(t)
}

type ProviderNewFunc func(*ConnectionConfig) (Provider, derrors.Error)

var providers = map[ProviderType]ProviderNewFunc{}

func Register(t ProviderType, f ProviderNewFunc) {
	log.Debug().Str("type", t.String()).Msg("registering metricstorage provider")
	providers[t] = f
}

// Depending on the configuration, create the right provider instance
func NewProvider(conf *ConnectionConfig) (Provider, derrors.Error) {
	f, found := providers[conf.providerType]
	if !found {
		return nil, derrors.NewInvalidArgumentError("provider not available").WithParams(conf.providerType)
	}

	return f(conf)
}
