/*
 * Copyright 2019 Nalej
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
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
	ListMetrics(tagSelector entities.TagSelector) ([]string, derrors.Error)

	// Query specific metric. If tagSelector is empty, return all values
	// available, aggregated with aggr. If tagSelector is contains
	// key-value pairs, return values for the union of those tags,
	// aggregated with aggr. If tagSelector contains a single entry,
	// values for that specific tag are returned and aggr is ignored.
	QueryMetric(metric string, tagSelector entities.TagSelector, timeRange *entities.TimeRange, aggr entities.AggregationMethod) ([]entities.MetricValue, derrors.Error)

	// Set retention policy. For now, we just set one single expiration
	// duration after which data gets deleted.
	SetRetention(dur time.Duration) (derrors.Error)
}

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
