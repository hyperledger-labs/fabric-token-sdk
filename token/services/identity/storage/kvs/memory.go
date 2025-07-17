/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"context"

	"github.com/IBM/idemix/bccsp/keystore"
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	mem "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/kvs"
)

func NewInMemory() (KVS, error) {
	return kvs.New(utils.MustGet(mem.NewDriver().NewKVS("")), "", kvs.DefaultCacheSize)
}

func Keystore(kvs KVS) keystore.KVS {
	return &kvsAdapter{kvs: kvs}
}

type kvsAdapter struct {
	kvs KVS
}

func (k *kvsAdapter) Put(id string, state interface{}) error {
	return k.kvs.Put(context.Background(), id, state)
}

func (k *kvsAdapter) Get(id string, state interface{}) error {
	return k.kvs.Get(context.Background(), id, state)
}
