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
	Type() ManagerType
	CommitterMaxRetries() int
	CommitterRetryWaitDuration() time.Duration
	DeliveryMapperParallelism() int
	DeliveryBlockProcessParallelism() int
	DeliveryListenerTimeout() time.Duration
	DeliveryLRUSize() int
	DeliveryLRUBuffer() int
}

const (
	Type                                   = "token.finality.type"
	CommitterMaxRetries                    = "token.finality.committer.maxRetries"
	CommitterRetryWaitDuration             = "token.finality.committer.retryWaitDuration"
	DeliveryMapperParallelism              = "token.finality.delivery.mapperParallelism"
	DeliveryBlockProcessParallelism        = "token.finality.delivery.blockProcessParallelism"
	DeliveryLRUSize                        = "token.finality.delivery.lruSize"
	DeliveryLRUBuffer                      = "token.finality.delivery.lruBuffer"
	DeliveryListenerTimeout                = "token.finality.delivery.listenerTimeout"
	DefaultCommitterMaxRetries             = 3
	DefaultCommitterRetryWaitDuration      = 5 * time.Second
	DefaultDeliveryMapperParallelism       = 10
	DefaultDeliveryBlockProcessParallelism = 10
	DefaultDeliveryLRUSize                 = 30
	DefaultDeliveryLRUBuffer               = 15
	DefaultDeliveryListenerTimeout         = 10 * time.Second
)

type ManagerType string

const (
	Delivery     ManagerType = "delivery"
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

	return Delivery
}

func (c *serviceListenerManagerConfig) CommitterMaxRetries() int {
	if v := c.c.GetInt(CommitterMaxRetries); v >= 0 {
		return v
	}

	return DefaultCommitterMaxRetries
}

func (c *serviceListenerManagerConfig) CommitterRetryWaitDuration() time.Duration {
	if v := c.c.GetDuration(CommitterRetryWaitDuration); v >= 0 {
		return v
	}

	return DefaultCommitterRetryWaitDuration
}

func (c *serviceListenerManagerConfig) DeliveryMapperParallelism() int {
	if v := c.c.GetInt(DeliveryMapperParallelism); v > 0 {
		return v
	}

	return DefaultDeliveryMapperParallelism
}

func (c *serviceListenerManagerConfig) DeliveryBlockProcessParallelism() int {
	if v := c.c.GetInt(DeliveryBlockProcessParallelism); v > 0 {
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
	if c.Type() == Delivery {
		return fmt.Sprintf("Delivery [mapperParalellism: %d, lru: (%d, %d), listenerTimeout: %v]", c.DeliveryMapperParallelism(), c.DeliveryLRUSize(), c.DeliveryLRUBuffer(), c.DeliveryListenerTimeout())
	}

	return fmt.Sprintf("Invalid config type: [%s]", c.Type())
}
