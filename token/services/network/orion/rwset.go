/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/services/otx"
	"github.com/hyperledger-labs/orion-server/pkg/types"
)

func orionKey(key string) string {
	return strings.ReplaceAll(key, string(rune(0)), "~")
}

type ReadOnlyRWSWrapper struct {
	qe *orion.SessionQueryExecutor
}

func NewReadOnlyRWSWrapper(qe *orion.SessionQueryExecutor) *ReadOnlyRWSWrapper {
	return &ReadOnlyRWSWrapper{qe: qe}
}

func (r *ReadOnlyRWSWrapper) SetState(namespace string, key string, value []byte) error {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	key = orionKey(key)
	return r.qe.Get(key)
}

func (r *ReadOnlyRWSWrapper) DeleteState(namespace string, key string) error {
	panic("programming error: this should not be called")
}

type TxRWSWrapper struct {
	me string
	db string
	tx *orion.Transaction
}

func NewTxRWSWrapper(me string, db string, tx *orion.Transaction) *TxRWSWrapper {
	return &TxRWSWrapper{me: me, db: db, tx: tx}
}

func (r *TxRWSWrapper) SetState(namespace string, key string, value []byte) error {
	key = orionKey(key)
	return r.tx.Put(
		r.db, key, value,
		&types.AccessControl{
			ReadWriteUsers: otx.UsersMap(r.me),
		},
	)
}

func (r *TxRWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	key = orionKey(key)
	return r.tx.Get(r.db, key)
}

func (r *TxRWSWrapper) DeleteState(namespace string, key string) error {
	key = orionKey(key)
	return r.tx.Delete(r.db, key)
}

type RWSWrapper struct {
	r *orion.RWSet
}

func NewRWSWrapper(r *orion.RWSet) *RWSWrapper {
	return &RWSWrapper{r: r}
}

func (r *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	key = orionKey(key)
	logger.Debugf("check orion key [%s]", key)
	return r.r.GetState(namespace, key, orion.FromIntermediate)
}
