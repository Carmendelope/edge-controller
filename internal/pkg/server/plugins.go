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

package server

// Start available plugins

import (
	"github.com/nalej/derrors"

	plugin "github.com/nalej/infra-net-plugin"

	// Available data plugins
	_ "github.com/nalej/edge-controller/internal/pkg/edgeplugin/metrics"

	// Available metric storage providers
	_ "github.com/nalej/edge-controller/internal/pkg/provider/metricstorage/influxdb"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

// Workaround for https://github.com/spf13/viper/issues/368
// We can't use Sub on nested options that are bound to a flag, so
// we re-create the configuration and then retrieve the correct
// sub-config.
func getSubConfig(config *viper.Viper, prefix string) *viper.Viper {
	fixedConf := viper.New()
	for _, k := range(config.AllKeys()) {
		fixedConf.Set(k, config.Get(k))
	}

	// Now that the nesting is correct, get the sub-config, if needed
	if prefix != "" {
		fixedConf = fixedConf.Sub(prefix)
	}

	// Make sure we don't return a nil config
	if fixedConf == nil {
		fixedConf = viper.New()
	}

	return fixedConf
}

func startRegisteredPlugins(config *viper.Viper) (derrors.Error) {
	for name, entry := range(plugin.ListPlugins()) {
		log.Info().Str("name", name.String()).Str("description", entry.Description).Msg("starting plugin")
		pluginConfig := config.Sub(name.String())
		if pluginConfig == nil {
			pluginConfig = viper.New()
		}
		derr := plugin.StartPlugin(name, pluginConfig)
		if derr != nil {
			plugin.StopAll()
			return derr
		}
	}

	return nil
}
