/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package db

import (
	driver3 "github.com/hyperledger-labs/fabric-smart-client/platform/common/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
)

func NewDriverHolder(cp driver.ConfigProvider, ds ...driver.NamedDriver) *DriverHolder {
	drivers := make(map[driver3.PersistenceType]driver.Driver, len(ds))
	for _, d := range ds {
		drivers[d.Name] = d.Driver
	}
	return &DriverHolder{drivers: drivers, cp: cp}
}

type DriverHolder struct {
	drivers map[driver3.PersistenceType]driver.Driver
	cp      driver.ConfigProvider
}

func (h *DriverHolder) NewTokenLockManager(keys ...string) *Manager[driver.TokenLockDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.TokenLockDB] { return d.NewTokenLock })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewWalletManager(keys ...string) *Manager[driver.WalletDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.WalletDB] { return d.NewWallet })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewIdentityManager(keys ...string) *Manager[driver.IdentityDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.IdentityDB] { return d.NewIdentity })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewTokenManager(keys ...string) *Manager[driver.TokenDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.TokenDB] { return d.NewToken })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewTokenNotifierManager(keys ...string) *Manager[driver.TokenNotifier] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.TokenNotifier] { return d.NewTokenNotifier })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewAuditTransactionManager(keys ...string) *Manager[driver.AuditTransactionDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.AuditTransactionDB] { return d.NewAuditTransaction })
	return NewManager(h.cp, openers, keys...)
}

func (h *DriverHolder) NewOwnerTransactionManager(keys ...string) *Manager[driver.TokenTransactionDB] {
	openers := transform(h.drivers, func(d driver.Driver) sql.Opener[driver.TokenTransactionDB] { return d.NewOwnerTransaction })
	return NewManager(h.cp, openers, keys...)
}

func transform[K comparable, S any, T any](ds map[K]S, transformer func(S) T) map[K]T {
	r := make(map[K]T, len(ds))
	for k, v := range ds {
		r[k] = transformer(v)
	}
	return r
}
