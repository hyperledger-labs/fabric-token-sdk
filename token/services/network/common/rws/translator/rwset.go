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

type ExRWSet interface {
	SetState(key string, value []byte) error
	GetState(key string) ([]byte, error)
	DeleteState(key string) error
	AddStateMustNotExist(key string) error
	AddStateMustExist(key string) error
}

type ExRWSetWrapper struct {
	RWSet     RWSet
	Namespace string
	TxID      string
}

func NewExRWSetWrapper(RWSet RWSet, namespace string, txID string) *ExRWSetWrapper {
	return &ExRWSetWrapper{RWSet: RWSet, Namespace: namespace, TxID: txID}
}

func (w *ExRWSetWrapper) SetState(key string, value []byte) error {
	return w.RWSet.SetState(w.Namespace, key, value)
}

func (w *ExRWSetWrapper) GetState(key string) ([]byte, error) {
	return w.RWSet.GetState(w.Namespace, key)
}

func (w *ExRWSetWrapper) DeleteState(key string) error {
	return w.RWSet.DeleteState(w.Namespace, key)
}

func (w *ExRWSetWrapper) AddStateMustNotExist(key string) error {
	tr, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(tr) != 0 {
		return errors.Errorf("state [%s:%s] already exists for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}

func (w *ExRWSetWrapper) AddStateMustExist(key string) error {
	h, err := w.RWSet.GetState(w.Namespace, key)
	if err != nil {
		return errors.Wrapf(err, "failed to read state [%s:%s] for [%s]", w.Namespace, key, w.TxID)
	}
	if len(h) == 0 {
		return errors.Errorf("state [%s:%s] does not exist for [%s]", w.Namespace, key, w.TxID)
	}
	return nil
}
