/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
)

type ListenerManagerConfig interface {
	DeliveryMapperParallelism() int
	DeliveryBlockProcessParallelism() int
	DeliveryListenerTimeout() time.Duration
	DeliveryLRUSize() int
	DeliveryLRUBuffer() int
}

const (
	DeliveryMapperParallelism              = "token.finality.delivery.mapperParallelism"
	DeliveryBlockProcessParallelism        = "token.finality.delivery.blockProcessParallelism"
	DeliveryLRUSize                        = "token.finality.delivery.lruSize"
	DeliveryLRUBuffer                      = "token.finality.delivery.lruBuffer"
	DeliveryListenerTimeout                = "token.finality.delivery.listenerTimeout"
	DefaultDeliveryMapperParallelism       = 10
	DefaultDeliveryBlockProcessParallelism = 10
	DefaultDeliveryLRUSize                 = 30
	DefaultDeliveryLRUBuffer               = 15
	DefaultDeliveryListenerTimeout         = 10 * time.Second
)

type ManagerType string

const (
	Delivery ManagerType = "delivery"
)

func NewListenerManagerConfig(configService driver.ConfigService) *serviceListenerManagerConfig {
	return &serviceListenerManagerConfig{c: configService}
}

type serviceListenerManagerConfig struct {
	c driver.ConfigService
}

func (c *serviceListenerManagerConfig) DeliveryMapperParallelism() int {
	if v := c.c.GetInt(DeliveryMapperParallelism); v > 0 {
		return v
	}

	return DefaultDeliveryMapperParallelism
}

func (c *serviceListenerManagerConfig) DeliveryBlockProcessParallelism() int {
	if v := c.c.GetInt(DeliveryBlockProcessParallelism); v >= 0 {
		return v
	}

	return DefaultDeliveryBlockProcessParallelism
}

func (c *serviceListenerManagerConfig) DeliveryLRUSize() int {
	if v := c.c.GetInt(DeliveryLRUSize); v >= 0 {
		return v
	}

	return DefaultDeliveryLRUSize
}

func (c *serviceListenerManagerConfig) DeliveryLRUBuffer() int {
	if v := c.c.GetInt(DeliveryLRUBuffer); v >= 0 {
		return v
	}

	return DefaultDeliveryLRUBuffer
}

func (c *serviceListenerManagerConfig) DeliveryListenerTimeout() time.Duration {
	if v := c.c.GetDuration(DeliveryListenerTimeout); v >= 0 {
		return v
	}

	return DefaultDeliveryListenerTimeout
}

func (c *serviceListenerManagerConfig) String() string {
	return fmt.Sprintf("Delivery [mapperParalellism: %d, lru: (%d, %d), listenerTimeout: %v]", c.DeliveryMapperParallelism(), c.DeliveryLRUSize(), c.DeliveryLRUBuffer(), c.DeliveryListenerTimeout())
}
