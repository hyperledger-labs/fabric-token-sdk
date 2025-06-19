/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package postgres

import (
	"context"
	"database/sql"
	driver2 "database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/onsi/gomega"
)

func mockTokenLockStorePostgress(db *sql.DB) *TokenLockStore {
	var dbs = common2.RWDB{
		ReadDB: db, WriteDB: db,
	}

	store, _ := NewTokenLockStore(&dbs, common.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	})
	return store
}

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

func TestCleanup(t *testing.T) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

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
		ExpectQuery("SELECT TOKEN_LOCKS.consumer_tx_id, TOKEN_LOCKS.tx_id, TOKEN_LOCKS.idx, REQUESTS.status, TOKEN_LOCKS.created_at, NOW\\(\\) AS now " +
			"FROM TOKEN_LOCKS LEFT JOIN REQUESTS ON TOKEN_LOCKS.consumer_tx_id = REQUESTS.tx_id " +
			"WHERE \\(\\(\\(REQUESTS.status = \\$1\\)\\)\\) OR \\(TOKEN_LOCKS.created_at < NOW\\(\\) - INTERVAL '1 seconds'\\)").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"consumer_tx_id", "tx_id", "idx", "status", "created_at", "now"}).AddRow(output...))

	mockDB.ExpectExec("DELETE FROM TOKEN_LOCKS USING REQUESTS WHERE " +
		"\\(TOKEN_LOCKS.consumer_tx_id = REQUESTS.tx_id\\) AND " +
		"\\(\\(\\(\\(REQUESTS.status = \\$1\\)\\)\\) OR \\(TOKEN_LOCKS.created_at < NOW\\(\\) - INTERVAL '1 seconds'\\)\\)").
		WithArgs(input).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = mockTokenLockStorePostgress(db).Cleanup(context.Background(), time.Second)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func TestLock(t *testing.T) {
	common.TestLock(t, mockTokenLockStore)
}

func TestUnlockByTxID(t *testing.T) {
	common.TestUnlockByTxID(t, mockTokenLockStore)
}
