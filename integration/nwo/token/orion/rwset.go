/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package orion

import (
	"strings"

	"github.com/hyperledger-labs/fabric-smart-client/platform/orion/services/otx"
	"github.com/hyperledger-labs/orion-sdk-go/pkg/bcdb"
	"github.com/hyperledger-labs/orion-server/pkg/types"
)

type RWSWrapper struct {
	me string
	db string
	tx bcdb.DataTxContext
}

func orionKey(key string) string {
	return strings.ReplaceAll(key, string(rune(0)), "~")
}

func (r *RWSWrapper) SetState(namespace string, key string, value []byte) error {
	key = orionKey(key)
	return r.tx.Put(
		r.db, key, value,
		&types.AccessControl{
			ReadWriteUsers: otx.UsersMap(r.me),
		},
	)
}

func (r *RWSWrapper) GetState(namespace string, key string) ([]byte, error) {
	key = orionKey(key)
	res, _, err := r.tx.Get(r.db, key)
	return res, err
}

func (r *RWSWrapper) DeleteState(namespace string, key string) error {
	key = orionKey(key)
	return r.tx.Delete(r.db, key)
}

func (r *RWSWrapper) Bytes() ([]byte, error) {
	return nil, nil
}

func (r *RWSWrapper) Done() {
}

func (r *RWSWrapper) Equals(right interface{}, namespace string) error {
	panic("implement me")
}
