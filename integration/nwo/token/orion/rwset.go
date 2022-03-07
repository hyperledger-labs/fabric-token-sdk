/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package orion

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/services/otx"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/hyperledger-labs/orion-server/pkg/types"
)

type rwsWrapper struct {
	me string
	db string
	tx bcdb.DataTxContext
}

func (r *rwsWrapper) SetState(namespace string, key string, value []byte) error {
	return r.tx.Put(
		r.db, key, value,
		&types.AccessControl{
			ReadWriteUsers: otx.UsersMap(r.me),
		},
	)
}

func (r *rwsWrapper) GetState(namespace string, key string) ([]byte, error) {
	res, _, err := r.tx.Get(r.db, key)
	return res, err
}

func (r *rwsWrapper) DeleteState(namespace string, key string) error {
	return r.tx.Delete(r.db, key)
}

func (r *rwsWrapper) Bytes() ([]byte, error) {
	return nil, nil
}

func (r *rwsWrapper) Done() {
	return
}

func (r *rwsWrapper) SetStateMetadata(namespace, key string, metadata map[string][]byte) error {
	return nil
}

func (r *rwsWrapper) Equals(right interface{}, namespace string) error {
	panic("implement me")
}
