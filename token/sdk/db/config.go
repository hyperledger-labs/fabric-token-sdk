/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
)

type ConfigProvider interface {
	ConfigurationFor(network, channel, namespace string) (config.Configuration, error)
}

type Config struct {
	configProvider    ConfigProvider
	configurationKeys []string
}

func NewConfig(configProvider ConfigProvider, configurationKeys ...string) *Config {
	return &Config{configProvider: configProvider, configurationKeys: configurationKeys}
}

func (c *Config) DriverFor(tmsID token.TMSID) (string, error) {
	tmsConfig, err := c.configProvider.ConfigurationFor(tmsID.Network, tmsID.Channel, tmsID.Namespace)
	if err != nil {
		return "", errors.WithMessagef(err, "failed to load configuration for tms [%s]", tmsID)
	}
	for _, key := range c.configurationKeys {
		if tmsConfig.IsSet(key) {
			return tmsConfig.GetString(key), nil
		}
	}
	return "", errors.Errorf("configuration not found")
}
