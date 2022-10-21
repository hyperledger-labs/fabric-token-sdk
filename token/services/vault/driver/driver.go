/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import "github.com/hyperledger-labs/fabric-token-sdk/token/token"

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	DeleteState(namespace string, key string) error
	Equals(rwset interface{}, namespace string) error
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
	NewQueryExecutor() (Executor, error)
	DeleteTokens(ns string, ids ...*token.ID) error
}
