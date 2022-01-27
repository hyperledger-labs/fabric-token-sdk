/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"go.uber.org/zap/zapcore"
)

type CacheIdentityBackendFunc func(opts *driver2.IdentityOptions) (view.Identity, []byte, error)

type CacheEntry struct {
	Identity view.Identity
	Audit    []byte
}

type CacheIdentity struct {
	backed CacheIdentityBackendFunc
	ch     chan CacheEntry
}

func NewCacheIdentity(backed CacheIdentityBackendFunc, size int) *CacheIdentity {
	ci := &CacheIdentity{backed: backed, ch: make(chan CacheEntry, size)}
	go ci.run()

	return ci
}

func (c *CacheIdentity) Identity(opts *driver2.IdentityOptions) (view.Identity, []byte, error) {
	if opts.EIDExtension && len(opts.AuditInfo) == 0 {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel...")
		}
		entry := <-c.ch
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel done [%s][%d]", entry.Identity, len(entry.Audit))
		}
		return entry.Identity, entry.Audit, nil
	}
	if logger.IsEnabledFor(zapcore.DebugLevel) {
		logger.Debugf("fetch identity from backend...")
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

func (c *CacheIdentity) run() {
	for {
		id, audit, err := c.backed(&driver2.IdentityOptions{EIDExtension: true})
		if err != nil {
			continue
		}
		c.ch <- CacheEntry{Identity: id, Audit: audit}
	}
}
