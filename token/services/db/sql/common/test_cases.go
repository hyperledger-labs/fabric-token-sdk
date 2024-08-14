/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package common

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	assert2 "github.com/stretchr/testify/assert"
	"github.com/test-go/testify/assert"
)

var TokenNotifierCases = []struct {
	Name string
	Fn   func(*testing.T, driver.TokenDB, driver.TokenNotifier)
}{
	{"SubscribeStore", TSubscribeStore},
	{"SubscribeStoreDelete", TSubscribeStoreDelete},
	{"SubscribeStoreNoCommit", TSubscribeStoreNoCommit},
	{"SubscribeRead", TSubscribeRead},
}

type dbEvent struct {
	op   driver2.Operation
	vals map[driver2.ColumnKey]string
}

func collectDBEvents(db driver.TokenNotifier) (*[]dbEvent, error) {
	ch := make(chan dbEvent)
	err := db.Subscribe(func(operation driver2.Operation, m map[driver2.ColumnKey]string) {
		logger.Infof("Received event: [%v]: %v", operation, m)
		ch <- dbEvent{op: operation, vals: m}
	})
	if err != nil {
		return nil, err
	}
	result := make([]dbEvent, 0, 1)
	go func() {
		for e := range ch {
			result = append(result, e)
		}
	}()
	return &result, nil
}

func TSubscribeStore(t *testing.T, db driver.TokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction(context.TODO())
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 2 }, time.Second, 20*time.Millisecond)
}

func TSubscribeStoreDelete(t *testing.T, db driver.TokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction(context.TODO())
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))
	assert.NoError(t, tx.Delete(context.TODO(), "tx1", 1, "alice"))
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 3 }, time.Second, 20*time.Millisecond)
}

func TSubscribeStoreNoCommit(t *testing.T, db driver.TokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction(context.TODO())
	assert.NoError(t, err)
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 1}, []string{"alice"}))

	assert2.Eventually(t, func() bool { return len(*result) == 0 }, time.Second, 20*time.Millisecond)
}

func TSubscribeRead(t *testing.T, db driver.TokenDB, notifier driver.TokenNotifier) {
	result, err := collectDBEvents(notifier)
	assert.Nil(t, err)
	tx, err := db.NewTokenDBTransaction(context.TODO())
	assert.NoError(t, err)
	//assert.NoError(t, tx.StoreToken(context.TODO(), driver.TokenRecord{TxID: "tx1", Index: 0}, []string{"alice"}))
	_, _, err = tx.GetToken(context.TODO(), "tx1", 0, true)
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit())

	assert2.Eventually(t, func() bool { return len(*result) == 0 }, time.Second, 20*time.Millisecond)
}

var TokensCases = []struct {
	Name string
	Fn   func(*testing.T, *TokenDB)
}{
	{"Transaction", TTransaction},
	{"SaveAndGetToken", TSaveAndGetToken},
	{"DeleteAndMine", TDeleteAndMine},
	{"GetTokenInfos", TGetTokenInfos},
	{"ListAuditTokens", TListAuditTokens},
	{"ListIssuedTokens", TListIssuedTokens},
	{"DeleteMultiple", TDeleteMultiple},
	{"PublicParams", TPublicParams},
	{"Certification", TCertification},
	{"QueryTokenDetails", TQueryTokenDetails},
}

func TTransaction(t *testing.T, db *TokenDB) {
	tx, err := db.NewTokenDBTransaction(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(context.TODO(), driver.TokenRecord{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}, []string{"alice"})
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit())

	tx, err = db.NewTokenDBTransaction(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err := tx.GetToken(context.TODO(), "tx1", 0, false)
	assert.NoError(t, err, "get token")
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)

	assert.NoError(t, tx.Delete(context.TODO(), "tx1", 0, "me"))
	tok, owners, err = tx.GetToken(context.TODO(), "tx1", 0, false)
	assert.NoError(t, err)
	assert.Nil(t, tok)
	assert.Len(t, owners, 0)

	tok, _, err = tx.GetToken(context.TODO(), "tx1", 0, true) // include deleted
	assert.NoError(t, err)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.NoError(t, tx.Rollback())

	tx, err = db.NewTokenDBTransaction(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken(context.TODO(), "tx1", 0, false)
	assert.NoError(t, err)
	assert.NotNil(t, tok)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)
	assert.NoError(t, tx.Delete(context.TODO(), "tx1", 0, "me"))
	assert.NoError(t, tx.Commit())

	tx, err = db.NewTokenDBTransaction(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken(context.TODO(), "tx1", 0, false)
	assert.NoError(t, err)
	assert.Nil(t, tok)
	assert.Equal(t, []string{}, owners)
	assert.NoError(t, tx.Commit())
}

func TSaveAndGetToken(t *testing.T, db *TokenDB) {
	for i := 0; i < 20; i++ {
		tr := driver.TokenRecord{
			TxID:           fmt.Sprintf("tx%d", i),
			Index:          0,
			IssuerRaw:      []byte{},
			OwnerRaw:       []byte{1, 2, 3},
			OwnerType:      "idemix",
			OwnerIdentity:  []byte{},
			Ledger:         []byte("ledger"),
			LedgerMetadata: []byte{},
			Quantity:       "0x02",
			Type:           "TST",
			Amount:         2,
			Owner:          true,
			Auditor:        false,
			Issuer:         false,
		}
		assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	}
	tr := driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 100),
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"dan"}))

	tr = driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 100), // only txid + index + ns is unique together
		Index:          1,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice", "bob"}))

	tr = driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 101),
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))

	tok, err := db.ListUnspentTokens()
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 24, "unspentTokensIterator: expected all tokens to be returned (2 for the one owned by alice and bob)")
	assert.Equal(t, "48", tok.Sum(64).Decimal(), "expect sum to be 2*22")
	assert.Len(t, tok.ByType("TST").Tokens, 23, "expect filter on type to work")
	for _, token := range tok.Tokens {
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner.Raw, "expected owner raw to not be empty")
	}

	tokens := getTokensBy(t, db, "alice", "")
	assert.NoError(t, err)
	assert.Len(t, tokens, 22, "unspentTokensIteratorBy: expected only Alice tokens to be returned")

	tokens = getTokensBy(t, db, "", "ABC")
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only ABC tokens to be returned")

	tokens = getTokensBy(t, db, "alice", "ABC")
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only Alice ABC tokens to be returned")

	unsp, err := db.GetTokens(&token.ID{TxId: "tx101", Index: 0})
	assert.NoError(t, err)
	assert.Len(t, unsp, 1)
	assert.Equal(t, "0x02", unsp[0].Quantity)
	assert.Equal(t, "ABC", unsp[0].Type)

	tr = driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 2000),
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "pineapple",
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, nil))
	_, err = db.GetTokens(&token.ID{TxId: fmt.Sprintf("tx%d", 2000), Index: 0})
	assert.NoError(t, err)

	tx, err := db.NewTokenDBTransaction(context.TODO())
	assert.NoError(t, err)
	_, owners, err := tx.GetToken(context.TODO(), fmt.Sprintf("tx%d", 2000), 0, true)
	assert.NoError(t, err)
	assert.Len(t, owners, 1)
	assert.NoError(t, tx.Rollback())
}

func getTokensBy(t *testing.T, db *TokenDB, ownerEID, typ string) []*token.UnspentToken {
	it, err := db.UnspentTokensIteratorBy(context.TODO(), ownerEID, typ)
	assert.NoError(t, err)
	defer it.Close()

	var tokens []*token.UnspentToken
	for {
		tok, err := it.Next()
		if err != nil {
			t.Errorf("error iterating over tokens: %s", err.Error())
		}
		if tok == nil {
			break
		}
		tokens = append(tokens, tok)
	}
	return tokens
}

func TDeleteAndMine(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	assert.NoError(t, db.DeleteTokens("tx103", &token.ID{TxId: "tx101", Index: 0}))

	tok, err := db.ListUnspentTokens()
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 2, "expected only tx101-0 to be deleted")

	mine, err := db.IsMine("tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine("tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	deletedBy, deleted, err := db.WhoDeletedTokens(tid...)
	assert.NoError(t, err)
	assert.True(t, deleted[0], "expected tx101-0 to be deleted")
	assert.Equal(t, "tx103", deletedBy[0], "expected tx101-0 to be deleted by tx103")
	assert.False(t, deleted[1], "expected tx101-0 to not be deleted")
	assert.Equal(t, "", deletedBy[1], "expected tx101-0 to not be deleted by tx103")
}

// // ListAuditTokens returns the audited tokens associated to the passed ids
func TListAuditTokens(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		OwnerRaw:       []byte{1, 2},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, nil))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		OwnerRaw:       []byte{3, 4},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, nil))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		OwnerRaw:       []byte{5, 6},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x03",
		Type:           "ABC",
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, nil))

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	tok, err := db.ListAuditTokens(tid...)
	assert.NoError(t, err)
	assert.Len(t, tok, 2)
	assert.Equal(t, "0x01", tok[0].Quantity, "expected tx101-0 to be returned")
	assert.Equal(t, "0x02", tok[1].Quantity, "expected tx101-1 to be returned")
	for _, token := range tok {
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner.Raw, "expected owner raw to not be empty")
	}

	tok, err = db.ListAuditTokens()
	assert.NoError(t, err)
	assert.Len(t, tok, 0)
}

func TListIssuedTokens(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		OwnerRaw:       []byte{1, 2},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		IssuerRaw:      []byte{11, 12},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          false,
		Auditor:        false,
		Issuer:         true,
	}
	assert.NoError(t, db.StoreToken(tr, nil))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		OwnerRaw:       []byte{3, 4},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		IssuerRaw:      []byte{13, 14},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         0,
		Owner:          false,
		Auditor:        false,
		Issuer:         true,
	}
	assert.NoError(t, db.StoreToken(tr, nil))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		OwnerRaw:       []byte{5, 6},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		OwnerWalletID:  "idemix",
		IssuerRaw:      []byte{15, 16},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x03",
		Type:           "DEF",
		Amount:         0,
		Owner:          false,
		Auditor:        false,
		Issuer:         true,
	}
	assert.NoError(t, db.StoreToken(tr, nil))

	tok, err := db.ListHistoryIssuedTokens()
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, tok.Tokens, 3)
	assert.Equal(t, 3, tok.Count(), "expected 3 issued tokens")
	assert.Equal(t, "0x01", tok.Tokens[0].Quantity, "expected tx101-0 to be returned")
	assert.Equal(t, "0x02", tok.Tokens[1].Quantity, "expected tx101-1 to be returned")
	assert.Equal(t, "0x03", tok.Tokens[2].Quantity, "expected tx102-0 to be returned")
	for _, token := range tok.Tokens {
		assert.NotNil(t, token.Issuer, "expected issuer to not be nil")
		assert.NotEmpty(t, token.Issuer.Raw, "expected issuer raw to not be empty")
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner.Raw, "expected owner raw to not be empty")
	}

	tok, err = db.ListHistoryIssuedTokens()
	assert.NoError(t, err)
	assert.Len(t, tok.ByType("DEF").Tokens, 1, "expected tx102-0 to be filtered")
	for _, token := range tok.Tokens {
		assert.NotNil(t, token.Issuer, "expected issuer to not be nil")
		assert.NotEmpty(t, token.Issuer.Raw, "expected issuer raw to not be empty")
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner.Raw, "expected owner raw to not be empty")
	}
}

// GetTokenInfos retrieves the token information for the passed ids.
// For each id, the callback is invoked to unmarshal the token information
func TGetTokenInfos(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("tx101l"),
		LedgerMetadata: []byte("tx101"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("tx102l"),
		LedgerMetadata: []byte("tx102"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          1,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("tx102l"),
		LedgerMetadata: []byte("tx102"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))

	ids := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx102", Index: 0},
	}

	infos, err := db.GetAllTokenInfos(ids)
	assert.NoError(t, err)
	for i, info := range infos {
		assert.Equal(t, ids[i].TxId, string(info))
		assert.Equal(t, uint64(0), ids[i].Index)
	}
	assert.Len(t, infos, 2)
	assert.Equal(t, "tx101", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))

	// Order should match the provided order
	ids = []*token.ID{
		{TxId: "tx102", Index: 1},
		{TxId: "tx102", Index: 0},
		{TxId: "tx101", Index: 0},
		{TxId: "non existent", Index: 0},
	}
	_, err = db.GetTokenInfos(ids)
	assert.Error(t, err)

	ids = []*token.ID{
		{TxId: "tx102", Index: 1},
		{TxId: "tx102", Index: 0},
		{TxId: "tx101", Index: 0},
	}
	infos, err = db.GetTokenInfos(ids)
	assert.NoError(t, err)
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))

	// infos and outputs
	toks, infos, err := db.GetTokenInfoAndOutputs(context.TODO(), ids)
	assert.NoError(t, err)
	assert.Len(t, infos, 3)
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))
	assert.Len(t, toks, 3)
	assert.Equal(t, "tx102l", string(toks[0]))
	assert.Equal(t, "tx102l", string(toks[1]))
	assert.Equal(t, "tx101l", string(toks[2]))
}

func TDeleteMultiple(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(tr, []string{"alice"}))
	assert.NoError(t, db.DeleteTokens("", &token.ID{TxId: "tx101", Index: 0}, &token.ID{TxId: "tx102", Index: 0}))

	tok, err := db.ListUnspentTokens()
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 1, "expected only tx101-0 and tx102-0 to be deleted", tok.Tokens)

	mine, err := db.IsMine("tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine("tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")
}

func TPublicParams(t *testing.T, db *TokenDB) {
	b := []byte("test bytes")
	b1 := []byte("test bytes1")

	res, err := db.PublicParams()
	assert.NoError(t, err) // not found
	assert.Nil(t, res)

	err = db.StorePublicParams(b)
	assert.NoError(t, err)

	res, err = db.PublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b)

	err = db.StorePublicParams(b1)
	assert.NoError(t, err)

	res, err = db.PublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b1)
}

func TCertification(t *testing.T, db *TokenDB) {
	wg := sync.WaitGroup{}
	wg.Add(40)
	for i := 0; i < 40; i++ {
		go func(i int) {
			tokenID := &token.ID{
				TxId:  fmt.Sprintf("tx_%d", i),
				Index: 0,
			}
			err := db.StoreToken(driver.TokenRecord{
				TxID:           tokenID.TxId,
				Index:          tokenID.Index,
				OwnerRaw:       []byte{1, 2, 3},
				OwnerType:      "idemix",
				OwnerIdentity:  []byte{},
				Quantity:       "0x01",
				Ledger:         []byte("ledger"),
				LedgerMetadata: []byte{},
				Type:           "ABC",
				Owner:          true,
			}, []string{"alice"})
			if err != nil {
				t.Error(err)
			}

			assert.NoError(t, db.StoreCertifications(map[*token.ID][]byte{
				tokenID: []byte(fmt.Sprintf("certification_%d", i)),
			}))
			assert.True(t, db.ExistsCertification(tokenID))
			certifications, err := db.GetCertifications([]*token.ID{tokenID})
			assert.NoError(t, err)
			for _, bytes := range certifications {
				assert.Equal(t, fmt.Sprintf("certification_%d", i), string(bytes))
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := 0; i < 40; i++ {
		tokenID := &token.ID{
			TxId:  fmt.Sprintf("tx_%d", i),
			Index: 0,
		}
		assert.True(t, db.ExistsCertification(tokenID))
		certifications, err := db.GetCertifications([]*token.ID{tokenID})
		assert.NoError(t, err)
		for _, bytes := range certifications {
			assert.Equal(t, fmt.Sprintf("certification_%d", i), string(bytes))
		}
	}

	// check the certification of a token that was never stored
	tokenID := &token.ID{
		TxId:  "pineapple",
		Index: 0,
	}
	assert.False(t, db.ExistsCertification(tokenID))

	certifications, err := db.GetCertifications([]*token.ID{tokenID})
	assert.Error(t, err)
	assert.Empty(t, certifications)

	// store an empty certification and check that an error is returned
	err = db.StoreCertifications(map[*token.ID][]byte{
		tokenID: {},
	})
	assert.Error(t, err)
	certifications, err = db.GetCertifications([]*token.ID{tokenID})
	assert.Error(t, err)
	assert.Empty(t, certifications)
}

func TQueryTokenDetails(t *testing.T, db *TokenDB) {
	tx, err := db.NewTokenDBTransaction(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	tx1 := driver.TokenRecord{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "TST1",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	tx2 := driver.TokenRecord{
		TxID:           "tx2",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "htlc",
		OwnerIdentity:  []byte("{}"),
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	tx21 := driver.TokenRecord{
		TxID:           "tx2",
		Index:          1,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "htlc",
		OwnerIdentity:  []byte("{}"),
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}

	err = tx.StoreToken(context.TODO(), tx1, []string{"alice"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(context.TODO(), tx2, []string{"alice"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(context.TODO(), tx21, []string{"bob"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// all
	res, err := db.QueryTokenDetails(driver.QueryTokenDetailsParams{})
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])
	assertEqual(t, tx21, res[2])
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")
	assert.Equal(t, false, res[2].IsSpent, "tx2-1 is not spent")

	// alice
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{WalletID: "alice"})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])

	// alice TST1
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{WalletID: "alice", TokenType: "TST1"})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx1, res[0])
	balance, err := db.Balance("alice", "TST1")
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// alice TST
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{WalletID: "alice", TokenType: "TST"})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx2, res[0])
	balance, err = db.Balance("alice", "TST")
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// bob TST
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{WalletID: "bob", TokenType: "TST"})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx21, res[0])
	balance, err = db.Balance("bob", "TST")
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// spent
	assert.NoError(t, db.DeleteTokens("delby", &token.ID{TxId: "tx2", Index: 1}))
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")

	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{IncludeDeleted: true})
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")
	assert.Equal(t, true, res[2].IsSpent, "tx2-1 is spent")
	assert.Equal(t, "delby", res[2].SpentBy)

	// by ids
	res, err = db.QueryTokenDetails(driver.QueryTokenDetailsParams{IDs: []*token.ID{{TxId: "tx1", Index: 0}, {TxId: "tx2", Index: 0}}, IncludeDeleted: true})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])
}

func assertEqual(t *testing.T, r driver.TokenRecord, d driver.TokenDetails) {
	assert.Equal(t, r.TxID, d.TxID)
	assert.Equal(t, r.Index, d.Index)
	assert.Equal(t, r.Amount, d.Amount)
	assert.Equal(t, r.OwnerType, d.OwnerType)
}
