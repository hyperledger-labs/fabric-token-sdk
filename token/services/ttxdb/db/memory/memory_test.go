/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	db2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/memory"
)

func TestMemory(t *testing.T) {
	d := &memory.Driver{}

	for _, c := range dbtest.TokenTransactionDBCases {
		db, err := d.NewOwnerTransaction(db2.MemoryOpts(token.TMSID{Network: c.Name}))
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
