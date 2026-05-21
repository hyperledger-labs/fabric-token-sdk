/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/selector/driver"
)

const (
	defaultDriver                 = driver.Sherdlock
	defaultLeaseExpiry            = 3 * time.Minute
	defaultLeaseCleanupTickPeriod = 1 * time.Minute
	defaultNumRetries             = 3
	defaultRetryInterval          = 5 * time.Second
	defaultFetcherCacheSize       = 0 // 0 means use fetcher default
	defaultFetcherCacheRefresh    = 0 // 0 means use fetcher default
	defaultFetcherCacheMaxQueries = 0 // 0 means use fetcher default

	// Security limits to prevent algorithmic attacks
	defaultMaxTokensPerSelection  = 10000            // Max tokens to iterate per selection
	defaultMaxLockAttempts        = 50000            // Max lock attempts per selection (5x iteration limit)
	defaultMaxRetryCycles         = 10               // Max outer retry loops
	defaultMaxLocksPerTransaction = 5000             // Max concurrent locks held per transaction
	defaultSelectionTimeout       = 30 * time.Second // Wall-clock timeout for selection
)

//go:generate counterfeiter -o mock/config_service.go -fake-name ConfigService . configService
type configService interface {
	UnmarshalKey(key string, rawVal any) error
}

// Limits defines resource limits for token selection to prevent algorithmic attacks
type Limits struct {
	MaxTokensPerSelection  int           `yaml:"maxTokensPerSelection,omitempty"`
	MaxLockAttempts        int           `yaml:"maxLockAttempts,omitempty"`
	MaxRetryCycles         int           `yaml:"maxRetryCycles,omitempty"`
	MaxLocksPerTransaction int           `yaml:"maxLocksPerTransaction,omitempty"`
	SelectionTimeout       time.Duration `yaml:"selectionTimeout,omitempty"`
}

type Config struct {
	Driver                 driver.Driver `yaml:"driver,omitempty"`
	RetryInterval          time.Duration `yaml:"retryInterval,omitempty"`
	NumRetries             int           `yaml:"numRetries,omitempty"`
	LeaseExpiry            time.Duration `yaml:"leaseExpiry,omitempty"`
	LeaseCleanupTickPeriod time.Duration `yaml:"leaseCleanupTickPeriod,omitempty"`
	FetcherCacheSize       int64         `yaml:"fetcherCacheSize,omitempty"`
	FetcherCacheRefresh    time.Duration `yaml:"fetcherCacheRefresh,omitempty"`
	FetcherCacheMaxQueries int           `yaml:"fetcherCacheMaxQueries,omitempty"`
	Limits                 Limits        `yaml:"limits,omitempty"`
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

func (c *Config) GetLeaseExpiry() time.Duration {
	if c.LeaseExpiry != 0 {
		return c.LeaseExpiry
	}

	return defaultLeaseExpiry
}

func (c *Config) GetLeaseCleanupTickPeriod() time.Duration {
	if c.LeaseCleanupTickPeriod != 0 {
		return c.LeaseCleanupTickPeriod
	}

	return defaultLeaseCleanupTickPeriod
}

func (c *Config) GetFetcherCacheSize() int64 {
	// Return 0 if not set, which will trigger use of fetcher default
	return c.FetcherCacheSize
}

func (c *Config) GetFetcherCacheRefresh() time.Duration {
	// Return 0 if not set, which will trigger use of fetcher default
	return c.FetcherCacheRefresh
}

func (c *Config) GetFetcherCacheMaxQueries() int {
	// Return 0 if not set, which will trigger use of fetcher default
	return c.FetcherCacheMaxQueries
}

// GetLimits returns the resource limits configuration with defaults applied
func (c *Config) GetLimits() Limits {
	limits := c.Limits

	if limits.MaxTokensPerSelection <= 0 {
		limits.MaxTokensPerSelection = defaultMaxTokensPerSelection
	}
	if limits.MaxLockAttempts <= 0 {
		limits.MaxLockAttempts = defaultMaxLockAttempts
	}
	if limits.MaxRetryCycles <= 0 {
		limits.MaxRetryCycles = defaultMaxRetryCycles
	}
	if limits.MaxLocksPerTransaction <= 0 {
		limits.MaxLocksPerTransaction = defaultMaxLocksPerTransaction
	}
	if limits.SelectionTimeout <= 0 {
		limits.SelectionTimeout = defaultSelectionTimeout
	}

	return limits
}

// Validate checks that the configuration is valid
func (c *Config) Validate() error {
	limits := c.GetLimits()

	if limits.MaxTokensPerSelection <= 0 {
		return errors.Errorf("maxTokensPerSelection must be positive, got %d", limits.MaxTokensPerSelection)
	}
	if limits.MaxLockAttempts <= 0 {
		return errors.Errorf("maxLockAttempts must be positive, got %d", limits.MaxLockAttempts)
	}
	if limits.MaxLockAttempts < limits.MaxTokensPerSelection {
		return errors.Errorf("maxLockAttempts (%d) should be >= maxTokensPerSelection (%d)",
			limits.MaxLockAttempts, limits.MaxTokensPerSelection)
	}
	if limits.MaxRetryCycles <= 0 {
		return errors.Errorf("maxRetryCycles must be positive, got %d", limits.MaxRetryCycles)
	}
	if limits.MaxLocksPerTransaction <= 0 {
		return errors.Errorf("maxLocksPerTransaction must be positive, got %d", limits.MaxLocksPerTransaction)
	}
	if limits.MaxLocksPerTransaction > limits.MaxTokensPerSelection {
		return errors.Errorf("maxLocksPerTransaction (%d) should be <= maxTokensPerSelection (%d)",
			limits.MaxLocksPerTransaction, limits.MaxTokensPerSelection)
	}
	if limits.SelectionTimeout <= 0 {
		return errors.Errorf("selectionTimeout must be positive, got %v", limits.SelectionTimeout)
	}

	return nil
}
