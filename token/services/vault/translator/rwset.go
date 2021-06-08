/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package translator

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
)

//go:generate counterfeiter -o mock/rwset.go -fake-name RWSet . RWSet

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string, opts ...fabric.GetStateOpt) ([]byte, error)
	DeleteState(namespace string, key string) error
	Bytes() ([]byte, error)
	Done()
	GetStateMetadata(namespace, key string, opts ...fabric.GetStateOpt) (map[string][]byte, error)
	SetStateMetadata(namespace, key string, metadata map[string][]byte) error
	AppendRWSet(raw []byte, nss ...string) error
	GetReadAt(ns string, i int) (string, []byte, error)
	GetWriteAt(ns string, i int) (string, []byte, error)
	NumReads(ns string) int
	NumWrites(ns string) int
	Namespaces() []string
}

//go:generate counterfeiter -o mock/rwsetvalidator.go -fake-name RWSetValidator . RWSetValidator

// RWSetValidator interface checks whether the information in the RWSet matches the expected outputs
type RWSetValidator interface {
	Validate(raw []byte) error
}

type ReadOnlyRWSet interface {
	// GetReadAt returns the i-th read (key, value) in the namespace ns  of this rwset.
	// The value is loaded from the ledger, if present. If the key's version in the ledger
	// does not match the key's version in the read, then it returns an error.
	GetReadAt(ns string, i int) (string, []byte, error)

	// GetWriteAt returns the i-th write (key, value) in the namespace ns of this rwset.
	GetWriteAt(ns string, i int) (string, []byte, error)

	// NumReads returns the number of reads in the namespace ns  of this rwset.
	NumReads(ns string) int

	// NumWrites returns the number of writes in the namespace ns of this rwset.
	NumWrites(ns string) int

	// Namespaces returns the namespace labels in this rwset.
	Namespaces() []string
}

type RWSetImporter interface {
	RWSet(raw []byte) (ReadOnlyRWSet, error)
}
