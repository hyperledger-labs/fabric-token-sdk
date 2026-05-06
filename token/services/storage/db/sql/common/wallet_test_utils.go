/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"database/sql"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	sq "github.com/Masterminds/squirrel"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/onsi/gomega"
)

type walletStoreConstructor func(*sql.DB, sq.PlaceholderFormat) *WalletStore

func TestGetWalletID(t *testing.T, store walletStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	roleID := 5
	output := driver.WalletID("my wallet")
	mockDB.
		ExpectQuery(sqlPattern(pf, "SELECT wallet_id FROM WALLETS WHERE (identity_hash = ? AND role_id = ?)")).
		WithArgs(tokenID.UniqueID(), roleID).
		WillReturnRows(mockDB.NewRows([]string{"wallet_id"}).AddRow(output))

	actualWalletID, err := store(db, pf).GetWalletID(t.Context(), tokenID, roleID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(actualWalletID).To(gomega.Equal(output))
}

func TestGetWalletIDs(t *testing.T, store walletStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	roleID := 5
	output := driver.WalletID("my wallet")
	mockDB.
		ExpectQuery(sqlPattern(pf, "SELECT DISTINCT wallet_id FROM WALLETS WHERE role_id = ?")).
		WithArgs(roleID).
		WillReturnRows(mockDB.NewRows([]string{"wallet_id"}).AddRow(output))

	actualWalletIDs, err := store(db, pf).GetWalletIDs(t.Context(), roleID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(actualWalletIDs).To(gomega.ConsistOf(output))
}

func TestLoadMeta(t *testing.T, store walletStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	roleID := 5
	walletID := driver.WalletID("my wallet")
	output := []byte("some meta data")
	mockDB.
		ExpectQuery(sqlPattern(pf, "SELECT meta FROM WALLETS WHERE (identity_hash = ? AND wallet_id = ? AND role_id = ?)")).
		WithArgs(tokenID.UniqueID(), walletID, roleID).
		WillReturnRows(mockDB.NewRows([]string{"meta"}).AddRow(output))

	actual, err := store(db, pf).LoadMeta(t.Context(), tokenID, walletID, roleID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
	gomega.Expect(actual).To(gomega.Equal(output))
}

func TestIdentityExists(t *testing.T, store walletStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	roleID := 5
	walletID := driver.WalletID("my wallet")
	mockDB.
		ExpectQuery(sqlPattern(pf, "SELECT wallet_id FROM WALLETS WHERE (identity_hash = ? AND wallet_id = ? AND role_id = ?)")).
		WithArgs(tokenID.UniqueID(), walletID, roleID).
		WillReturnRows(mockDB.NewRows([]string{"wallet_id"}).AddRow(walletID))

	exists := store(db, pf).IdentityExists(t.Context(), tokenID, walletID, roleID)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(exists).To(gomega.BeTrue())
}

func TestStoreIdentity(t *testing.T, store walletStoreConstructor, pf sq.PlaceholderFormat) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	eID := "5678"
	walletID := driver.WalletID("my wallet")
	roleID := 5

	mockDB.ExpectExec(sqlPattern(pf, "INSERT INTO WALLETS (identity_hash,meta,wallet_id,role_id,created_at,enrollment_id) VALUES (?,?,?,?,?,?) ON CONFLICT DO NOTHING")).
		WithArgs(tokenID.UniqueID(), []uint8(nil), walletID, roleID, sqlmock.AnyArg(), eID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err = store(db, pf).StoreIdentity(t.Context(), tokenID, eID, walletID, roleID, nil)

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
	gomega.Expect(err).ToNot(gomega.HaveOccurred())
}

func TestStoreIdentityIdempotent(t *testing.T, store walletStoreConstructor) {
	gomega.RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	eID := "5678"
	walletID := driver.WalletID("my wallet")
	roleID := 5

	// Use Dollar format for mock patterns — tests idempotency, not SQL dialect
	pf := sq.Dollar
	insertQuery := sqlPattern(pf, "INSERT INTO WALLETS (identity_hash,meta,wallet_id,role_id,created_at,enrollment_id) VALUES (?,?,?,?,?,?) ON CONFLICT DO NOTHING")

	// First call: row inserted (1 row affected)
	mockDB.ExpectExec(insertQuery).
		WithArgs(tokenID.UniqueID(), []uint8(nil), walletID, roleID, sqlmock.AnyArg(), eID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Second call: conflict, 0 rows affected — must still return nil
	mockDB.ExpectExec(insertQuery).
		WithArgs(tokenID.UniqueID(), []uint8(nil), walletID, roleID, sqlmock.AnyArg(), eID).
		WillReturnResult(sqlmock.NewResult(0, 0))

	s := store(db, pf)
	err = s.StoreIdentity(t.Context(), tokenID, eID, walletID, roleID, nil)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	err = s.StoreIdentity(t.Context(), tokenID, eID, walletID, roleID, nil)
	gomega.Expect(err).ToNot(gomega.HaveOccurred())

	gomega.Expect(mockDB.ExpectationsWereMet()).To(gomega.Succeed())
}

func sqlPattern(pf sq.PlaceholderFormat, query string) string {
	replaced, err := pf.ReplacePlaceholders(query)
	if err != nil {
		return query
	}

	return regexp.QuoteMeta(replaced)
}
