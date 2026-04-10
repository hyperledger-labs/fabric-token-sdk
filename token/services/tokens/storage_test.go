/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

func TestDBStorage(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockPub := &mockPublisher{}
	mockDB := &mockTokenDB{}

	storage, err := NewDBStorage(mockPub, &tokendb.StoreService{TokenStore: mockDB}, tmsID)
	require.NoError(t, err)
	assert.NotNil(t, storage)

	// Test TransactionExists
	mockDB.TransactionExistsReturns = true
	exists, err := storage.TransactionExists(ctx, "tx1")
	require.NoError(t, err)
	assert.True(t, exists)

	// Test StorePublicParams
	err = storage.StorePublicParams(ctx, []byte("params"))
	require.NoError(t, err)

	// Test NewTransaction
	mockDB.NewTransactionReturns = &mockTokenDBTransaction{}
	tx, err := storage.NewTransaction()
	require.NoError(t, err)
	assert.NotNil(t, tx)
}

func TestTransaction_DeleteToken(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}
	pub := &mockPublisher{}

	tx, err := NewTransaction(pub, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	// nil token returned — no notify expected
	err = tx.DeleteToken(ctx, token2.ID{TxId: "tx1", Index: 0}, "actor1")
	require.NoError(t, err)
	assert.Equal(t, 0, pub.PublishCalls)

	// token present — notify should be called for each owner
	mockTx.GetTokenValue = &token2.Token{Type: "TOK", Owner: []byte("alice")}
	mockTx.GetTokenOwners = []string{"wallet1"}
	err = tx.DeleteToken(ctx, token2.ID{TxId: "tx1", Index: 0}, "actor1")
	require.NoError(t, err)
	assert.True(t, pub.PublishCalls > 0)
}

func TestTransaction_DeleteTokens(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}
	pub := &mockPublisher{}

	tx, err := NewTransaction(pub, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	ids := []*token2.ID{
		{TxId: "tx1", Index: 0},
		{TxId: "tx1", Index: 1},
	}
	err = tx.DeleteTokens(ctx, "actor1", ids)
	require.NoError(t, err)
}

func TestTransaction_AppendToken(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}
	pub := &mockPublisher{}

	tx, err := NewTransaction(pub, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	tta := TokenToAppend{
		txID:      "tx1",
		index:     0,
		tok:       &token2.Token{Type: "TOK", Owner: []byte("alice"), Quantity: "0x64"},
		precision: 64,
		owners:    []string{"wallet1"},
		flags:     Flags{Mine: true},
	}
	err = tx.AppendToken(ctx, tta)
	require.NoError(t, err)
	assert.True(t, pub.PublishCalls > 0)
}

func TestTransaction_AppendToken_EmptyOwner(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}
	pub := &mockPublisher{}

	tx, err := NewTransaction(pub, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	// empty owner string — should not publish
	tta := TokenToAppend{
		txID:      "tx1",
		index:     0,
		tok:       &token2.Token{Type: "TOK", Owner: []byte("alice"), Quantity: "0x64"},
		precision: 64,
		owners:    []string{""},
	}
	err = tx.AppendToken(ctx, tta)
	require.NoError(t, err)
	assert.Equal(t, 0, pub.PublishCalls)
}

func TestTransaction_Notify_NilPublisher(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}

	// nil publisher — should not panic
	tx, err := NewTransaction(nil, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)
	tx.Notify(ctx, AddToken, tmsID, "wallet1", "TOK", "tx1", 0)
}

func TestTransaction_Rollback(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}

	tx, err := NewTransaction(nil, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	err = tx.Rollback()
	require.NoError(t, err)
}

func TestTransaction_SetSpendableFlag(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}

	tx, err := NewTransaction(nil, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	ids := []*token2.ID{{TxId: "tx1", Index: 0}, {TxId: "tx2", Index: 1}}
	err = tx.SetSpendableFlag(ctx, true, ids)
	require.NoError(t, err)
}

func TestTransaction_SetSpendableBySupportedTokenTypes(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockTx := &mockTokenDBTransaction{}

	tx, err := NewTransaction(nil, &tokendb.Transaction{TokenStoreTransaction: mockTx}, tmsID)
	require.NoError(t, err)

	err = tx.SetSpendableBySupportedTokenTypes(ctx, []token2.Format{"fmt1"})
	require.NoError(t, err)
}

func TestTokenProcessorEvent(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	msg := &TokenMessage{
		TMSID:     tmsID,
		WalletID:  "wallet1",
		TokenType: "TOK",
		TxID:      "tx1",
		Index:     0,
	}
	e := NewTokenProcessorEvent(AddToken, msg)
	assert.Equal(t, AddToken, e.Topic())
	assert.NotNil(t, e.Message())
}

func TestDBStorage_NewTransaction_Error(t *testing.T) {
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockDB := &mockTokenDB{
		NewTransactionError: assert.AnError,
	}
	storage, err := NewDBStorage(nil, &tokendb.StoreService{TokenStore: mockDB}, tmsID)
	require.NoError(t, err)

	_, err = storage.NewTransaction()
	require.Error(t, err)
}
