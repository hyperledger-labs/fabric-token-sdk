/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"database/sql"
	driver2 "database/sql/driver"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func mockTokenLockStorePostgress(db *sql.DB) *TokenLockStore {
	var dbs = common2.RWDB{
		ReadDB: db, WriteDB: db,
	}

	store, _ := NewTokenLockStore(&dbs, common3.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	})

	return store
}

func mockTokenLockStore(db *sql.DB, pf sq.PlaceholderFormat) *common3.TokenLockStore {
	store, _ := common3.NewTokenLockStore(db, db, common3.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	}, pf)

	return store
}

func TestCleanup(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	input := driver.Deleted

	consumerTxID := "1234"
	tokenID := token.ID{TxId: "5678", Index: 5}
	status := &input
	createdAt := time.Date(2025, time.June, 8, 10, 0, 0, 0, time.UTC)

	var timeNow time.Time
	output := []driver2.Value{
		consumerTxID, tokenID.TxId, tokenID.Index, *status, createdAt, timeNow,
	}
	mockDB.
		ExpectQuery(regexp.QuoteMeta(
			"SELECT tl.consumer_tx_id, tl.tx_id, tl.idx, tr.status, tl.created_at, NOW() AS now"+
				" FROM TOKEN_LOCKS AS tl"+
				" LEFT JOIN REQUESTS AS tr ON tl.consumer_tx_id = tr.tx_id"+
				" WHERE (tr.status = $1 OR tl.created_at < $2)",
		)).
		WithArgs(input, sqlmock.AnyArg()).
		WillReturnRows(mockDB.NewRows([]string{"consumer_tx_id", "tx_id", "idx", "status", "created_at", "now"}).AddRow(output...))

	mockDB.ExpectExec(regexp.QuoteMeta(
		"DELETE FROM TOKEN_LOCKS" +
			" WHERE (created_at < $1" +
			" OR EXISTS (SELECT 1 FROM REQUESTS WHERE REQUESTS.tx_id = TOKEN_LOCKS.consumer_tx_id AND REQUESTS.status IN (3)))",
	)).
		WithArgs(sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = mockTokenLockStorePostgress(db).Cleanup(t.Context(), time.Second)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
}

func TestLock(t *testing.T) {
	common3.TestLock(t, mockTokenLockStore, sq.Dollar)
}

func TestUnlockByTxID(t *testing.T) {
	common3.TestUnlockByTxID(t, mockTokenLockStore, sq.Dollar)
}
