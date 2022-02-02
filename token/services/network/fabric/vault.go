/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
)

type Vault struct {
	ch *fabric.Channel
}

func NewVault(ch *fabric.Channel) *Vault {
	return &Vault{ch: ch}
}

func (v *Vault) NewQueryExecutor() (driver.Executor, error) {
	qe, err := v.ch.Vault().NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	return &Executor{qe: qe}, nil
}

func (v *Vault) NewRWSet(txid string) (driver.RWSet, error) {
	rws, err := v.ch.Vault().NewRWSet(txid)
	if err != nil {
		return nil, err
	}
	return NewRWSWrapper(rws), nil
}

type Executor struct {
	qe *fabric.QueryExecutor
}

func (e *Executor) Done() {
	e.qe.Done()
}

func (e *Executor) GetState(namespace string, key string) ([]byte, error) {
	return e.qe.GetState(namespace, key)
}

func (e *Executor) GetStateRangeScanIterator(namespace string, start string, end string) (driver.Iterator, error) {
	it, err := e.qe.GetStateRangeScanIterator(namespace, start, end)
	if err != nil {
		return nil, err
	}
	return &Iterator{it: it}, nil
}

func (e *Executor) GetCachedStateRangeScanIterator(namespace string, start string, end string) (driver.Iterator, error) {
	it, err := e.qe.GetCachedStateRangeScanIterator(namespace, start, end)
	if err != nil {
		return nil, err
	}
	return &Iterator{it: it}, nil
}

func (e *Executor) GetStateMetadata(namespace string, id string) (map[string][]byte, error) {
	r, _, _, err := e.qe.GetStateMetadata(namespace, id)
	return r, err
}

type Iterator struct {
	it *fabric.ResultsIterator
}

func (i *Iterator) Close() {
	i.it.Close()
}

func (i *Iterator) Next() (driver.Entry, error) {
	r, err := i.it.Next()
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, nil
	}
	return r, nil
}
