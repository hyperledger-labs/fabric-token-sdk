/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
)

type WalletIdentityCacheBackendFunc func() (view.Identity, error)

type WalletIdentityCache struct {
	Logger  *flogging.FabricLogger
	backed  WalletIdentityCacheBackendFunc
	ch      chan view.Identity
	timeout time.Duration
}

func NewWalletIdentityCache(Logger *flogging.FabricLogger, backed WalletIdentityCacheBackendFunc, size int) *WalletIdentityCache {
	ci := &WalletIdentityCache{
		Logger:  Logger,
		backed:  backed,
		ch:      make(chan view.Identity, size),
		timeout: time.Millisecond * 100,
	}
	if size > 0 {
		go ci.run()
	}
	return ci
}

func (c *WalletIdentityCache) Identity() (view.Identity, error) {
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
