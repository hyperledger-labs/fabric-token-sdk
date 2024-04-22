/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package sql

import (
	"fmt"
	"path"
	"sync"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/db/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
)

func initTokenDB(driverName, dataSourceName, tablePrefix string, maxOpenConns int) (*TokenDB, error) {
	d := NewSQLDBOpener("", "")
	sqlDB, err := d.OpenSQLDB(driverName, dataSourceName, maxOpenConns, false)
	if err != nil {
		return nil, err
	}
	return NewTokenDB(sqlDB, tablePrefix, true)
}

func TestTokensSqlite(t *testing.T) {
	tempDir := t.TempDir()
	for _, c := range TokensCases {
		db, err := initTokenDB("sqlite", fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)", path.Join(tempDir, "db.sqlite")), c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTokensSqliteMemory(t *testing.T) {
	for _, c := range TokensCases {
		db, err := initTokenDB("sqlite", "file:tmp?_pragma=busy_timeout(5000)&mode=memory&cache=shared", c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
}

func TestTokensPostgres(t *testing.T) {
	terminate, pgConnStr := startPostgresContainer(t)
	defer terminate()

	for _, c := range TokensCases {
		db, err := initTokenDB("postgres", pgConnStr, c.Name, 10)
		if err != nil {
			t.Fatal(err)
		}
		t.Run(c.Name, func(xt *testing.T) {
			defer db.Close()
			c.Fn(xt, db)
		})
	}
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
	{"TCertification", TCertification},
}

func TTransaction(t *testing.T, db *TokenDB) {
	tx, err := db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	err = tx.StoreToken(driver.TokenRecord{
		TxID:           "tx1",
		Index:          0,
		IssuerRaw:      []byte{},
		OwnerRaw:       []byte{1, 2, 3},
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

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err := tx.GetToken("tx1", 0, false)
	assert.NoError(t, err, "get token")
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)

	assert.NoError(t, tx.Delete("tx1", 0, "me"))
	tok, owners, err = tx.GetToken("tx1", 0, false)
	assert.NoError(t, err)
	assert.Nil(t, tok)
	assert.Len(t, owners, 0)

	tok, _, err = tx.GetToken("tx1", 0, true) // include deleted
	assert.NoError(t, err)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.NoError(t, tx.Rollback())

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken("tx1", 0, false)
	assert.NoError(t, err)
	assert.NotNil(t, tok)
	assert.Equal(t, "0x02", tok.Quantity)
	assert.Equal(t, []string{"alice"}, owners)
	assert.NoError(t, tx.Delete("tx1", 0, "me"))
	assert.NoError(t, tx.Commit())

	tx, err = db.NewTokenDBTransaction()
	if err != nil {
		t.Fatal(err)
	}
	tok, owners, err = tx.GetToken("tx1", 0, false)
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

	keys, unsp, err := db.GetTokens(&token.ID{TxId: "tx101", Index: 0})
	assert.NoError(t, err)
	assert.Len(t, keys, 1)
	assert.Len(t, unsp, 1)
	assert.Equal(t, "0x02", unsp[0].Quantity)
	assert.Equal(t, "ABC", unsp[0].Type)
	assert.Equal(t, keys[0], "\x00ztoken\x00tx101\x000\x00")
}

func getTokensBy(t *testing.T, db *TokenDB, ownerEID, typ string) []*token.UnspentToken {
	it, err := db.UnspentTokensIteratorBy(ownerEID, typ)
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
	keys, toks, infos, err := db.GetTokenInfoAndOutputs(ids)
	assert.NoError(t, err)
	assert.Len(t, infos, 3)
	assert.Equal(t, "tx102", string(infos[0]))
	assert.Equal(t, "tx102", string(infos[1]))
	assert.Equal(t, "tx101", string(infos[2]))
	assert.Len(t, toks, 3)
	assert.Equal(t, "tx102l", string(toks[0]))
	assert.Equal(t, "tx102l", string(toks[1]))
	assert.Equal(t, "tx101l", string(toks[2]))
	assert.Equal(t, "\x00ztoken\x00tx101\x000\x00", string(keys[2]))
}

func TDeleteMultiple(t *testing.T, db *TokenDB) {
	tr := driver.TokenRecord{
		TxID:           "tx101",
		Index:          0,
		OwnerRaw:       []byte{1, 2, 3},
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
				Quantity:       "0x01",
				Ledger:         []byte("ledger"),
				LedgerMetadata: []byte{},
				Type:           "ABC",
				Owner:          true,
			}, []string{"alice"})
			if err != nil {
				t.Fatal(err)
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
	assert.NoError(t, db.StoreCertifications(map[*token.ID][]byte{
		tokenID: {},
	}))
	assert.Error(t, err)
	certifications, err = db.GetCertifications([]*token.ID{tokenID})
	assert.Error(t, err)
	assert.Empty(t, certifications)
}
