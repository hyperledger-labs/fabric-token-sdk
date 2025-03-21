/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"fmt"
	"testing"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs/hashicorp"
	"github.com/stretchr/testify/assert"
)

func TestIdentityDBWithHashicorpVault(t *testing.T) {
	terminate, vaultURL, token := hashicorp.StartHashicorpVaultContainer(t, 11200)
	defer terminate()
	client, err := hashicorp.NewVaultClient(vaultURL, token)
	assert.NoError(t, err)

	for i, c := range dbtest.IdentityCases {
		backend, err := hashicorp.NewWithClient(client, fmt.Sprintf("kv1/data/token-sdk/%d/", i))
		assert.NoError(t, err)
		db := NewIdentityDB(backend, token2.TMSID{
			Network:   "apple",
			Channel:   "pears",
			Namespace: "strawberries",
		})
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestIdentityDBWithInMEmoryKVS(t *testing.T) {
	for _, c := range dbtest.IdentityCases {
		backend, err := NewInMemory()
		assert.NoError(t, err)
		db := NewIdentityDB(backend, token2.TMSID{
			Network:   "apple",
			Channel:   "pears",
			Namespace: "strawberries",
		})
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}
