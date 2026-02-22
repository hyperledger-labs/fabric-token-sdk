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

	idriver "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"go.uber.org/zap/zapcore"
)

var logger = logging.MustGetLogger()

type IdentityCacheBackendFunc func(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error)

type IdentityCache struct {
	once      sync.Once
	backed    IdentityCacheBackendFunc
	auditInfo []byte

	cache        chan *idriver.IdentityDescriptor
	cacheTimeout time.Duration

	metrics *Metrics
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int, auditInfo []byte, metrics *Metrics) *IdentityCache {
	logger.Debugf("new identity cache with size [%d]", size)
	ci := &IdentityCache{
		backed:       backed,
		cache:        make(chan *idriver.IdentityDescriptor, size),
		auditInfo:    auditInfo,
		cacheTimeout: 5 * time.Millisecond,
		metrics:      metrics,
	}

	return ci
}

func (c *IdentityCache) Identity(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
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

func (c *IdentityCache) fetchIdentityFromCache(ctx context.Context) (*idriver.IdentityDescriptor, error) {
	var identityDescriptor *idriver.IdentityDescriptor

	var start time.Time

	if logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}

	timeout := time.NewTimer(c.cacheTimeout)
	defer timeout.Stop()

	logger.DebugfContext(ctx, "fetch identity")
	select {
	case entry := <-c.cache:
		if entry == nil {
			return c.backed(ctx, c.auditInfo)
		}
		identityDescriptor = entry

		c.metrics.CacheLevelGauge.Add(-1)
		logger.DebugfContext(ctx, "fetched identity from cache")

		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.DebugfContext(ctx, "fetching identity from cache [%s][%d] took [%v]", identityDescriptor.Identity, len(identityDescriptor.AuditInfo), time.Since(start))
		}
	case <-timeout.C:
		logger.DebugfContext(ctx, "generate identity on the spot")
		var err error
		identityDescriptor, err = c.backed(ctx, c.auditInfo)
		if err != nil {
			return nil, err
		}
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.DebugfContext(ctx, "fetching identity from backend after a timeout [%s][%d] took [%v]", identityDescriptor.Identity, len(identityDescriptor.AuditInfo), time.Since(start))
		}
	}
	logger.DebugfContext(ctx, "fetch identity done")

	return identityDescriptor, nil
}

func (c *IdentityCache) fetchIdentityFromBackend(ctx context.Context, auditInfo []byte) (*idriver.IdentityDescriptor, error) {
	logger.DebugfContext(ctx, "fetching identity from backend")
	identityDescriptor, err := c.backed(ctx, auditInfo)
	if err != nil {
		return nil, err
	}
	logger.DebugfContext(ctx, "fetch identity from backend done [%s][%d]", identityDescriptor.Identity, len(identityDescriptor.AuditInfo))

	return identityDescriptor, nil
}

func (c *IdentityCache) provisionIdentities() {
	count := 0
	ctx := context.Background()
	for {
		identityDescriptor, err := c.backed(ctx, c.auditInfo)
		if err != nil {
			logger.Errorf("failed to provision identity [%s]", err)

			continue
		}
		logger.DebugfContext(ctx, "generated new idemix identity [%d]", count)
		c.metrics.CacheLevelGauge.Add(1)
		c.cache <- identityDescriptor
	}
}
