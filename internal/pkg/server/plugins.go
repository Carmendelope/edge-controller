/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

// Start available plugins

import (
	"github.com/nalej/derrors"

	"github.com/nalej/service-net-agent/pkg/plugin"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func startRegisteredPlugins(config *viper.Viper) (derrors.Error) {
	for name, entry := range(plugin.ListPlugins()) {
		log.Info().Str("name", name.String()).Str("description", entry.Description).Msg("starting plugin")
		pluginConfig := config.Sub(name.String())
		derr := plugin.StartPlugin(name, pluginConfig)
		if derr != nil {
			plugin.StopAll()
			return derr
		}
	}

	return nil
}
