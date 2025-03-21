/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"fmt"
	"path"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/dbtest"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/driver/sql"
	sqlite2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/sqlite"
)

func TestIdentitySqlite(t *testing.T) {
	for _, c := range dbtest.IdentityCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   fmt.Sprintf("file:%s?_pragma=busy_timeout(20000)", path.Join(t.TempDir(), "db.sqlite")),
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite2.NewIdentityDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestIdentitySqliteMemory(t *testing.T) {
	for _, c := range dbtest.IdentityCases {
		db, err := sql.OpenSqlite(common.Opts{
			DataSource:   "file:tmp?_pragma=busy_timeout(20000)&mode=memory&cache=shared",
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite2.NewIdentityDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}

func TestIdentityPostgres(t *testing.T) {
	terminate, pgConnStr := common.StartPostgresContainer(t)
	defer terminate()

	for _, c := range dbtest.IdentityCases {
		db, err := sql.OpenPostgres(common.Opts{
			DataSource:   pgConnStr,
			TablePrefix:  c.Name,
			MaxOpenConns: 10,
		}, sqlite2.NewIdentityDB)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			c.Fn(xt, db)
		})
	}
}
