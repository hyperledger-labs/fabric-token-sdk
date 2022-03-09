/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion"
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/services/otx"
	"github.com/hyperledger-labs/orion-server/pkg/types"
)

type ReadOnlyRWSWrapper struct {
	qe *orion.SessionQueryExecutor
}

func (r *ReadOnlyRWSWrapper) SetState(namespace string, key string, value []byte) error {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	res, _, err := r.qe.Get(key)
	return res, err
}

func (r *ReadOnlyRWSWrapper) DeleteState(namespace string, key string) error {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) Bytes() ([]byte, error) {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) Done() {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	panic("programming error: this should not be called")
}

func (r *ReadOnlyRWSWrapper) Equals(right interface{}, namespace string) error {
	panic("programming error: this should not be called")
}

type RWSWrapper struct {
	me string
	db string
	tx *orion.Transaction
}

func (r *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	return r.tx.Put(
		r.db, key, value,
		&types.AccessControl{
			ReadWriteUsers: otx.UsersMap(r.me),
		},
	)
}

func (r *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	res, _, err := r.tx.Get(r.db, key)
	return res, err
}

func (r *RWSWrapper) DeleteState(namespace string, key string) error {
	return r.tx.Delete(r.db, key)
}

func (r *RWSWrapper) Bytes() ([]byte, error) {
	return nil, nil
}

func (r *RWSWrapper) Done() {
	return
}

func (r *RWSWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return nil
}

func (r *RWSWrapper) Equals(right interface{}, namespace string) error {
	panic("implement me")
}
