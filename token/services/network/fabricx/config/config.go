/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/fabric/config"
)

const (
	Type = config.Type
)

type ManagerType = config.ManagerType

const (
	Notification ManagerType = "notification"
)

func NewListenerManagerConfig(configService driver.ConfigService) *serviceListenerManagerConfig {
	return &serviceListenerManagerConfig{c: configService}
}

type serviceListenerManagerConfig struct {
	c driver.ConfigService
}

func (c *serviceListenerManagerConfig) Type() ManagerType {
	if v := ManagerType(c.c.GetString(Type)); len(v) > 0 {
		return v
	}
	return Notification
}
