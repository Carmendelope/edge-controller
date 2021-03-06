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

package edgeplugin


// Edge Controller plugin registry

import (
	"strings"

	"github.com/nalej/derrors"

	"github.com/nalej/grpc-edge-controller-go"

	plugin "github.com/nalej/infra-net-plugin"
)

type EdgeRegistry struct {
	*plugin.Registry
}

// This contains the exact same registered plugins as the general default
// registry - we just add this to add some specific edge registry methods.
var defaultRegistry = NewEdgeRegistry(plugin.DefaultRegistry())

func NewEdgeRegistry(parent *plugin.Registry) *EdgeRegistry {
        r := &EdgeRegistry{parent}

        return r
}

func (r *EdgeRegistry) HandleAgentData(assetId string, data *grpc_edge_controller_go.PluginData) (derrors.Error) {
	// Check protocol consistency and get plugin name
	name, found := grpc_edge_controller_go.Plugin_name[int32(data.GetPlugin())]
	if !found {
		return derrors.NewInvalidArgumentError("plugin not found in mapping - likely inconsistent gRPC protocol versions").WithParams(data.GetPlugin())
	}

	// Check if plugin is availble and running - get instance
	p, derr := plugin.GetPlugin(plugin.PluginName(strings.ToLower(name)))
	if derr != nil {
		return derr
	}

	// Check if this is an edge plugin
	ep, ok := p.(EdgePlugin)
	if !ok {
		return derrors.NewInvalidArgumentError("data received for non-edge plugin").WithParams(name)
	}

	// Hand off to plugin - plugin will do type casting
	return ep.HandleAgentData(assetId, data)
}

func HandleAgentData(assetId string, data *grpc_edge_controller_go.PluginData) (derrors.Error) {
	return defaultRegistry.HandleAgentData(assetId, data)
}
