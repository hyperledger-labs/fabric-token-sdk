/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"
	"time"

	q "github.com/LFDT-Panurus/panurus/token/services/storage/db/sql/query"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"

	"github.com/LFDT-Panurus/panurus/token/services/storage/db/driver"
	common3 "github.com/LFDT-Panurus/panurus/token/services/storage/db/sql/common"
	. "github.com/onsi/gomega"
)

func mockTokenLockStore(db *sql.DB) *common3.TokenLockStore {
	var dbs = common2.RWDB{
		ReadDB: db, WriteDB: db,
	}

	store, _ := NewTokenLockStore(&dbs, common3.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Tokens:     "TOKENS",
		Requests:   "REQUESTS",
	})

	return store.TokenLockStore
}

func TestIsStale(t *testing.T) {
	RegisterTestingT(t)

	query, args := q.DeleteFrom("TokenLocks").
		Where(IsStale("TokenLocks", "Requests", 5*time.Second)).
		Format(NewConditionInterpreter())

	Expect(query).To(Equal("DELETE FROM TokenLocks WHERE tx_id IN (" +
		"SELECT tl.tx_id " +
		"FROM TokenLocks AS tl " +
		"LEFT JOIN Requests AS tr " +
		"ON tl.tx_id = tr.tx_id " +
		"WHERE (tr.status = $1) OR (tl.created_at < datetime('now', '-5 seconds'))" +
		")"))
	Expect(args).To(ConsistOf(driver.Deleted))
}

func TestLock(t *testing.T) {
	common3.TestLock(t, mockTokenLockStore)
}

func TestUnlockByTxID(t *testing.T) {
	common3.TestUnlockByTxID(t, mockTokenLockStore)
}
