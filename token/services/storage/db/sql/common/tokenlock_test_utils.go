/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/onsi/gomega"
)

type tokenLockStoreConstructor func(*sql.DB, sq.PlaceholderFormat) *TokenLockStore

func TestLock(t *testing.T, store tokenLockStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.ID{TxId: "1234", Index: 5}
	trID := "5555"
	now := sqlmock.AnyArg()

	mockDB.
		ExpectExec(sqlPattern(pf, "INSERT INTO TOKEN_LOCKS (consumer_tx_id,tx_id,idx,created_at) VALUES (?,?,?,?)")).
		WithArgs(trID, tokenID.TxId, tokenID.Index, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store(db, pf).Lock(t.Context(), &tokenID, trID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func TestUnlockByTxID(t *testing.T, store tokenLockStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	consumerTxID := "1234"

	mockDB.
		ExpectExec(sqlPattern(pf, "DELETE FROM TOKEN_LOCKS WHERE consumer_tx_id = ?")).
		WithArgs(consumerTxID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store(db, pf).UnlockByTxID(t.Context(), consumerTxID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
