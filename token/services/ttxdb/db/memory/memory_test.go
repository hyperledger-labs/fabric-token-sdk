/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package memory

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttxdb/db/dbtest"
)

type MockServiceProvider struct{}

func (sp MockServiceProvider) GetService(v interface{}) (interface{}, error) {
	return v, nil
}

func TestMemory(t *testing.T) {
	d := new(Driver)

	for _, c := range dbtest.Cases {
		db, err := d.Open(new(MockServiceProvider), c.Name)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}
