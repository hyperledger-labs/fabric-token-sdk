/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils"
	memory "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/memory"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/kvs"
)

func NewInMemory() (KVS, error) {
	return kvs.New(utils.MustGet(memory.NewDriver().NewKVS("")), "", kvs.DefaultCacheSize)
}
