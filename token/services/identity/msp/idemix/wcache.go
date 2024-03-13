/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"go.uber.org/zap/zapcore"
)

type WalletIdentityCacheBackendFunc func() (view.Identity, error)

type WalletIdentityCache struct {
	backed  WalletIdentityCacheBackendFunc
	ch      chan view.Identity
	timeout time.Duration
}

func NewWalletIdentityCache(backed WalletIdentityCacheBackendFunc, size int) *WalletIdentityCache {
	ci := &WalletIdentityCache{
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
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel done [%s][%d]", entry)
		}
		return entry, nil
	default:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel timeout")
		}
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
