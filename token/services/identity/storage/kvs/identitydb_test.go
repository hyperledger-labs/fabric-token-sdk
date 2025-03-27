/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package kvs

import (
	"testing"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/stretchr/testify/assert"
)

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
