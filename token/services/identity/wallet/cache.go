/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package wallet

import (
	"context"
	"sync"
	"time"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap/zapcore"
)

type IdentityCacheBackendFunc func(ctx context.Context) (*driver.RecipientData, error)

type IdentityCache struct {
	Logger logging.Logger

	once    sync.Once
	backed  IdentityCacheBackendFunc
	cache   chan *driver.RecipientData
	timeout time.Duration
}

func NewIdentityCache(Logger logging.Logger, backed IdentityCacheBackendFunc, size int) *IdentityCache {
	if size < 0 {
		size = 0
	}
	ci := &IdentityCache{
		Logger:  Logger,
		backed:  backed,
		cache:   make(chan *driver.RecipientData, size),
		timeout: time.Millisecond * 100,
	}
	return ci
}

func (c *IdentityCache) RecipientData(ctx context.Context) (*driver.RecipientData, error) {
	c.once.Do(func() {
		c.Logger.Debugf("provision identities with cache size [%d]", cap(c.cache))
		if cap(c.cache) > 0 {
			go c.provisionIdentities()
		}
	})
	span := trace.SpanFromContext(ctx)

	var start time.Time
	if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
		start = time.Now()
	}
	timeout := time.NewTimer(c.timeout)
	defer timeout.Stop()

	var identity *driver.RecipientData
	var err error
	span.AddEvent("fetch_recipient_data")
	select {
	case entry := <-c.cache:
		span.AddEvent("got_recipient_data_from_cache")
		identity = entry
		if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
			c.Logger.Debugf("fetching wallet identity from cache [%s] took [%v]", identity, time.Since(start))
		}
	case <-timeout.C:
		span.AddEvent("generate_recipient_data_on_the_spot")
		identity, err = c.backed(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "failed fetching wallet identity")
		}
		if c.Logger.IsEnabledFor(zapcore.DebugLevel) {
			c.Logger.Debugf("fetching wallet identity from backend after a timeout [%s] took [%v]", identity, time.Since(start))
		}
	case <-ctx.Done():
		return nil, errors.New("context is done")
	}
	span.AddEvent("got_recipient_data")

	return identity, nil
}

func (c *IdentityCache) provisionIdentities() {
	ctx := context.Background()
	for {
		id, err := c.backed(ctx)
		if err != nil {
			continue
		}
		c.cache <- id
	}
}
