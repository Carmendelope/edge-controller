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

	// Protocol (http/https), hostname and port
	Address string

	// Database name
	Database string
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
		Address: providerConf.GetString("address"),
		Database: providerConf.GetString("database"),
	}

	return connConf, nil
}
