/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/logging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type WalletIdentityCacheBackendFunc func() (driver.Identity, error)

type WalletIdentityCache struct {
	Logger  logging.Logger
	backed  WalletIdentityCacheBackendFunc
	ch      chan driver.Identity
	timeout time.Duration
}

func NewWalletIdentityCache(Logger logging.Logger, backed WalletIdentityCacheBackendFunc, size int) *WalletIdentityCache {
	ci := &WalletIdentityCache{
		Logger:  Logger,
		backed:  backed,
		ch:      make(chan driver.Identity, size),
		timeout: time.Millisecond * 100,
	}
	if size > 0 {
		go ci.run()
	}
	return ci
}

func (c *WalletIdentityCache) Identity() (driver.Identity, error) {
	select {
	case entry := <-c.ch:
		c.Logger.Debugf("fetch identity from producer channel done [%s][%d]", entry)
		return entry, nil
	default:
		c.Logger.Debugf("fetch identity from producer channel timeout")
		return c.backed()
	}
}

func (c *WalletIdentityCache) run() {
	for {
		id, err := c.backed()
		if err != nil {
			continue
		}
		c.ch <- id
	}
}
