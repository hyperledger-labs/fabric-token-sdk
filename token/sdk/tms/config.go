/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/config"
)

type ConfigServiceWrapper struct {
	*config.Service
}

func NewConfigServiceWrapper(service *config.Service) *ConfigServiceWrapper {
	return &ConfigServiceWrapper{Service: service}
}

func (c *ConfigServiceWrapper) Configurations() ([]driver.Configuration, error) {
	configs, err := c.Service.Configurations()
	if err != nil {
		return nil, err
	}
	result := make([]driver.Configuration, len(configs))
	for i, config := range configs {
		result[i] = config
	}

	return result, nil
}

func (c *ConfigServiceWrapper) ConfigurationFor(network string, channel string, namespace string) (driver.Configuration, error) {
	return c.Service.ConfigurationFor(network, channel, namespace)
}
