/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"context"
	"testing"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/storage/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

type TestTokenDB interface {
	driver.TokenStore

	StoreToken(ctx context.Context, tr driver.TokenRecord, owners []string) error
	GetAllTokenInfos(ctx context.Context, ids []*token.ID) ([][]byte, error)
}

const (
	TST = token.Type("TST")
	ABC = token.Type("ABC")
)

var TokenNotifierCases = []struct {
	Name string
	Fn   func(*testing.T, TestTokenDB, driver.TokenNotifier)
}{
	{"SubscribeStore", TSubscribeStore},
	{"SubscribeStoreDelete", TSubscribeStoreDelete},
	{"SubscribeStoreNoCommit", TSubscribeStoreNoCommit},
	{"SubscribeRead", TSubscribeRead},
}

var tokenRecords = []driver.TokenRecord{
	{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	},
	{
		TxID:           "tx1",
		Index:          1,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	},
}

type tokenSubscriber struct {
	notifier driver.TokenNotifier
}

func (t *tokenSubscriber) Subscribe(f func(operation driver2.Operation, vals map[driver2.ColumnKey]string)) error {
	return t.notifier.Subscribe(f)
}

func TSubscribeStore(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	t.Helper()
	result, err := collectDBEvents(&tokenSubscriber{notifier: notifier})
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	require.NoError(t, err)
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[0], []string{"alice"}))
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[1], []string{"alice"}))
	require.NoError(t, tx.Commit())

	require.NoError(t, result.AssertSize(2))
}

func TSubscribeStoreDelete(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	t.Helper()
	result, err := collectDBEvents(&tokenSubscriber{notifier: notifier})
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	require.NoError(t, err)
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[0], []string{"alice"}))
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[1], []string{"alice"}))
	require.NoError(t, tx.Delete(t.Context(), token.ID{TxId: "tx1", Index: 1}, "alice"))
	require.NoError(t, tx.Commit())

	require.NoError(t, result.AssertSize(3))
}

func TSubscribeStoreNoCommit(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	t.Helper()
	result, err := collectDBEvents(&tokenSubscriber{notifier: notifier})
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	require.NoError(t, err)
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[0], []string{"alice"}))
	require.NoError(t, tx.StoreToken(t.Context(), tokenRecords[1], []string{"alice"}))

	require.NoError(t, result.AssertSize(0))
}

func TSubscribeRead(t *testing.T, db TestTokenDB, notifier driver.TokenNotifier) {
	t.Helper()
	result, err := collectDBEvents(&tokenSubscriber{notifier: notifier})
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction()
	require.NoError(t, err)
	// require.NoError(t, tx.StoreToken(t.Context(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	_, _, err = tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, true)
	require.NoError(t, err)
	require.NoError(t, tx.Commit())

	require.NoError(t, result.AssertSize(0))
}
