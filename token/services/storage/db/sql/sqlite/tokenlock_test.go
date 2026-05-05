/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sqlite

import (
	"database/sql"
	"testing"

	sq "github.com/Masterminds/squirrel"
	common3 "github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/sql/common"
	. "github.com/onsi/gomega"
)

func mockTokenLockStore(db *sql.DB, pf sq.PlaceholderFormat) *common3.TokenLockStore {
	store, _ := common3.NewTokenLockStore(db, db, common3.TableNames{
		TokenLocks: "TOKEN_LOCKS",
		Requests:   "REQUESTS",
	}, pf)

	return store
}

func TestLock(t *testing.T) {
	RegisterTestingT(t)
	common3.TestLock(t, mockTokenLockStore, sq.Question)
}

func TestUnlockByTxID(t *testing.T) {
	RegisterTestingT(t)
	common3.TestUnlockByTxID(t, mockTokenLockStore, sq.Question)
}
