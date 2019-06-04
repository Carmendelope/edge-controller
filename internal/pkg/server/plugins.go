/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

// Start available plugins

import (
	"github.com/nalej/derrors"

	"github.com/nalej/service-net-agent/pkg/plugin"

	// Available plugins
	_ "github.com/nalej/edge-controller/internal/pkg/edgeplugin/metrics"

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
