/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package test

// Dummy Metric Storage provider for testing

import (
	"time"

	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
)

const TestProviderType metricstorage.ProviderType = "test"

type TestProvider struct {
	IsConnected bool
	SchemaCreated bool

	Retention time.Duration
	Database string
	Address string

	LastMetrics *entities.MetricsData
	LastTags map[string]string
}

func init() {
	metricstorage.Register(TestProviderType, NewTestProvider)
}

func NewTestProvider(conf *metricstorage.ConnectionConfig) (metricstorage.Provider, derrors.Error) {
	t := &TestProvider{
		Database: conf.Database,
		Address: conf.Address,
	}

	return t, nil
}

func (t *TestProvider) Connect() derrors.Error {
	if t.IsConnected {
		return derrors.NewInternalError("already connected")
	}
	t.IsConnected = true
	return nil
}

func (t *TestProvider) Disconnect() derrors.Error {
	if !t.IsConnected {
		return derrors.NewInternalError("not connected")
	}
	t.IsConnected = false
	return nil
}

func (t *TestProvider) Connected() bool {
	return t.IsConnected
}

// "Creates schema" the first time it's called
func (t *TestProvider) CreateSchema(ifNeeded bool) derrors.Error {
	if t.SchemaCreated && !ifNeeded {
		return derrors.NewInvalidArgumentError("schema already created")
	}
	t.SchemaCreated = true
	return nil
}

// Accepts but only stores last
func (t *TestProvider) StoreMetricsData(metrics *entities.MetricsData, extraTags map[string]string) derrors.Error {
	t.LastMetrics = metrics
	t.LastTags = extraTags
	return nil
}

// Returns static answers
func (t *TestProvider) ListMetrics(tagSelector entities.TagSelector) ([]string, derrors.Error) {
	return []string{}, nil
}

// answers with only last values, ignoring timerange and tag for now
func (t *TestProvider) QueryMetric(metric string, tagSelector entities.TagSelector, timeRange *entities.TimeRange, aggr entities.AggregationMethod) ([]entities.MetricValue, derrors.Error) {
	values := []entities.MetricValue{}
	for _, m := range(t.LastMetrics.Metrics) {
		if m.Name != metric {
			continue
		}

		for _, f := range(m.Fields) {
			v := entities.MetricValue{
				Timestamp: t.LastMetrics.Timestamp,
				Value: int64(f),
			}
			values = append(values, v)
		}
	}
	return values, nil
}

func (t *TestProvider) SetRetention(dur time.Duration) (derrors.Error) {
	t.Retention = dur
	return nil
}
