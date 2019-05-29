/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package edgeplugin

// Edge Controller plugin infrastructure

import (
	"github.com/nalej/derrors"
	"github.com/nalej/grpc-edge-controller-go"
	"github.com/nalej/service-net-agent/pkg/plugin"
)

type EdgePlugin interface {
	plugin.Plugin

	// Handle plugin-specific data received from Agent
	HandleAgentData(data *grpc_edge_controller_go.PluginData) (derrors.Error)
}
