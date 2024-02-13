/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

type ValidationCode = driver.ValidationCode

const (
	_               ValidationCode = iota
	Valid                          = driver.Valid           // Transaction is valid and committed
	Invalid                        = driver.Invalid         // Transaction is invalid and has been discarded
	Busy                           = driver.Busy            // Transaction does not yet have a validity state
	Unknown                        = driver.Unknown         // Transaction is unknown
	HasDependencies                = driver.HasDependencies // Transaction is unknown but has known dependencies
)

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	DeleteState(namespace string, key string) error
}

type Entry interface {
	K() string
	V() []byte
}

type Iterator interface {
	Close()
	Next() (Entry, error)
}

type Executor interface {
	Done()
	GetState(namespace string, key string) ([]byte, error)
	GetStateRangeScanIterator(namespace string, s string, e string) (Iterator, error)
	GetStateMetadata(namespace string, id string) (map[string][]byte, error)
}

type Vault interface {
	driver.Vault
	NewQueryExecutor() (Executor, error)
	DeleteTokens(ns string, ids ...*token.ID) error
}
