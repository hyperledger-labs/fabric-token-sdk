/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package translator

import (
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
)

//go:generate counterfeiter -o mock/rwset.go -fake-name RWSet . RWSet

// RWSet interface, used to read from, and write to, a rwset.
type RWSet interface {
	SetState(namespace string, key string, value []byte) error
	GetState(namespace string, key string) ([]byte, error)
	DeleteState(namespace string, key string) error
}

// ExRWSet interface, used to manipulate the rwset in a more friendly way
type ExRWSet interface {
	// SetState adds a write entry to the rwset that write to given value to given key.
	SetState(key string, value []byte) error
	// GetState returns the value bound to the passed key
	GetState(key string) ([]byte, error)
	// DeleteState adds a write entry to the rwset that deletes the passed key
	DeleteState(key string) error
	// AddStateMustNotExist adds a read dependency that enforces that the passed key does not exist
	AddStateMustNotExist(key string) error
	// AddStateMustExist adds a read dependency that enforces that the passed key does exist
	AddStateMustExist(key string) error
}

type RWSetWrapper struct {
	RWSet     RWSet
	Namespace string
	TxID      string
}

func NewRWSetWrapper(RWSet RWSet, namespace string, txID string) *RWSetWrapper {
	return &RWSetWrapper{RWSet: RWSet, Namespace: namespace, TxID: txID}
}

func (w *RWSetWrapper) SetState(key string, value []byte) error {
	return w.RWSet.SetState(w.Namespace, key, value)
}

func (w *RWSetWrapper) GetState(key string) ([]byte, error) {
	return w.RWSet.GetState(w.Namespace, key)
}

func (w *RWSetWrapper) DeleteState(key string) error {
	return w.RWSet.DeleteState(w.Namespace, key)
}

func (w *RWSetWrapper) AddStateMustNotExist(key string) error {
	tr, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(tr) != 0 {
		return errors.Errorf("state [%s:%s] already exists for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}

func (w *RWSetWrapper) AddStateMustExist(key string) error {
	h, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(h) == 0 {
		return errors.Errorf("state [%s:%s] does not exist for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}
