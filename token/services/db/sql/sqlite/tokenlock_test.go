/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"
	"time"

	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/onsi/gomega"
)

func mockTokenLockStore(db *sql.DB) *common.TokenLockStore {
	var dbs = common2.RWDB{
		ReadDB: db, WriteDB: db,
	}

	store, _ := NewTokenLockStore(&dbs, common.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	})
	return store.TokenLockStore
}

func TestIsStale(t *testing.T) {
	gomega.RegisterTestingT(t)

	query, args := q.DeleteFrom("TokenLocks").
		Where(IsStale("TokenLocks", "Requests", 5*time.Second)).
		Format(sqlite.NewConditionInterpreter())

	gomega.Expect(query).To(gomega.Equal("DELETE FROM TokenLocks WHERE tx_id IN (" +
		"SELECT tl.tx_id " +
		"FROM TokenLocks AS tl " +
		"LEFT JOIN Requests AS tr " +
		"ON tl.tx_id = tr.tx_id " +
		"WHERE (tr.status = $1) OR (tl.created_at < datetime('now', '-5 seconds'))" +
		")"))
	gomega.Expect(args).To(gomega.ConsistOf(driver.Deleted))
}

func TestLock(t *testing.T) {
	common.TestLock(t, mockTokenLockStore)
}

func TestUnlockByTxID(t *testing.T) {
	common.TestUnlockByTxID(t, mockTokenLockStore)
}
