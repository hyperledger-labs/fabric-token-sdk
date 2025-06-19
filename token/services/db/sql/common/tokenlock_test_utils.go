/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/onsi/gomega"
)

type tokenLockStoreConstructor func(*sql.DB) *TokenLockStore

func TestLock(t *testing.T, store tokenLockStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.ID{TxId: "1234", Index: 5}
	trID := "5555"
	now := sqlmock.AnyArg()

	mockDB.
		ExpectExec("INSERT INTO TOKEN_LOCKS \\(consumer_tx_id, tx_id, idx, created_at\\) VALUES \\(\\$1, \\$2, \\$3, \\$4\\)").
		WithArgs(trID, tokenID.TxId, tokenID.Index, now).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store(db).Lock(context.Background(), &tokenID, trID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func TestUnlockByTxID(t *testing.T, store tokenLockStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	consumerTxID := "1234"

	mockDB.
		ExpectExec("DELETE FROM TOKEN_LOCKS WHERE consumer_tx_id = \\$1").
		WithArgs(consumerTxID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = store(db).UnlockByTxID(context.Background(), consumerTxID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}
