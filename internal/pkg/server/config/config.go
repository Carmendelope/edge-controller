/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package config

import (
	"github.com/nalej/derrors"
	"github.com/nalej/edge-controller/version"
	"github.com/rs/zerolog/log"
	"time"
)

type Config struct {
	// Debug level is active.
	Debug bool
	// Port where the edge controller receives messages from the management cluster.
	Port int
	// Port where the edge controller receives messages from agents.
	AgentPort int
	// UseInMemoryProviders determines if the in memory providers are used.
	UseInMemoryProviders bool
	// UseBBoltProviders determines if Bbolt providers are used
	UseBBoltProviders bool
	// BboltPath determines the file on disk where  a snapshot of the data is
	BboltPath string
	// NotifyPeriod determines how often the EIC sends data back to the management cluster.
	NotifyPeriod time.Duration
	// EdgeManagementURL with the URL required to connect to the Management cluster
	EdgeManagementURL string
	// OrganizationId with the organization identifier
	OrganizationId string
	// EdgeControllerId with the edge controller identifier
	EdgeControllerId string

}

// Validate the current configuration.
func (conf * Config) Validate() derrors.Error {
	if conf.Port <= 0 {
		return derrors.NewInvalidArgumentError("port must be specified")
	}
	if conf.AgentPort <= 0 {
		return derrors.NewInvalidArgumentError("agentPort must be specified")
	}
	if conf.NotifyPeriod.Seconds() < 1 {
		return derrors.NewInvalidArgumentError("notifyPeriod should be minimum 1s")
	}
	if conf.UseBBoltProviders {
		if conf.BboltPath == "" {
			return derrors.NewAlreadyExistsError("bboltpatth must be specified")
		}
	}
	if !conf.UseBBoltProviders && !conf.UseInMemoryProviders {
		return derrors.NewInvalidArgumentError("a type of provider must be selected")
	}
	if conf.UseBBoltProviders && conf.UseInMemoryProviders {
		return derrors.NewInvalidArgumentError("only one type of provider must be selected")
	}
	return nil
}

// Print the current configuration to the log system.
func (conf *Config) Print() {
	log.Info().Str("app", version.AppVersion).Str("commit", version.Commit).Msg("Version")
	log.Info().Int("management", conf.Port).Int("agent", conf.AgentPort).Msg("gRPC port")
	if conf.UseInMemoryProviders {
		log.Info().Bool("UseInMemoryProviders", conf.UseInMemoryProviders).Msg("Using in-memory providers")
	}
	if conf.UseBBoltProviders {
		log.Info().Bool("UseBBoltProviders", conf.UseBBoltProviders).Msg("Using bbolt providers")
		log.Info().Str("BboltPath", conf.BboltPath).Msg("BboltPath")
	}
	log.Info().Str("duration", conf.NotifyPeriod.String()).Msg("Notify period")
}