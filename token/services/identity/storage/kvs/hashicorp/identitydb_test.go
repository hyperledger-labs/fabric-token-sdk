/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashicorp_test

import (
	"fmt"
	"testing"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/storage/kvs/hashicorp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/dbtest"
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
		db := kvs.NewIdentityStore(backend, token2.TMSID{
			Network:   "apple",
			Channel:   "pears",
			Namespace: "strawberries",
		})
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}
