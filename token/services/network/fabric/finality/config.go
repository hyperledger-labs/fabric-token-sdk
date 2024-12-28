/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package finality

import (
	"fmt"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
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

type ManagerType string

const (
	Committer ManagerType = "committer"
	Delivery  ManagerType = "delivery"
)

func NewListenerManagerConfig(configService driver.ConfigService) *serviceListenerManagerConfig {
	return &serviceListenerManagerConfig{c: configService}
}

type serviceListenerManagerConfig struct {
	c driver.ConfigService
}

func (c *serviceListenerManagerConfig) Type() ManagerType {
	if v := ManagerType(c.c.GetString("token.finality.type")); len(v) > 0 {
		return v
	}
	return Committer
}

func (c *serviceListenerManagerConfig) CommitterMaxRetries() int {
	if v := c.c.GetInt("token.finality.committer.maxRetries"); v >= 0 {
		return v
	}
	return 3
}

func (c *serviceListenerManagerConfig) CommitterRetryWaitDuration() time.Duration {
	if v := c.c.GetDuration("token.finality.committer.retryWaitDuration"); v >= 0 {
		return v
	}
	return 5 * time.Second
}

func (c *serviceListenerManagerConfig) DeliveryMapperParallelism() int {
	if v := c.c.GetInt("token.finality.delivery.mapperParallelism"); v > 0 {
		return v
	}
	return 10
}

func (c *serviceListenerManagerConfig) DeliveryBlockProcessParallelism() int {
	return c.c.GetInt("token.finality.delivery.blockProcessParallelism")
}

func (c *serviceListenerManagerConfig) DeliveryLRUSize() int {
	if v := c.c.GetInt("token.finality.delivery.lruSize"); v >= 0 {
		return v
	}
	return 30
}

func (c *serviceListenerManagerConfig) DeliveryLRUBuffer() int {
	if v := c.c.GetInt("token.finality.delivery.lruBuffer"); v >= 0 {
		return v
	}
	return 15
}

func (c *serviceListenerManagerConfig) DeliveryListenerTimeout() time.Duration {
	if v := c.c.GetDuration("token.finality.delivery.listenerTimeout"); v >= 0 {
		return v
	}
	return 10 * time.Second
}

func (c *serviceListenerManagerConfig) String() string {
	if c.Type() == Delivery {
		return fmt.Sprintf("Delivery [mapperParalellism: %d, lru: (%d, %d), listenerTimeout: %v]", c.DeliveryMapperParallelism(), c.DeliveryLRUSize(), c.DeliveryLRUBuffer(), c.DeliveryListenerTimeout())
	}
	if c.Type() == Committer {
		return fmt.Sprintf("Committer [retries: (%d, %v)]", c.CommitterMaxRetries(), c.CommitterRetryWaitDuration())
	}
	return fmt.Sprintf("Invalid config type: [%s]", c.Type())
}
