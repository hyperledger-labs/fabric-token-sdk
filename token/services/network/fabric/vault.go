/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabric

import (
	"runtime/debug"

	"github.com/hashicorp/go-uuid"
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network/processor"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/vault/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

type Vault struct {
	ch         *fabric.Channel
	tokenStore processor.TokenStore
}

func NewVault(ch *fabric.Channel, tokenStore processor.TokenStore) *Vault {
	return &Vault{
		ch:         ch,
		tokenStore: tokenStore,
	}
}

func (v *Vault) NewQueryExecutor() (driver.Executor, error) {
	qe, err := v.ch.Vault().NewQueryExecutor()
	if err != nil {
		return nil, err
	}
	return &Executor{qe: qe}, nil
}

func (v *Vault) NewRWSet(txID string) (driver.RWSet, error) {
	rws, err := v.ch.Vault().NewRWSet(txID)
	if err != nil {
		return nil, err
	}
	return NewRWSWrapper(rws), nil
}

func (v *Vault) DeleteTokens(ns string, ids ...*token.ID) error {
	// prepare a rws with deletes
	id, err := uuid.GenerateUUID()
	if err != nil {
		return errors.Wrapf(err, "failed to generated uuid")
	}
	txID := "delete_" + id
	rws, err := v.ch.Vault().NewRWSet(txID)
	if err != nil {
		return err
	}

	wrappedRWS := &rwsWrapper{RWSet: rws}
	for _, id := range ids {
		if err := v.tokenStore.DeleteFabToken(ns, id.TxId, id.Index, wrappedRWS, string(debug.Stack())); err != nil {
			return errors.Wrapf(err, "failed to append deletion of [%s]", id)
		}
	}
	rws.Done()

	if err := v.ch.Vault().CommitTX(txID, 0, 0); err != nil {
		return errors.WithMessagef(err, "failed to commit rws with token delitions")
	}

	return nil
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
