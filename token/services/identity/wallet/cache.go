/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/metrics"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.uber.org/zap/zapcore"
)

var (
	cacheLevelOpts = metrics.GaugeOpts{
		Namespace:    "wallet",
		Name:         "recipient_data_cache_level",
		Help:         "Level of the wallet recipient data cache",
		LabelNames:   []string{"network", "channel", "namespace"},
		StatsdFormat: "%{#fqname}.%{network}.%{channel}.%{namespace}",
	}
)

var logger = logging.MustGetLogger()

type IdentityCacheBackendFunc func(ctx context.Context) (*driver.RecipientData, error)

type IdentityCache struct {
	Logger logging.Logger

	once   sync.Once
	backed IdentityCacheBackendFunc

	cache           chan *driver.RecipientData
	cacheTimeout    time.Duration
	cacheLevelGauge metrics.Gauge
}

func NewIdentityCache(Logger logging.Logger, backed IdentityCacheBackendFunc, size int, metricsProvider metrics.Provider) *IdentityCache {
	if size < 0 {
		size = 0
	}
	ci := &IdentityCache{
		Logger:          Logger,
		backed:          backed,
		cache:           make(chan *driver.RecipientData, size),
		cacheTimeout:    time.Millisecond * 5,
		cacheLevelGauge: metricsProvider.NewGauge(cacheLevelOpts),
	}
	return ci
}

func (c *IdentityCache) RecipientData(ctx context.Context) (*driver.RecipientData, error) {
	c.once.Do(func() {
		c.Logger.Infof("provision wallet recipient data with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})

	var start time.Time
	if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}
	timeout := time.NewTimer(c.cacheTimeout)
	defer timeout.Stop()

	var identity *driver.RecipientData
	var err error
	logger.DebugfContext(ctx, "fetching wallet recipient data")
	select {
	case entry := <-c.cache:
		c.cacheLevelGauge.Add(-1)
		logger.DebugfContext(ctx, "fetched wallet recipient data from cache")
		identity = entry
		c.Logger.DebugfContext(ctx, "fetching wallet identity from cache [%s] took [%v]", identity, time.Since(start))
	case <-timeout.C:
		logger.DebugfContext(ctx, "generating wallet recipient data on the spot")
		identity, err = c.backed(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed fetching wallet identity")
		}
		c.Logger.DebugfContext(ctx, "fetching wallet identity from backend after a timeout [%s] took [%v]", identity, time.Since(start))
	case <-ctx.Done():
		return nil, errors.New("context is done")
	}
	logger.DebugfContext(ctx, "fetching wallet recipient data done")

	return identity, nil
}

func (c *IdentityCache) provisionIdentities() {
	ctx := context.Background()
	for {
		id, err := c.backed(ctx)
		if err != nil {
			continue
		}
		c.cacheLevelGauge.Add(1)
		c.cache <- id
	}
}
