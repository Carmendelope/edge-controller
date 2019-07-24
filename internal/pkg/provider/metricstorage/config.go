/*
 * Copyright (C) 2019 Nalej - All Rights Reserved
 */

package metricstorage

// Connection configuration for metric storage provider

import (
	"time"

	"github.com/nalej/derrors"

	"github.com/influxdata/influxql" // For convenient duration parsing
	"github.com/spf13/viper"

	"github.com/rs/zerolog/log"
)

type ConnectionConfig struct {
	providerType ProviderType

	// Protocol (http/https), hostname and port
	Address string

	// Database name
	Database string

	// Retention policy duration
	Retention time.Duration
}

const defaultProviderType ProviderType = "influxdb"

func NewConnectionConfig(conf *viper.Viper) (*ConnectionConfig, derrors.Error) {
	// Current we only have the InfluxDB provider
	t := defaultProviderType

	confProvider := conf.GetString("provider")
	if confProvider != "" {
		t = ProviderType(confProvider)
	}

	providerConf := conf.Sub(t.String())
	if providerConf == nil {
		providerConf = viper.New()
	}

	dur, derr := retentionFromStr(conf.GetString("retention"))
	if derr != nil {
		return nil, derr
	}

	connConf := &ConnectionConfig{
		providerType: t,
		Address: providerConf.GetString("address"),
		Database: providerConf.GetString("database"),
		Retention: dur,
	}

	return connConf, nil
}

func retentionFromStr(retentionStr string) (time.Duration, derrors.Error) {
	if retentionStr == "inf" || retentionStr == "" {
		log.Warn().Msg("metrics data retention period set to infinite - data will never be expired")
		return 0, nil
	}

	dur, err := influxql.ParseDuration(retentionStr)
	if err != nil {
		return 0, derrors.NewInvalidArgumentError("invalid retention duration", err).WithParams(retentionStr)
	}

	return dur, nil
}
