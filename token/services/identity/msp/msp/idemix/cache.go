/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	"runtime"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"go.uber.org/zap/zapcore"
)

type IdentityCacheBackendFunc func(opts *driver.IdentityOptions) (view.Identity, []byte, error)

type identityCacheEntry struct {
	Identity view.Identity
	Audit    []byte
}

type IdentityCache struct {
	once   sync.Once
	backed IdentityCacheBackendFunc
	cache  chan identityCacheEntry
	opts   *driver.IdentityOptions
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int, opts *driver.IdentityOptions) *IdentityCache {
	ci := &IdentityCache{
		backed: backed,
		cache:  make(chan identityCacheEntry, size),
		opts:   opts,
	}

	return ci
}

func (c *IdentityCache) Identity(opts *driver.IdentityOptions) (view.Identity, []byte, error) {
	if opts != nil {
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

func (c *IdentityCache) fetchIdentityFromCache(opts *driver.IdentityOptions) (view.Identity, []byte, error) {
	var identity view.Identity
	var audit []byte

	var start time.Time

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}

	timeout := time.NewTimer(time.Second)
	defer timeout.Stop()

	select {

	case entry := <-c.cache:
		identity = entry.Identity
		audit = entry.Audit

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from cache [%s][%d] took %v", identity, len(audit), time.Since(start))
		}

	case <-timeout.C:
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

func (c *IdentityCache) fetchIdentityFromBackend(opts *driver.IdentityOptions) (view.Identity, []byte, error) {
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
	count := 0
	for {
		id, audit, err := c.backed(c.opts)
		if err != nil {
			logger.Errorf("failed to provision identity [%s]", err)
			continue
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("generated new idemix identity [%d]", count)
		}
		c.cache <- identityCacheEntry{Identity: id, Audit: audit}
	}
}
