/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"bytes"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger("token-sdk.services.identity.msp.idemix")

type IdentityCacheBackendFunc func(opts *common.IdentityOptions) (driver.Identity, []byte, error)

type identityCacheEntry struct {
	Identity driver.Identity
	Audit    []byte
}

type IdentityCache struct {
	once   sync.Once
	backed IdentityCacheBackendFunc
	cache  chan identityCacheEntry
	opts   *common.IdentityOptions
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int, opts *common.IdentityOptions) *IdentityCache {
	logger.Debugf("new identity cache with size [%d]", size)
	ci := &IdentityCache{
		backed: backed,
		cache:  make(chan identityCacheEntry, size),
		opts:   opts,
	}

	return ci
}

func (c *IdentityCache) Identity(opts *common.IdentityOptions) (driver.Identity, []byte, error) {
	if opts != nil {
		// are the opts equal to the cache opts, if yes, use the cache
		if c.opts != nil && (opts.EIDExtension != c.opts.EIDExtension || !bytes.Equal(opts.AuditInfo, c.opts.AuditInfo)) {
			return c.fetchIdentityFromBackend(opts)
		}
	}

	c.once.Do(func() {
		logger.Debugf("provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("fetching identity from cache...")
	}

	return c.fetchIdentityFromCache()
}

func (c *IdentityCache) fetchIdentityFromCache() (driver.Identity, []byte, error) {
	var identity driver.Identity
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
			logger.Debugf("fetching identity from cache [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	case <-timeout.C:
		id, a, err := c.backed(c.opts)
		if err != nil {
			return nil, nil, err
		}
		identity = id
		audit = a

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from backend after a timeout [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	}
	return identity, audit, nil
}

func (c *IdentityCache) fetchIdentityFromBackend(opts *common.IdentityOptions) (driver.Identity, []byte, error) {
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
