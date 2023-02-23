/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"runtime"
	"sync"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"go.uber.org/zap/zapcore"
)

const DefaultCacheSize = 3

type IdentityCacheBackendFunc func(opts *driver2.IdentityOptions) (view.Identity, []byte, error)

type identityCacheEntry struct {
	Identity view.Identity
	Audit    []byte
}

type IdentityCache struct {
	once   sync.Once
	backed IdentityCacheBackendFunc
	cache  chan identityCacheEntry
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int) *IdentityCache {
	ci := &IdentityCache{
		backed: backed,
		cache:  make(chan identityCacheEntry, size),
	}

	return ci
}

func (c *IdentityCache) Identity(opts *driver2.IdentityOptions) (view.Identity, []byte, error) {
	if !opts.EIDExtension || len(opts.AuditInfo) != 0 {
		return c.fetchIdentityFromBackend(opts)
	}

	c.once.Do(func() {
		if cap(c.cache) > 0 {
			// Spin up as many background goroutines as we need to prepare identities in the background.
			for i := 0; i < runtime.NumCPU(); i++ {
				go c.provisionIdentities()
			}
		}
	})

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("fetching identity from cache...")
	}

	return c.fetchIdentityFromCache(opts)

}

func (c *IdentityCache) fetchIdentityFromCache(opts *driver2.IdentityOptions) (view.Identity, []byte, error) {
	var identity view.Identity
	var audit []byte

	var start time.Time
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}

	select {
	case entry := <-c.cache:
		identity = entry.Identity
		audit = entry.Audit

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from cache [%s][%d] took %v", identity, len(audit), time.Since(start))
		}
	default:
		id, a, err := c.backed(opts)
		if err != nil {
			return nil, nil, err
		}
		identity = id
		audit = a

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from backend after a timeout [%s][%d] took %v", identity, len(audit), time.Since(start))
		}
	}

	return identity, audit, nil

}

func (c *IdentityCache) fetchIdentityFromBackend(opts *driver2.IdentityOptions) (view.Identity, []byte, error) {
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("fetching identity from backend")
	}
	id, audit, err := c.backed(opts)
	if err != nil {
		return nil, nil, err
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("fetch identity from backend done [%s][%d]", id, len(audit))
	}

	return id, audit, nil
}

func (c *IdentityCache) provisionIdentities() {
	for {
		id, audit, err := c.backed(&driver2.IdentityOptions{EIDExtension: true})
		if err != nil {
			continue
		}
		c.cache <- identityCacheEntry{Identity: id, Audit: audit}
	}
}

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
