/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/tokendb/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

var ns = "testing"

func TestTokensSqlite(t *testing.T) {
	tempDir := t.TempDir()

	for _, c := range TokensCases {
		initSqlite(t, tempDir, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close() // TODO
			c.Fn(xt, Tokens)
		})
	}
}

func TestTokensSqliteMemory(t *testing.T) {
	for _, c := range TokensCases {
		initSqliteMemory(t, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Tokens)
		})
	}
}

func TestTokensPostgres(t *testing.T) {
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range TokensCases {
		initPostgres(t, pgConnStr, c.Name)
		t.Run(c.Name, func(xt *testing.T) {
			defer Transactions.Close()
			c.Fn(xt, Tokens)
		})
	}
}

var TokensCases = []struct {
	Name string
	Fn   func(*testing.T, *TokenDB)
}{
	{"SaveAndGetToken", TSaveAndGetToken},
	{"DeleteAndMine", TDeleteAndMine},
	{"GetTokenInfos", TGetTokenInfos},
	{"ListAuditTokens", TListAuditTokens},
	{"ListIssuedTokens", TListIssuedTokens},
	{"DeleteMultiple", TDeleteMultiple},
	{"PublicParams", TPublicParams},
}

func TSaveAndGetToken(t *testing.T, db *TokenDB) {
	for i := 0; i < 20; i++ {
		owners := []string{"alice"}
		tr := driver.TokenRecord{
			TxID:           fmt.Sprintf("tx%d", i),
			Index:          0,
			Namespace:      ns,
			IssuerRaw:      []byte{},
			OwnerRaw:       []byte{1, 2, 3},
			Ledger:         []byte("ledger"),
			LedgerMetadata: []byte{},
			Quantity:       "0x02",
			Type:           "TST",
			Amount:         0,
		}
		assert.NoError(t, db.StoreOwnerToken(tr, owners))
	}
	tr := driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 100),
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"dan"}))

	tr = driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 100), // only txid + index + ns is unique together
		Index:          1,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "TST",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice", "bob"}))

	tr = driver.TokenRecord{
		TxID:           fmt.Sprintf("tx%d", 101),
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))

	tok, err := db.ListUnspentTokens(ns)
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 23, "unspentTokensIterator: expected all tokens to be returned")
	assert.Equal(t, 23, tok.Count())
	assert.Equal(t, "46", tok.Sum(64).Decimal(), "expect sum to be 2*23")
	assert.Len(t, tok.ByType("TST").Tokens, 22, "expect filter on type to work")

	tokens := getTokensBy(t, db, "alice", "")
	assert.NoError(t, err)
	assert.Len(t, tokens, 22, "unspentTokensIteratorBy: expected only Alice tokens to be returned")

	tokens = getTokensBy(t, db, "", "ABC")
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only ABC tokens to be returned")

	tokens = getTokensBy(t, db, "alice", "ABC")
	assert.Len(t, tokens, 1, "unspentTokensIteratorBy: expected only Alice ABC tokens to be returned")

	keys, unsp, err := db.GetTokens(ns, &token.ID{TxId: "tx101", Index: 0})
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Len(t, unsp, 1)
	assert.Equal(t, "0x02", unsp[0].Quantity)
	assert.Equal(t, "ABC", unsp[0].Type)
	assert.Equal(t, keys[0], "\x00ztoken\x00tx101\x000\x00")
}

func getTokensBy(t *testing.T, db *TokenDB, ownerEID, typ string) []*token.UnspentToken {
	it, err := db.UnspentTokensIteratorBy(ns, ownerEID, typ)
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
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))
	assert.NoError(t, db.Delete(ns, "tx101", 0, "tx103"))

	tok, err := db.ListUnspentTokens(ns)
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 2, "expected only tx101-0 to be deleted")

	mine, err := db.IsMine(ns, "tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine(ns, "tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	deletedBy, deleted, err := db.WhoDeletedTokens(ns, tid...)
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
		Namespace:      ns,
		OwnerRaw:       []byte{1, 2},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreAuditToken(tr))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		Namespace:      ns,
		OwnerRaw:       []byte{3, 4},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreAuditToken(tr))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		Namespace:      ns,
		OwnerRaw:       []byte{5, 6},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x03",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreAuditToken(tr))

	tid := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx101", Index: 1},
	}
	tok, err := db.ListAuditTokens(ns, tid...)
	assert.NoError(t, err)
	assert.Len(t, tok, 2)
	assert.Equal(t, "0x01", tok[0].Quantity, "expected tx101-0 to be returned")
	assert.Equal(t, "0x02", tok[1].Quantity, "expected tx101-1 to be returned")

	tok, err = db.ListAuditTokens(ns)
	assert.NoError(t, err)
	assert.Len(t, tok, 0)
}

func TListIssuedTokens(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		Namespace:      ns,
		OwnerRaw:       []byte{1, 2},
		IssuerRaw:      []byte{11, 12},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreIssuedToken(tr))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		Namespace:      ns,
		OwnerRaw:       []byte{3, 4},
		IssuerRaw:      []byte{13, 14},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x02",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreIssuedToken(tr))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		Namespace:      ns,
		OwnerRaw:       []byte{5, 6},
		IssuerRaw:      []byte{15, 16},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x03",
		Type:           "DEF",
		Amount:         0,
	}
	assert.NoError(t, db.StoreIssuedToken(tr))

	tok, err := db.ListHistoryIssuedTokens(ns)
	if err != nil {
		t.Fatal(err)
	}
	assert.Len(t, tok.Tokens, 3)
	assert.Equal(t, 3, tok.Count(), "expected 3 issued tokens")
	assert.Equal(t, "0x01", tok.Tokens[0].Quantity, "expected tx101-0 to be returned")
	assert.Equal(t, "0x02", tok.Tokens[1].Quantity, "expected tx101-1 to be returned")
	assert.Equal(t, "0x03", tok.Tokens[2].Quantity, "expected tx102-0 to be returned")

	tok, err = db.ListHistoryIssuedTokens(ns)
	assert.NoError(t, err)
	assert.Len(t, tok.ByType("DEF").Tokens, 1, "expected tx102-0 to be filtered")
}

// GetTokenInfos retrieves the token information for the passed ids.
// For each id, the callback is invoked to unmarshal the token information
func TGetTokenInfos(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("tx101l"),
		LedgerMetadata: []byte("tx101"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("tx102l"),
		LedgerMetadata: []byte("tx102"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          1,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("tx102l"),
		LedgerMetadata: []byte("tx102"),
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))

	ids := []*token.ID{
		{TxId: "tx101", Index: 0},
		{TxId: "tx102", Index: 0},
	}
	assert.NoError(t, db.GetTokenInfos(ns, ids, func(id *token.ID, info []byte) error {
		assert.Equal(t, id.TxId, string(info))
		assert.Equal(t, uint64(0), id.Index)
		return nil
	}))

	infos, err := db.GetAllTokenInfos(ns, ids)
	assert.NoError(t, err)
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
	infos = [][]byte{}
	assert.NoError(t, db.GetTokenInfos(ns, ids, func(id *token.ID, info []byte) error {
		infos = append(infos, info)
		return nil
	}))
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))
	assert.Equal(t, "", string(infos[3]))

	// infos and outputs
	toks := [][]byte{}
	infos = [][]byte{}
	keys := []string{}
	assert.NoError(t, db.GetTokenInfoAndOutputs(ns, ids, func(id *token.ID, key string, tok []byte, info []byte) error {
		toks = append(toks, tok)
		infos = append(infos, info)
		keys = append(keys, key)
		return nil
	}))
	assert.NoError(t, err)
	assert.Len(t, infos, 4)
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))
	assert.Len(t, toks, 4)
	assert.Equal(t, "tx102l", string(toks[0]))
	assert.Equal(t, "tx102l", string(toks[1]))
	assert.Equal(t, "tx101l", string(toks[2]))
	assert.Equal(t, "\x00ztoken\x00tx101\x000\x00", string(keys[2]))
}

func TDeleteMultiple(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))
	tr = driver.TokenRecord{
		TxID:           "tx101",
		Index:          1,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"bob"}))
	tr = driver.TokenRecord{
		TxID:           "tx102",
		Index:          0,
		Namespace:      ns,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
		Ledger:         []byte("ledger"),
		LedgerMetadata: []byte{},
		Quantity:       "0x01",
		Type:           "ABC",
		Amount:         0,
	}
	assert.NoError(t, db.StoreOwnerToken(tr, []string{"alice"}))
	assert.NoError(t, db.DeleteTokens(ns,
		&token.ID{TxId: "tx101", Index: 0},
		&token.ID{TxId: "tx102", Index: 0},
	))

	tok, err := db.ListUnspentTokens(ns)
	assert.NoError(t, err)
	assert.Len(t, tok.Tokens, 1, "expected only tx101-0 and tx102-0 to be deleted", tok.Tokens)

	mine, err := db.IsMine(ns, "tx101", 0)
	assert.NoError(t, err)
	assert.False(t, mine, "expected deleted token to not be mine")

	mine, err = db.IsMine(ns, "tx101", 1)
	assert.NoError(t, err)
	assert.True(t, mine, "expected existing token to be mine")
}

func TPublicParams(t *testing.T, db *TokenDB) {
	b := []byte("test bytes")
	b1 := []byte("test bytes1")

	res, err := db.GetRawPublicParams()
	assert.NoError(t, err) // not found
	assert.Nil(t, res)

	err = db.StorePublicParams(b)
	assert.NoError(t, err)

	res, err = db.GetRawPublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b)

	err = db.StorePublicParams(b1)
	assert.NoError(t, err)

	res, err = db.GetRawPublicParams()
	assert.NoError(t, err) // not found
	assert.Equal(t, res, b1)
}
