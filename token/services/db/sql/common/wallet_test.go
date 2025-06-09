/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package common_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver/sql/sqlite"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	. "github.com/onsi/gomega"
)

func mockTransactionsStore(db *sql.DB) *common.WalletStore {
	val, _ := common.NewWalletStore(db, db, common.TableNames{
		Wallets: "WALLETS",
	}, sqlite.NewConditionInterpreter())
	return val
}

func TestGetWalletID(t *testing.T) {
	RegisterTestingT(t)
	db, mockDB, err := sqlmock.New()
	Expect(err).ToNot(HaveOccurred())

	tokenID := token.Identity([]byte("1234"))
	roleID := 5
	output := driver.WalletID("my wallet")
	mockDB.
		ExpectQuery("SELECT wallet_id FROM WALLETS WHERE \\(identity_hash = \\$1\\) AND \\(role_id = \\$2\\)").
		WithArgs(tokenID.UniqueID(), roleID).
		WillReturnRows(mockDB.NewRows([]string{"request"}).AddRow(output))

	actualWalletID, err := mockTransactionsStore(db).GetWalletID(context.Background(), tokenID, roleID)

	Expect(mockDB.ExpectationsWereMet()).To(Succeed())
	Expect(err).ToNot(HaveOccurred())
	Expect(actualWalletID).To(Equal(output))
}
