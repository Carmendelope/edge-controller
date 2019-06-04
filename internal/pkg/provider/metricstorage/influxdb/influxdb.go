/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package influxdb

// InfluxDB Metric Storage provider

import (
	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"
	"github.com/nalej/edge-controller/internal/pkg/provider/metricstorage"
)

const InfluxDBProviderType metricstorage.ProviderType = "influxdb"

type InfluxDBProvider struct {
}

func init() {
	metricstorage.Register(InfluxDBProviderType, NewInfluxDBProvider)
}

func NewInfluxDBProvider(conf *metricstorage.ConnectionConfig) (metricstorage.Provider, derrors.Error) {
	i := &InfluxDBProvider{

	}

	return i, nil
}

// Create a connection to the storage system. All relevant information
// should be passed when creating the provider instance
func (i *InfluxDBProvider) Connect() derrors.Error {
	return nil
}

// Disconnect from the storage system
func (i *InfluxDBProvider) Disconnect() derrors.Error {
	return nil
}

// Check if there is a connection
func (i *InfluxDBProvider) Connected() bool {
	return false
}

// Create the schema needed to store metrics data. Returns an error if
// any of the entities already exist, unless `ifNeeded` is set.
func (i *InfluxDBProvider) CreateSchema(ifNeeded bool) derrors.Error {
	return nil
}

// Store metrics
func (i *InfluxDBProvider) StoreMetricsData(*entities.MetricsData) derrors.Error {
	return nil
}
