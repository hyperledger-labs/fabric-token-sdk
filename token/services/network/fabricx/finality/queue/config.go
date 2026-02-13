/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package queue

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
)

const (
	// Workers is the configuration key for the number of worker goroutines
	Workers = "token.finality.notification.workers"
	// QueueSize is the configuration key for the size of the event buffer
	QueueSize = "token.finality.notification.queueSize"
	// DefaultWorkers is the default number of worker goroutines
	DefaultWorkers = 10
	// DefaultQueueSize is the default size of the event buffer
	DefaultQueueSize = 1000
)

// ConfigGetter models the configuration getter for the event queue
type ConfigGetter interface {
	// Workers returns the number of worker goroutines
	Workers() int
	// QueueSize returns the size of the event buffer
	QueueSize() int
}

// NewConfig creates a new ConfigGetter
func NewConfig(configService driver.ConfigService) *serviceConfig {
	return &serviceConfig{c: configService}
}

type serviceConfig struct {
	c driver.ConfigService
}

// Workers returns the number of worker goroutines.
func (c *serviceConfig) Workers() int {
	if v := c.c.GetInt(Workers); v > 0 {
		return v
	}
	return DefaultWorkers
}

// QueueSize returns the size of the event buffer.
func (c *serviceConfig) QueueSize() int {
	if v := c.c.GetInt(QueueSize); v > 0 {
		return v
	}
	return DefaultQueueSize
}
