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

}

func NewConnectionConfig(conf *viper.Viper) (*ConnectionConfig, derrors.Error) {
	return nil, nil
}
