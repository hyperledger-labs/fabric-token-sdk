/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

type WalletIdentityCacheBackendFunc func() (driver.Identity, error)

type WalletIdentityCache struct {
	Logger logging.Logger

	once    sync.Once
	backed  WalletIdentityCacheBackendFunc
	cache   chan driver.Identity
	timeout time.Duration
}

func NewWalletIdentityCache(Logger logging.Logger, backed WalletIdentityCacheBackendFunc, size int) *WalletIdentityCache {
	if size < 0 {
		size = 0
	}
	ci := &WalletIdentityCache{
		Logger:  Logger,
		backed:  backed,
		cache:   make(chan driver.Identity, size),
		timeout: time.Millisecond * 100,
	}
	return ci
}

func (c *WalletIdentityCache) Identity() (driver.Identity, error) {
	c.once.Do(func() {
		c.Logger.Debugf("provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	var start time.Time
	if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}
	timeout := time.NewTimer(c.timeout)
	defer timeout.Stop()

	var identity driver.Identity
	var err error
	select {
	case entry := <-c.cache:
		identity = entry
		if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
			c.Logger.Debugf("fetching wallet identity from cache [%s] took [%v]", identity, time.Since(start))
		}
	case <-timeout.C:
		identity, err = c.backed()
		if err != nil {
			return nil, errors.Wrap(err, "failed fetching wallet identity")
		}
		if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
			c.Logger.Debugf("fetching wallet identity from backend after a timeout [%s] took [%v]", identity, time.Since(start))
		}
	}
	return identity, nil
}

func (c *WalletIdentityCache) provisionIdentities() {
	for {
		id, err := c.backed()
		if err != nil {
			continue
		}
		c.cache <- id
	}
}
