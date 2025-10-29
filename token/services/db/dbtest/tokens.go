/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package dbtest

import (
	"fmt"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	tdriver "github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/sql/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

type cfgProvider func(string) driver.Driver

func TokensTest(t *testing.T, cfgProvider cfgProvider) {
	t.Helper()
	for _, c := range tokensCases {
		t.Run(c.Name, func(xt *testing.T) {
			driver := cfgProvider(c.Name)
			db, err := driver.NewToken("", c.Name)
			if err != nil {
				t.Fatal(err)
			}
			tokenDB, ok := db.(*common.TokenStore)
			assert.True(xt, ok)
			defer utils.IgnoreError(tokenDB.Close)
			c.Fn(t, db.(*common.TokenStore))
		})
	}
	// for _, c := range TokenNotifierCases {
	//	db, err := initTokenNDB(sql2.Postgres, pgConnStr, c.Name, 10)
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//	t.Run(c.Name, func(xt *testing.T) {
	//		defer Close(db)
	//		c.Fn(xt, db)
	//	})
	// }
}

var tokensCases = []struct {
	Name string
	Fn   func(*testing.T, TestTokenDB)
}{
	{"Transaction", TTokenTransaction},
	{"SaveAndGetToken", TSaveAndGetToken},
	{"DeleteAndMine", TDeleteAndMine},
	{"GetTokenMetadata", TGetTokenInfos},
	{"ListAuditTokens", TListAuditTokens},
	{"ListIssuedTokens", TListIssuedTokens},
	{"DeleteMultiple", TDeleteMultiple},
	{"PublicParams", TPublicParams},
	{"Certification", TCertification},
	{"QueryTokenDetails", TQueryTokenDetails},
	{"TTokenTypes", TTokenTypes},
}

func TTokenTransaction(t *testing.T, db TestTokenDB) {
	t.Helper()
	tx, err := db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(t.Context(), driver.TokenRecord{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           TST,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}, []string{"alice"})
	assert.NoError(t, err)
	assert.NoError(t, tx.Commit())

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err := tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, false)
	assert.NoError(t, err, "get token")
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)

	assert.NoError(t, tx.Delete(t.Context(), token.ID{TxId: "tx1"}, "me"))
	tok, owners, err = tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, false)
	assert.NoError(t, err)
	assert.Nil(t, tok)
	assert.Len(t, owners, 0)

	tok, _, err = tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, true) // include deleted
	assert.NoError(t, err)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.NoError(t, tx.Rollback())

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, false)
	assert.NoError(t, err)
	assert.NotNil(t, tok)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)
	assert.NoError(t, tx.Delete(t.Context(), token.ID{TxId: "tx1"}, "me"))
	assert.NoError(t, tx.Commit())

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken(t.Context(), token.ID{TxId: "tx1"}, false)
	assert.NoError(t, err)
	assert.Nil(t, tok)
	assert.Equal(t, []string(nil), owners)
	assert.NoError(t, tx.Commit())
}

func TSaveAndGetToken(t *testing.T, db TestTokenDB) {
	t.Helper()
	for i := range 20 {
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
			Type:           TST,
			Amount:         2,
			Owner:          true,
			Auditor:        false,
			Issuer:         false,
		}
		assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
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
		Type:           TST,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"dan"}))

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
		Type:           TST,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice", "bob"}))

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
		Type:           ABC,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))

	tok, err := db.ListUnspentTokens(t.Context())
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 24, "unspentTokensIterator: expected all tokens to be returned (2 for the one owned by alice and bob)")
	assert.Equal(t, "48", tok.Sum(64).Decimal(), "expect sum to be 2*22")
	assert.Len(t, tok.ByType(TST).Tokens, 23, "expect filter on type to work")
	for _, token := range tok.Tokens {
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner, "expected owner raw to not be empty")
	}

	tokens := getTokensBy(t, db, "alice", "")
	assert.NoError(t, err)
	assert.Len(t, tokens, 22, "unspentTokensIteratorBy: expected only Alice tokens to be returned")

	tokens = getTokensBy(t, db, "", ABC)
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only ABC tokens to be returned")

	tokens = getTokensBy(t, db, "alice", ABC)
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only Alice ABC tokens to be returned")

	unsp, err := db.GetTokens(t.Context(), &token.ID{TxId: "tx101", Index: 0})
	assert.NoError(t, err)
	assert.Len(t, unsp, 1)
	assert.Equal(t, "0x02", unsp[0].Quantity)
	assert.Equal(t, ABC, unsp[0].Type)

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
		Type:           ABC,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))
	_, err = db.GetTokens(t.Context(), &token.ID{TxId: fmt.Sprintf("tx%d", 2000)})
	assert.NoError(t, err)

	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	_, owners, err := tx.GetToken(t.Context(), token.ID{TxId: fmt.Sprintf("tx%d", 2000)}, true)
	assert.NoError(t, err)
	assert.Len(t, owners, 1)
	assert.NoError(t, tx.Rollback())
}

func getTokensBy(t *testing.T, db TestTokenDB, ownerEID string, typ token.Type) []*token.UnspentToken {
	t.Helper()
	it, err := db.UnspentTokensIteratorBy(t.Context(), ownerEID, typ)
	assert.NoError(t, err)

	tokens, err := iterators.ReadAllPointers(it)
	assert.NoError(t, err, "error iterating over tokens")

	return tokens
}

func TDeleteAndMine(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"bob"}))
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
	assert.NoError(t, db.DeleteTokens(ctx, "tx103", &token.ID{TxId: "tx101", Index: 0}))

	tok, err := db.ListUnspentTokens(t.Context())
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 2, "expected only tx101-0 to be deleted")

	mine, err := db.IsMine(ctx, "tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine(ctx, "tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	deletedBy, deleted, err := db.WhoDeletedTokens(ctx, tid...)
	assert.NoError(t, err)
	assert.True(t, deleted[0], "expected tx101-0 to be deleted")
	assert.Equal(t, "tx103", deletedBy[0], "expected tx101-0 to be deleted by tx103")
	assert.False(t, deleted[1], "expected tx101-0 to not be deleted")
	assert.Equal(t, "", deletedBy[1], "expected tx101-0 to not be deleted by tx103")
}

// // ListAuditTokens returns the audited tokens associated to the passed ids
func TListAuditTokens(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
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
		Type:           ABC,
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))
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
		Type:           ABC,
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))
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
		Type:           ABC,
		Amount:         0,
		Owner:          false,
		Auditor:        true,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	tok, err := db.ListAuditTokens(ctx, tid...)
	assert.NoError(t, err)
	assert.Len(t, tok, 2)
	assert.Equal(t, "0x01", tok[0].Quantity, "expected tx101-0 to be returned")
	assert.Equal(t, "0x02", tok[1].Quantity, "expected tx101-1 to be returned")
	for _, token := range tok {
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner, "expected owner raw to not be empty")
	}

	tok, err = db.ListAuditTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, tok, 0)
}

func TListIssuedTokens(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
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
		Type:           ABC,
		Amount:         0,
		Owner:          false,
		Auditor:        false,
		Issuer:         true,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))
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
		Type:           ABC,
		Amount:         0,
		Owner:          false,
		Auditor:        false,
		Issuer:         true,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))
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
	assert.NoError(t, db.StoreToken(t.Context(), tr, nil))

	tok, err := db.ListHistoryIssuedTokens(ctx)
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
		assert.NotEmpty(t, token.Issuer, "expected issuer raw to not be empty")
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner, "expected owner raw to not be empty")
	}

	tok, err = db.ListHistoryIssuedTokens(ctx)
	assert.NoError(t, err)
	assert.Len(t, tok.ByType("DEF").Tokens, 1, "expected tx102-0 to be filtered")
	for _, token := range tok.Tokens {
		assert.NotNil(t, token.Issuer, "expected issuer to not be nil")
		assert.NotEmpty(t, token.Issuer, "expected issuer raw to not be empty")
		assert.NotNil(t, token.Owner, "expected owner to not be nil")
		assert.NotEmpty(t, token.Owner, "expected owner raw to not be empty")
	}
}

// GetTokenMetadata retrieves the token information for the passed ids.
// For each id, the callback is invoked to unmarshal the token information
func TGetTokenInfos(t *testing.T, db TestTokenDB) {
	t.Helper()
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"bob"}))
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
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
		Type:           ABC,
		Amount:         0,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))

	ids := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx102", Index: 0},
	}

	infos, err := db.GetAllTokenInfos(t.Context(), ids)
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
	_, err = db.GetTokenMetadata(t.Context(), ids)
	assert.Error(t, err)

	ids = []*token.ID{
		{TxId: "tx102", Index: 1},
		{TxId: "tx102", Index: 0},
		{TxId: "tx101", Index: 0},
	}
	infos, err = db.GetTokenMetadata(t.Context(), ids)
	assert.NoError(t, err)
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))

	// infos and outputs
	toks, infos, _, err := db.GetTokenOutputsAndMeta(t.Context(), ids)
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

func TDeleteMultiple(t *testing.T, db TestTokenDB) {
	t.Helper()
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           ABC,
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           ABC,
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           ABC,
		Owner:          true,
	}
	assert.NoError(t, db.StoreToken(t.Context(), tr, []string{"alice"}))
	assert.NoError(t, db.DeleteTokens(t.Context(), "", &token.ID{TxId: "tx101", Index: 0}, &token.ID{TxId: "tx102", Index: 0}))

	tok, err := db.ListUnspentTokens(t.Context())
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 1, "expected only tx101-0 and tx102-0 to be deleted", tok.Tokens)

	mine, err := db.IsMine(t.Context(), "tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine(t.Context(), "tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")
}

func TPublicParams(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
	b := []byte("test bytes")
	bHash := utils.Hashable(b).Raw()
	b1 := []byte("test bytes1")
	b1Hash := utils.Hashable(b1).Raw()

	res, err := db.PublicParams(ctx)
	assert.NoError(t, err) // not found
	assert.Nil(t, res)

	err = db.StorePublicParams(ctx, b)
	assert.NoError(t, err)

	res, err = db.PublicParams(ctx)
	assert.NoError(t, err)
	assert.Equal(t, res, b)

	// retrieve by hash
	res, err = db.PublicParamsByHash(ctx, bHash)
	assert.NoError(t, err)
	assert.Equal(t, res, b)

	err = db.StorePublicParams(ctx, b1)
	assert.NoError(t, err)

	res, err = db.PublicParams(ctx)
	assert.NoError(t, err)
	assert.Equal(t, res, b1)

	// retrieve by hash
	res, err = db.PublicParamsByHash(ctx, b1Hash)
	assert.NoError(t, err)
	assert.Equal(t, res, b1)
}

func TCertification(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
	wg := sync.WaitGroup{}
	wg.Add(40)
	for i := range 40 {
		go func(i int) {
			tokenID := &token.ID{
				TxId:  fmt.Sprintf("tx_%d", i),
				Index: 0,
			}
			err := db.StoreToken(t.Context(), driver.TokenRecord{
				TxID:           tokenID.TxId,
				Index:          tokenID.Index,
				OwnerRaw:       []byte{1, 2, 3},
				OwnerType:      "idemix",
				OwnerIdentity:  []byte{},
				Quantity:       "0x01",
				Ledger:         []byte("ledger"),
				LedgerMetadata: []byte{},
				Type:           ABC,
				Owner:          true,
			}, []string{"alice"})
			if err != nil {
				t.Error(err)
			}

			assert.NoError(t, db.StoreCertifications(ctx, map[*token.ID][]byte{
				tokenID: []byte(fmt.Sprintf("certification_%d", i)),
			}))
			assert.True(t, db.ExistsCertification(ctx, tokenID))
			certifications, err := db.GetCertifications(ctx, []*token.ID{tokenID})
			assert.NoError(t, err)
			for _, bytes := range certifications {
				assert.Equal(t, fmt.Sprintf("certification_%d", i), string(bytes))
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	for i := range 40 {
		tokenID := &token.ID{
			TxId:  fmt.Sprintf("tx_%d", i),
			Index: 0,
		}
		assert.True(t, db.ExistsCertification(ctx, tokenID))
		certifications, err := db.GetCertifications(ctx, []*token.ID{tokenID})
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
	assert.False(t, db.ExistsCertification(ctx, tokenID))

	certifications, err := db.GetCertifications(ctx, []*token.ID{tokenID})
	assert.Error(t, err)
	assert.Empty(t, certifications)

	// store an empty certification and check that an error is returned
	err = db.StoreCertifications(ctx, map[*token.ID][]byte{
		tokenID: {},
	})
	assert.Error(t, err)
	certifications, err = db.GetCertifications(ctx, []*token.ID{tokenID})
	assert.Error(t, err)
	assert.Empty(t, certifications)
}

func TQueryTokenDetails(t *testing.T, db TestTokenDB) {
	t.Helper()
	ctx := t.Context()
	tx, err := db.NewTokenDBTransaction()
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
		Type:           TST,
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
		Type:           TST,
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}

	err = tx.StoreToken(t.Context(), tx1, []string{"alice"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(t.Context(), tx2, []string{"alice"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(t.Context(), tx21, []string{"bob"})
	if err != nil {
		t.Fatal(err)
	}
	err = tx.Commit()
	if err != nil {
		t.Fatal(err)
	}

	// all
	res, err := db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{})
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])
	assertEqual(t, tx21, res[2])
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")
	assert.Equal(t, false, res[2].IsSpent, "tx2-1 is not spent")

	// alice
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{WalletID: "alice"})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])

	// alice TST1
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{WalletID: "alice", TokenType: "TST1"})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx1, res[0])
	balance, err := db.Balance(ctx, "alice", "TST1")
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// alice TST
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{WalletID: "alice", TokenType: TST})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx2, res[0])
	balance, err = db.Balance(ctx, "alice", TST)
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// bob TST
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{WalletID: "bob", TokenType: TST})
	assert.NoError(t, err)
	assert.Len(t, res, 1)
	assertEqual(t, tx21, res[0])
	balance, err = db.Balance(ctx, "bob", TST)
	assert.NoError(t, err)
	assert.Equal(t, res[0].Amount, balance)

	// spent
	assert.NoError(t, db.DeleteTokens(ctx, "delby", &token.ID{TxId: "tx2", Index: 1}))
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")

	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{IncludeDeleted: true})
	assert.NoError(t, err)
	assert.Len(t, res, 3)
	assert.Equal(t, false, res[0].IsSpent, "tx1 is not spent")
	assert.Equal(t, false, res[1].IsSpent, "tx2-0 is not spent")
	assert.Equal(t, true, res[2].IsSpent, "tx2-1 is spent")
	assert.Equal(t, "delby", res[2].SpentBy)

	// by ids
	res, err = db.QueryTokenDetails(ctx, driver.QueryTokenDetailsParams{IDs: []*token.ID{{TxId: "tx1", Index: 0}, {TxId: "tx2", Index: 0}}, IncludeDeleted: true})
	assert.NoError(t, err)
	assert.Len(t, res, 2)
	assertEqual(t, tx1, res[0])
	assertEqual(t, tx2, res[1])
}

func TTokenTypes(t *testing.T, db TestTokenDB) {
	t.Helper()
	tx, err := db.NewTokenDBTransaction()
	assert.NoError(t, err)
	tx1 := driver.TokenRecord{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		OwnerType:      "idemix",
		OwnerIdentity:  []byte{},
		Ledger:         []byte("ledger"),
		LedgerFormat:   "CLEAR",
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           TST,
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
		LedgerFormat:   "CLEAR1",
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST1",
		Amount:         2,
		Owner:          true,
		Auditor:        false,
		Issuer:         false,
	}
	assert.NoError(t, tx.StoreToken(t.Context(), tx1, []string{"alice"}))
	assert.NoError(t, tx.StoreToken(t.Context(), tx2, []string{"alice"}))
	assert.NoError(t, tx.Commit())

	it, err := db.SpendableTokensIteratorBy(t.Context(), "", TST)
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, TST, 1)
	it, err = db.SpendableTokensIteratorBy(t.Context(), "", "TST1")
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, "TST1", 1)

	// make all non-spendable
	tx, err = db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.SetSpendableBySupportedTokenFormats(t.Context(), []token.Format{"htlc"}))
	assert.NoError(t, tx.Commit())

	it, err = db.SpendableTokensIteratorBy(t.Context(), "", TST)
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, TST, 0)
	it, err = db.SpendableTokensIteratorBy(t.Context(), "", "TST1")
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, "TST1", 0)

	// make TST spendable
	tx, err = db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.SetSpendableBySupportedTokenFormats(t.Context(), []token.Format{"CLEAR"}))
	assert.NoError(t, tx.Commit())

	it, err = db.SpendableTokensIteratorBy(t.Context(), "", TST)
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, TST, 1)
	it, err = db.SpendableTokensIteratorBy(t.Context(), "", "TST1")
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, "TST1", 0)

	// make TST1 spendable
	tx, err = db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.SetSpendableBySupportedTokenFormats(t.Context(), []token.Format{"CLEAR1"}))
	assert.NoError(t, tx.Commit())

	it, err = db.SpendableTokensIteratorBy(t.Context(), "", TST)
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, TST, 0)
	it, err = db.SpendableTokensIteratorBy(t.Context(), "", "TST1")
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, "TST1", 1)

	// make both spendable
	tx, err = db.NewTokenDBTransaction()
	assert.NoError(t, err)
	assert.NoError(t, tx.SetSpendableBySupportedTokenFormats(t.Context(), []token.Format{"CLEAR", "CLEAR1"}))
	assert.NoError(t, tx.Commit())

	it, err = db.SpendableTokensIteratorBy(t.Context(), "", TST)
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, TST, 1)
	it, err = db.SpendableTokensIteratorBy(t.Context(), "", "TST1")
	assert.NoError(t, err)
	consumeSpendableTokensIterator(t, it, "TST1", 1)
}

func consumeSpendableTokensIterator(t *testing.T, it tdriver.SpendableTokensIterator, tokenType token.Type, count int) {
	t.Helper()
	defer it.Close()
	for range count {
		tok, err := it.Next()
		assert.NoError(t, err)
		assert.Equal(t, tokenType, tok.Type)
	}
	tok, err := it.Next()
	assert.NoError(t, err)
	assert.Nil(t, tok)
}

func assertEqual(t *testing.T, r driver.TokenRecord, d driver.TokenDetails) {
	t.Helper()
	assert.Equal(t, r.TxID, d.TxID)
	assert.Equal(t, r.Index, d.Index)
	assert.Equal(t, r.Amount, d.Amount)
	assert.Equal(t, r.OwnerType, d.OwnerType)
}
