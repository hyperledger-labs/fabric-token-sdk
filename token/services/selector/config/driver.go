/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/driver"
	"github.com/pkg/errors"
)

const (
	defaultDriver            = driver.Sherdlock
	defaultEvictionInterval  = 3 * time.Minute
	defaultCleanupTickPeriod = 1 * time.Minute
	defaultNumRetries        = 3
	defaultRetryInterval     = 5 * time.Second
)

type configService interface {
	UnmarshalKey(key string, rawVal interface{}) error
}

type Config struct {
	Driver            driver.Driver `yaml:"driver,omitempty"`
	RetryInterval     time.Duration `yaml:"retryInterval,omitempty"`
	NumRetries        int           `yaml:"numRetries,omitempty"`
	EvictionInterval  time.Duration `yaml:"evictionInterval,omitempty"`
	CleanupTickPeriod time.Duration `yaml:"cleanupTickPeriod,omitempty"`
}

// New returns a SelectorConfig with the values from the token.selector key
func New(config configService) (*Config, error) {
	c := &Config{}
	err := config.UnmarshalKey("token.selector", c)
	if err != nil {
		return nil, errors.Wrap(err, "invalid config for key [token.selector]: expected retryInterval (duration) and numRetries (integer))")
	}
	return c, nil
}

func (c *Config) GetDriver() driver.Driver {
	if c.Driver == "" {
		return defaultDriver
	}
	return c.Driver
}

func (c *Config) GetNumRetries() int {
	if c.NumRetries > 0 {
		return c.NumRetries
	}
	return defaultNumRetries
}

func (c *Config) GetRetryInterval() time.Duration {
	if c.RetryInterval != 0 {
		return c.RetryInterval
	}
	return defaultRetryInterval
}

func (c *Config) GetEvictionInterval() time.Duration {
	if c.EvictionInterval != 0 {
		return c.EvictionInterval
	}
	return defaultEvictionInterval
}

func (c *Config) GetCleanupTickPeriod() time.Duration {
	if c.CleanupTickPeriod != 0 {
		return c.CleanupTickPeriod
	}
	return defaultCleanupTickPeriod
}
