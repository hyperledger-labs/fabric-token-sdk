/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"context"
	"database/sql"
	driver2 "database/sql/driver"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	common2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/common"
	q "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/query"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/postgres"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	. "github.com/onsi/gomega"
)

func mockTokenLockStore(db *sql.DB) *postgres.TokenLockStore {
	var dbs = common2.RWDB{
		ReadDB: db, WriteDB: db,
	}

	val, _ := postgres.NewTokenLockStore(&dbs, common.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	})
	return val
}

func TestIsStale(t *testing.T) {
	RegisterTestingT(t)

	query, args := q.DeleteFrom("TokenLocks").
		Where(IsStale("TokenLocks", "Requests", 5*time.Second)).
		Format(sqlite.NewConditionInterpreter())

	Expect(query).To(Equal("DELETE FROM TokenLocks WHERE tx_id IN (" +
		"SELECT tl.tx_id " +
		"FROM TokenLocks AS tl " +
		"LEFT JOIN Requests AS tr " +
		"ON tl.tx_id = tr.tx_id " +
		"WHERE (tr.status = $1) OR (tl.created_at < datetime('now', '-5 seconds'))" +
		")"))
	Expect(args).To(ConsistOf(driver.Deleted))
}

func TestLogStaleLocks(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	input := driver.TxStatus(3)

	record := postgres.LockEntry{
		ConsumerTxID: "1234",
		TokenID:      token.ID{TxId: "5678", Index: 5},
		Status:       &input,
		CreatedAt:    time.Date(2025, time.June, 8, 10, 0, 0, 0, time.UTC),
	}
	var timeNow time.Time
	output := []driver2.Value{
		record.ConsumerTxID, record.TokenID.TxId, record.TokenID.Index, *record.Status, record.CreatedAt, timeNow,
	}
	mockDB.
		ExpectQuery("SELECT TOKEN_LOCKS.consumer_tx_id, TOKEN_LOCKS.tx_id, TOKEN_LOCKS.idx, REQUESTS.status, TOKEN_LOCKS.created_at, NOW\\(\\) AS now " +
			"FROM TOKEN_LOCKS LEFT JOIN REQUESTS ON TOKEN_LOCKS.consumer_tx_id = REQUESTS.tx_id " +
			"WHERE \\(\\(\\(REQUESTS.status = \\$1\\)\\)\\) OR \\(TOKEN_LOCKS.created_at < NOW\\(\\) - INTERVAL '1 seconds'\\)").
		WithArgs(input).
		WillReturnRows(mockDB.NewRows([]string{"consumer_tx_id", "tx_id", "idx", "status", "created_at", "now"}).AddRow(output...))

	records, err := mockTokenLockStore(db).LogStaleLocks(context.Background(), time.Second)
	record.LeaseExpiry = records[0].LeaseExpiry

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	Expect(records[0]).To(Equal(record))
}
