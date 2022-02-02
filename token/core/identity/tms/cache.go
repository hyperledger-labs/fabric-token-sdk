/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tms

import (
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"go.uber.org/zap/zapcore"
)

type IdentityCacheBackendFunc func(opts *driver2.IdentityOptions) (view.Identity, []byte, error)

type identityCacheEntry struct {
	Identity view.Identity
	Audit    []byte
}

type IdentityCache struct {
	backed  IdentityCacheBackendFunc
	ch      chan identityCacheEntry
	timeout time.Duration
}

func NewIdentityCache(backed IdentityCacheBackendFunc, size int) *IdentityCache {
	ci := &IdentityCache{
		backed:  backed,
		ch:      make(chan identityCacheEntry, size),
		timeout: time.Millisecond * 100,
	}
	go ci.run()

	return ci
}

func (c *IdentityCache) Identity(opts *driver2.IdentityOptions) (view.Identity, []byte, error) {
	if opts.EIDExtension && len(opts.AuditInfo) == 0 {
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel...")
		}
		select {
		case entry := <-c.ch:
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("fetch identity from producer channel done [%s][%d]", entry.Identity, len(entry.Audit))
			}
			return entry.Identity, entry.Audit, nil
		case <-time.After(c.timeout):
			if logger.IsEnabledFor(zapcore.DebugLevel) {
				logger.Debugf("fetch identity from producer channel timeout")
			}
			return c.backed(opts)
		}

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

func (c *IdentityCache) run() {
	for {
		id, audit, err := c.backed(&driver2.IdentityOptions{EIDExtension: true})
		if err != nil {
			continue
		}
		c.ch <- identityCacheEntry{Identity: id, Audit: audit}
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
	go ci.run()

	return ci
}

func (c *WalletIdentityCache) Identity() (view.Identity, error) {
	select {
	case entry := <-c.ch:
		if logger.IsEnabledFor(zapcore.DebugLevel) {
			logger.Debugf("fetch identity from producer channel done [%s][%d]", entry)
		}
		return entry, nil
	case <-time.After(c.timeout):
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
