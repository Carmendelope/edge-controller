/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metricstorage

// Metric storage provider interfaces and creation

import (
	"github.com/nalej/derrors"

	"github.com/nalej/edge-controller/internal/pkg/entities"
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
	StoreMetricsData(*entities.MetricsData) derrors.Error

}

func NewProvider(conf *ConnectionConfig) (Provider, derrors.Error) {
	return nil, nil
}
