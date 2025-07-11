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

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

var (
	cacheLevelOpts = metrics.GaugeOpts{
		Namespace:    "idemix",
		Name:         "cache_level",
		Help:         "Level of the idemix cache",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)

type IdentityCacheBackendFunc func(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error)

type identityCacheEntry struct {
	Identity driver.Identity
	Audit    []byte
}

type IdentityCache struct {
	once      sync.Once
	backed    IdentityCacheBackendFunc
	auditInfo []byte

	cache           chan identityCacheEntry
	cacheTimeout    time.Duration
	cacheLevelGauge metrics.Gauge
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int, auditInfo []byte, metricsProvider metrics.Provider) *IdentityCache {
	logger.Debugf("new identity cache with size [%d]", size)
	ci := &IdentityCache{
		backed:          backed,
		cache:           make(chan identityCacheEntry, size),
		auditInfo:       auditInfo,
		cacheTimeout:    5 * time.Millisecond,
		cacheLevelGauge: metricsProvider.NewGauge(cacheLevelOpts),
	}

	return ci
}

func (c *IdentityCache) Identity(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	// Is the auditInfo equal to that used to fill the cache? If yes, use the cache
	if !bytes.Equal(auditInfo, c.auditInfo) {
		return c.fetchIdentityFromBackend(ctx, auditInfo)
	}

	c.once.Do(func() {
		logger.Debugf("provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	logger.Debugf("fetching identity from cache...")

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

	span := trace.SpanFromContext(ctx)
	span.AddEvent("fetch_identity")
	select {
	case entry := <-c.cache:
		c.cacheLevelGauge.Add(-1)
		span.AddEvent("got_identity_from_cache")
		identity = entry.Identity
		audit = entry.Audit

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from cache [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	case <-timeout.C:
		span.AddEvent("generate_identity_on_the_spot")
		id, a, err := c.backed(ctx, c.auditInfo)
		if err != nil {
			return nil, nil, err
		}
		identity = id
		audit = a

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetching identity from backend after a timeout [%s][%d] took [%v]", identity, len(audit), time.Since(start))
		}
	}
	span.AddEvent("got_identity")
	return identity, audit, nil
}

func (c *IdentityCache) fetchIdentityFromBackend(ctx context.Context, auditInfo []byte) (driver.Identity, []byte, error) {
	logger.Debugf("fetching identity from backend")
	id, audit, err := c.backed(ctx, auditInfo)
	if err != nil {
		return nil, nil, err
	}
	logger.Debugf("fetch identity from backend done [%s][%d]", id, len(audit))

	return id, audit, nil
}

func (c *IdentityCache) provisionIdentities() {
	count := 0
	ctx := context.Background()
	for {
		id, audit, err := c.backed(ctx, c.auditInfo)
		if err != nil {
			logger.Errorf("failed to provision identity [%s]", err)
			continue
		}
		logger.Debugf("generated new idemix identity [%d]", count)
		c.cacheLevelGauge.Add(1)
		c.cache <- identityCacheEntry{Identity: id, Audit: audit}
	}
}
