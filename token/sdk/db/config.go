/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/config"
	"github.com/pkg/errors"
)

type ConfigProvider interface {
	UnmarshalKey(key string, rawVal interface{}) error
	GetString(key string) string
	IsSet(key string) bool
	TranslatePath(path string) string
}

type Config struct {
	configProvider ConfigProvider
	key            string
}

func NewConfig(configProvider ConfigProvider, key string) *Config {
	return &Config{configProvider: configProvider, key: key}
}

func (c *Config) DriverFor(tmsID token.TMSID) (string, error) {
	tmsConfig, err := config.NewTokenSDK(c.configProvider).GetTMS(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	return tmsConfig.GetString(c.key), nil
}
