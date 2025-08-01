/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package cache

import (
	"bytes"
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

type IdentityCacheBackendFunc func(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error)

type identityCacheEntry struct {
	Identity driver.Identity
	Audit    []byte
}

type IdentityCache struct {
	once      sync.Once
	backed    IdentityCacheBackendFunc
	auditInfo []byte

	cache        chan identityCacheEntry
	cacheTimeout time.Duration

	metrics *Metrics
}

func NewIdentityCache(ctx context.Context, backed IdentityCacheBackendFunc, size int, auditInfo []byte, metrics *Metrics) *IdentityCache {
	logger.DebugfContext(ctx, "new identity cache with size [%d]", size)
	ci := &IdentityCache{
		backed:       backed,
		cache:        make(chan identityCacheEntry, size),
		auditInfo:    auditInfo,
		cacheTimeout: 5 * time.Millisecond,
		metrics:      metrics,
	}

	return ci
}

func (c *IdentityCache) Identity(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	// Is the auditInfo equal to that used to fill the cache? If yes, use the cache
	if !bytes.Equal(auditInfo, c.auditInfo) {
		return c.fetchIdentityFromBackend(ctx, auditInfo)
	}

	c.once.Do(func() {
		logger.DebugfContext(ctx, "provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	logger.DebugfContext(ctx, "fetching identity from cache...")

	return c.fetchIdentityFromCache(ctx)
}

func (c *IdentityCache) fetchIdentityFromCache(ctx context.Context) (driver.Identity, []byte, error) {
	var identity driver.Identity
	var audit []byte

	var start time.Time

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}

	timeout := time.NewTimer(c.cacheTimeout)
	defer timeout.Stop()

	logger.DebugfContext(ctx, "fetch identity")
	select {
	case entry := <-c.cache:
		c.metrics.CacheLevelGauge.Add(-1)
		logger.DebugfContext(ctx, "fetched identity from cache")
		identity = entry.Identity
		audit = entry.Audit

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.DebugfContext(ctx, "fetching identity from cache [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	case <-timeout.C:
		logger.DebugfContext(ctx, "generate identity on the spot")
		id, a, err := c.backed(ctx, c.auditInfo)
		if err != nil {
			return nil, nil, err
		}
		identity = id
		audit = a

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.DebugfContext(ctx, "fetching identity from backend after a timeout [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	}
	logger.DebugfContext(ctx, "fetch identity done")

	return identity, audit, nil
}

func (c *IdentityCache) fetchIdentityFromBackend(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	logger.DebugfContext(ctx, "fetching identity from backend")
	id, audit, err := c.backed(ctx, auditInfo)
	if err != nil {
		return nil, nil, err
	}
	logger.DebugfContext(ctx, "fetch identity from backend done [%s][%d]", id, len(audit))

	return id, audit, nil
}

func (c *IdentityCache) provisionIdentities() {
	count := 0
	ctx := context.Background()
	for {
		id, audit, err := c.backed(ctx, c.auditInfo)
		if err != nil {
			logger.ErrorfContext(ctx, "failed to provision identity [%s]", err)
			continue
		}
		logger.DebugfContext(ctx, "generated new idemix identity [%d]", count)
		c.metrics.CacheLevelGauge.Add(1)
		c.cache <- identityCacheEntry{Identity: id, Audit: audit}
	}
}
