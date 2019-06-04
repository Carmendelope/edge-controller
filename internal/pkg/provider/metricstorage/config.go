/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metricstorage

// Connection configuration for metric storage provider

import (
	"github.com/nalej/derrors"

	"github.com/spf13/viper"
)

type ConnectionConfig struct {
	providerType ProviderType
}

const defaultProviderType ProviderType = "influxdb"

func NewConnectionConfig(conf *viper.Viper) (*ConnectionConfig, derrors.Error) {
	// Current we only have the InfluxDB provider
	t := defaultProviderType

	providerConf := conf.Sub(t.String())
	if providerConf == nil {
		providerConf = viper.New()
	}

	connConf := &ConnectionConfig{
		providerType: t,
	}

	return connConf, nil
}
