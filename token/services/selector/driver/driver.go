/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"time"

	"github.com/pkg/errors"
)

const (
	Sherdlock Driver = "sherdlock"
	Simple    Driver = "simple"
)

type SelectorConfig interface {
	GetDriver() Driver
	GetNumRetries() int
	GetRetryInterval() time.Duration
}

type Driver string

type Config struct {
	Driver        Driver        `yaml:"driver,omitempty"`
	RetryInterval time.Duration `yaml:"retryInterval,omitempty"`
	NumRetries    int           `yaml:"numRetries,omitempty"`
}

func (c Config) GetDriver() Driver {
	if c.Driver == "" {
		return Sherdlock
	}
	return c.Driver
}

func (c Config) GetNumRetries() int {
	if c.NumRetries > 0 {
		return c.NumRetries
	}
	return 3
}
func (c Config) GetRetryInterval() time.Duration {
	if c.RetryInterval != 0 {
		return c.RetryInterval
	}
	return 5 * time.Second
}

type configService interface {
	UnmarshalKey(key string, rawVal interface{}) error
}

// New returns a SelectorConfig with the values from the token.selector key
func New(config configService) (SelectorConfig, error) {
	c := Config{}
	err := config.UnmarshalKey("token.selector", &c)
	if err != nil {
		return c, errors.Wrap(err, "invalid config for key [token.selector]: expected retryInterval (duration) and numRetries (integer))")
	}
	return c, nil
}
