/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package edgeplugin

// Base plugin that can be embedded in other plugins to adhere to the interface

import (
	"github.com/nalej/derrors"
	"github.com/nalej/grpc-edge-controller-go"
	plugin "github.com/nalej/infra-net-plugin"
)

type BaseEdgePlugin struct {
        plugin.BasePlugin
}

// A plugin can embed BaseEdgePlugin and only has to define GetPluginDescriptor
// func (b *BasePlugin) GetPluginDescriptor() *PluginDescriptor

func (b *BaseEdgePlugin) HandleAgentData(assetId string, data *grpc_edge_controller_go.PluginData) (derrors.Error) {
	return derrors.NewUnimplementedError("plugin data handler not implemented")
}
