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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

type IdentityCacheBackendFunc func(auditInfo []byte) (driver.Identity, []byte, error)

type identityCacheEntry struct {
	Identity driver.Identity
	Audit    []byte
}

type IdentityCache struct {
	once      sync.Once
	backed    IdentityCacheBackendFunc
	auditInfo []byte
	cache     chan identityCacheEntry
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int, auditInfo []byte) *IdentityCache {
	logger.Debugf("new identity cache with size [%d]", size)
	ci := &IdentityCache{
		backed:    backed,
		cache:     make(chan identityCacheEntry, size),
		auditInfo: auditInfo,
	}

	return ci
}

func (c *IdentityCache) Identity(auditInfo []byte) (driver.Identity, []byte, error) {
	// Is the auditInfo equal to that used to fill the cache? If yes, use the cache
	if !bytes.Equal(auditInfo, c.auditInfo) {
		return c.fetchIdentityFromBackend(auditInfo)
	}

	c.once.Do(func() {
		logger.Debugf("provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	logger.Debugf("fetching identity from cache...")

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
		id, a, err := c.backed(c.auditInfo)
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

func (c *IdentityCache) fetchIdentityFromBackend(auditInfo []byte) (driver.Identity, []byte, error) {
	logger.Debugf("fetching identity from backend")
	id, audit, err := c.backed(auditInfo)
	if err != nil {
		return nil, nil, err
	}
	logger.Debugf("fetch identity from backend done [%s][%d]", id, len(audit))

	return id, audit, nil
}

func (c *IdentityCache) provisionIdentities() {
	count := 0
	for {
		id, audit, err := c.backed(c.auditInfo)
		if err != nil {
			logger.Errorf("failed to provision identity [%s]", err)
			continue
		}
		logger.Debugf("generated new idemix identity [%d]", count)
		c.cache <- identityCacheEntry{Identity: id, Audit: audit}
	}
}
