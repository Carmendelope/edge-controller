/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package server

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/version"
	"github.com/rs/zerolog/log"
)

type Config struct {
	// Debug level is active.
	Debug bool
	// Port where the edge controller receives messages from the management cluster.
	Port int
	// Port where the edge controller receives messages from agents.
	AgentPort int
}

// Validate the current configuration.
func (conf * Config) Validate() derrors.Error {
	if conf.Port <= 0 {
		return derrors.NewInvalidArgumentError("port must be specified")
	}
	if conf.AgentPort <= 0 {
		return derrors.NewInvalidArgumentError("agentPort must be specified")
	}
	return nil
}

// Print the current configuration to the log system.
func (conf *Config) Print() {
	log.Info().Str("app", version.AppVersion).Str("commit", version.Commit).Msg("Version")
	log.Info().Int("management", conf.Port).Int("agent", conf.AgentPort).Msg("gRPC port")
}