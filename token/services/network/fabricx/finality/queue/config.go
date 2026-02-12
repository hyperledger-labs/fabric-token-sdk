/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package queue

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
)

const (
	Workers          = "token.finality.notification.workers"
	QueueSize        = "token.finality.notification.queueSize"
	DefaultWorkers   = 10
	DefaultQueueSize = 1000
)

type ConfigGetter interface {
	Workers() int
	QueueSize() int
}

func NewConfig(configService driver.ConfigService) *serviceConfig {
	return &serviceConfig{c: configService}
}

type serviceConfig struct {
	c driver.ConfigService
}

func (c *serviceConfig) Workers() int {
	if v := c.c.GetInt(Workers); v > 0 {
		return v
	}
	return DefaultWorkers
}

func (c *serviceConfig) QueueSize() int {
	if v := c.c.GetInt(QueueSize); v > 0 {
		return v
	}
	return DefaultQueueSize
}
