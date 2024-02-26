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
	"github.com/pkg/errors"
)

type ReadOnlyRWSWrapper struct {
	qe *orion.SessionQueryExecutor
}

func orionKey(key string) string {
	return strings.ReplaceAll(key, string(rune(0)), "~")
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

func (r *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	key = orionKey(key)
	return r.r.SetState(namespace, key, value)
}

func (r *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	key = orionKey(key)
	return r.r.GetState(namespace, key)
}

func (r *RWSWrapper) DeleteState(namespace string, key string) error {
	key = orionKey(key)
	return r.r.DeleteState(namespace, key)
}

func (r *RWSWrapper) Bytes() ([]byte, error) {
	return r.r.Bytes()
}

func (r *RWSWrapper) Done() {
	r.r.Done()
}

func (r *RWSWrapper) Equals(right interface{}, namespace string) error {
	switch t := right.(type) {
	case *RWSWrapper:
		return r.r.Equals(t.r, namespace)
	case *orion.RWSet:
		return r.r.Equals(t, namespace)
	default:
		return errors.Errorf("invalid type, got [%T]", t)
	}
}
