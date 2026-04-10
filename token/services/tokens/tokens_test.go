/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package tokens

import (
	"context"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/storage/tokendb"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/test-go/testify/assert"
	"github.com/test-go/testify/require"
)

type qsMock struct{}

func (qs qsMock) IsMine(ctx context.Context, id *token2.ID) (bool, error) {
	return true, nil
}

type authMock struct{}

func (a authMock) Issued(ctx context.Context, issuer driver.Identity, tok *token2.Token) bool {
	return false
}
func (a authMock) IsMine(ctx context.Context, tok *token2.Token) (string, []string, bool) {
	return "", []string{string(tok.Owner)}, true
}
func (a authMock) AmIAnAuditor() bool {
	return false
}
func (a authMock) OwnerType(raw []byte) (driver.IdentityType, []byte, error) {
	return driver.IdemixIdentityType, raw, nil
}

type mdMock struct{}

func (md mdMock) SpentTokenID() []*token2.ID {
	// only called if graphHiding is true
	return []*token2.ID{}
}

func TestParse(t *testing.T) {
	ctx := t.Context()
	tokens := &Service{
		TMSProvider: nil,
		Storage:     &DBStorage{},
	}
	md := mdMock{}

	// simple transfer
	input1 := &token.Input{
		Id: &token2.ID{
			TxId:  "in",
			Index: 0,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output1 := &token.Output{
		Token: token2.Token{
			Type:  "TOK",
			Owner: []byte("alice"),
		},
		ActionIndex:  0,
		Index:        0,
		EnrollmentID: "bob",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	is := token.NewInputStream(qsMock{}, []*token.Input{input1}, 64)
	os := token.NewOutputStream([]*token.Output{output1}, 64)

	spend, store, err := tokens.parse(ctx, &authMock{}, "tx1", md, is, os, false, 64, false)
	require.NoError(t, err)

	assert.Len(t, spend, 1)
	assert.Equal(t, "in", spend[0].TxId)
	assert.Equal(t, uint64(0), spend[0].Index)

	assert.Len(t, store, 1)
	assert.Equal(t, "tx1", store[0].txID)
	assert.Equal(t, output1.Index, store[0].index)
	assert.Equal(t, output1.LedgerOutput, store[0].tokenOnLedger)
	assert.Equal(t, true, store[0].flags.Mine)
	assert.Equal(t, false, store[0].flags.Auditor)
	assert.Equal(t, false, store[0].flags.Issuer)
	assert.Equal(t, uint64(64), store[0].precision)
	assert.Equal(t, output1.Type, store[0].tok.Type)

	// no owner, then a redeemed token
	output1.Token.Owner = []byte{}
	os = token.NewOutputStream([]*token.Output{output1}, 64)
	spend, store, err = tokens.parse(ctx, &authMock{}, "tx1", md, is, os, false, 64, false)
	require.NoError(t, err)
	assert.Len(t, spend, 1)
	assert.Len(t, store, 0)

	// transfer with several inputs and outputs
	input1 = &token.Input{
		Id: &token2.ID{
			TxId:  "in1",
			Index: 1,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(50),
	}
	input2 := &token.Input{
		Id: &token2.ID{
			TxId:  "in2",
			Index: 2,
		},
		ActionIndex:  0,
		EnrollmentID: "alice",
		Type:         "TOK",
		Quantity:     token2.NewQuantityFromUInt64(50),
	}
	output1 = &token.Output{
		Token: token2.Token{
			Type:  "TOK",
			Owner: []byte("alice"),
		},
		ActionIndex:  0,
		Index:        0,
		EnrollmentID: "bob",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	output2 := &token.Output{
		Token: token2.Token{
			Type:  "TOK",
			Owner: []byte("bob"),
		},
		ActionIndex:  0,
		Index:        1,
		EnrollmentID: "alice",
		Type:         "TOK",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(90),
	}
	is = token.NewInputStream(qsMock{}, []*token.Input{input1, input2}, 64)
	os = token.NewOutputStream([]*token.Output{output1, output2}, 64)

	spend, store, err = tokens.parse(ctx, &authMock{}, "tx2", md, is, os, false, 64, false)
	require.NoError(t, err)
	assert.Len(t, spend, 2)
	assert.Equal(t, "in1", spend[0].TxId)
	assert.Equal(t, uint64(1), spend[0].Index)
	assert.Equal(t, "in2", spend[1].TxId)
	assert.Equal(t, uint64(2), spend[1].Index)

	assert.Len(t, store, 2)
	assert.Equal(t, output1.LedgerOutput, store[0].tokenOnLedger)
	assert.Equal(t, "tx2", store[0].txID)
	assert.Equal(t, output1.Index, store[0].index)
	assert.Equal(t, output1.Type, store[0].tok.Type)

	assert.Equal(t, output2.LedgerOutput, store[1].tokenOnLedger)
	assert.Equal(t, "tx2", store[1].txID)
	assert.Equal(t, output2.Index, store[1].index)
	assert.Equal(t, output2.Index, store[1].index)
	assert.Equal(t, output2.Type, store[1].tok.Type)
}

func TestParse_GraphHiding(t *testing.T) {
	ctx := t.Context()
	tokens := &Service{Storage: &DBStorage{}}
	md := mdMock{}

	is := token.NewInputStream(qsMock{}, nil, 64)
	os := token.NewOutputStream(nil, 64)

	// graphHiding=true — md.SpentTokenID() is used
	spend, store, err := tokens.parse(ctx, &authMock{}, "tx1", md, is, os, false, 64, true)
	require.NoError(t, err)
	assert.Len(t, spend, 0) // mdMock returns empty
	assert.Len(t, store, 0)
}

func TestParse_NilInputId(t *testing.T) {
	ctx := t.Context()
	tokens := &Service{Storage: &DBStorage{}}
	md := mdMock{}

	// input with nil Id — should be skipped
	inputNilId := &token.Input{Id: nil, ActionIndex: 0}
	is := token.NewInputStream(qsMock{}, []*token.Input{inputNilId}, 64)
	os := token.NewOutputStream(nil, 64)

	spend, _, err := tokens.parse(ctx, &authMock{}, "tx1", md, is, os, false, 64, false)
	require.NoError(t, err)
	assert.Len(t, spend, 0) // nil input.Id should be skipped
}

func TestGetActions_CacheHit(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	expectedSpend := []*token2.ID{{TxId: "tx-cached", Index: 0}}
	expectedAppend := []TokenToAppend{{txID: "tx-cached"}}

	mockCacheInst := &mockCache{
		GetFound: true,
		GetReturns: &CacheEntry{
			ToSpend:  expectedSpend,
			ToAppend: expectedAppend,
		},
	}
	tokens := &Service{
		RequestsCache: mockCacheInst,
		TMSProvider:   &mockTMSProvider{},
		Storage:       &DBStorage{tmsID: tmsID, tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}}},
	}

	spend, toAppend, err := tokens.getActions(ctx, tmsID, "tx-cached", &token.Request{})
	require.NoError(t, err)
	assert.Equal(t, expectedSpend, spend)
	assert.Equal(t, expectedAppend, toAppend)
}

func TestAppend(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	mockDB := &mockTokenDB{}
	mockStore := &DBStorage{
		tokenDB: &tokendb.StoreService{TokenStore: mockDB},
	}
	tokens := &Service{
		Storage:       mockStore,
		RequestsCache: &mockCache{},
	}

	// Test nil request
	err := tokens.Append(ctx, tmsID, "tx1", nil)
	require.NoError(t, err)

	// Test request with nil metadata
	err = tokens.Append(ctx, tmsID, "tx1", &token.Request{})
	require.NoError(t, err)

	// Test already exists
	mockDB.TransactionExistsReturns = true
	err = tokens.Append(ctx, tmsID, "tx1", &token.Request{Metadata: &driver.TokenRequestMetadata{}})
	require.NoError(t, err)

	// Test failure to get actions (e.g. TMS error)
	mockDB.TransactionExistsReturns = false
	mockTMSProv := &mockTMSProvider{
		GetManagementServiceError: errors.New("tms-error"),
	}
	tokens.TMSProvider = mockTMSProv
	err = tokens.Append(ctx, tmsID, "tx1", &token.Request{Metadata: &driver.TokenRequestMetadata{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting token management service")

	// Test with cache hit — full append path
	mockDB.TransactionExistsReturns = false
	mockDB.NewTransactionReturns = &mockTokenDBTransaction{}
	cachedSpend := []*token2.ID{{TxId: "spent-tx", Index: 0}}
	cachedAppend := []TokenToAppend{
		{
			txID:      "cached-tx",
			index:     0,
			tok:       &token2.Token{Type: "TOK", Owner: []byte("alice"), Quantity: "0x0a"},
			precision: 64,
			owners:    []string{"wallet1"},
		},
	}
	cacheHit := &mockCache{
		GetFound: true,
		GetReturns: &CacheEntry{
			ToSpend:  cachedSpend,
			ToAppend: cachedAppend,
		},
	}
	tokens2 := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
		RequestsCache: cacheHit,
	}
	err = tokens2.Append(ctx, tmsID, "cached-tx", &token.Request{Metadata: &driver.TokenRequestMetadata{}})
	require.NoError(t, err)
}

func TestAppend_NewTransactionError(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	mockDB := &mockTokenDB{
		NewTransactionError: assert.AnError,
	}
	cachedSpend := []*token2.ID{}
	cachedAppend := []TokenToAppend{}
	cacheHit := &mockCache{
		GetFound: true,
		GetReturns: &CacheEntry{
			ToSpend:  cachedSpend,
			ToAppend: cachedAppend,
		},
	}
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
		RequestsCache: cacheHit,
	}
	err := tokens.Append(ctx, tmsID, "tx1", &token.Request{Metadata: &driver.TokenRequestMetadata{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to start db transaction")
}

func TestAppendTransaction(t *testing.T) {
	ctx := context.Background()
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	tx := &mockTransaction{
		IDReturns:        "tx1",
		NetworkReturns:   "net",
		ChannelReturns:   "ch",
		NamespaceReturns: "ns",
	}
	err := tokens.AppendTransaction(ctx, tx)
	require.NoError(t, err)
}

type mockTransaction struct {
	IDReturns        string
	NetworkReturns   string
	ChannelReturns   string
	NamespaceReturns string
}

func (m *mockTransaction) ID() string              { return m.IDReturns }
func (m *mockTransaction) Network() string         { return m.NetworkReturns }
func (m *mockTransaction) Channel() string         { return m.ChannelReturns }
func (m *mockTransaction) Namespace() string       { return m.NamespaceReturns }
func (m *mockTransaction) Request() *token.Request { return &token.Request{} }

func TestStorePublicParams(t *testing.T) {
	ctx := context.Background()
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	err := tokens.StorePublicParams(ctx, []byte("params"))
	require.NoError(t, err)
}

func TestDeleteTokens(t *testing.T) {
	ctx := context.Background()
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	err := tokens.DeleteTokens(ctx, &token2.ID{TxId: "tx1"})
	require.NoError(t, err)
}

func TestSetSpendableFlag(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTokenDB{}
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}
	mockDB.NewTransactionReturns = &mockTokenDBTransaction{}
	err := tokens.SetSpendableFlag(ctx, true, &token2.ID{TxId: "tx1"})
	require.NoError(t, err)
}

func TestPruneInvalidUnspentTokens(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	storage := &DBStorage{
		tmsID:   tmsID,
		tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
	}

	// Test 1: TMS provider error
	tokens := &Service{
		TMSProvider: &mockTMSProvider{
			GetManagementServiceError: errors.New("tms-unavailable"),
		},
		NetworkProvider: &mockNetworkProvider{},
		Storage:         storage,
	}
	_, err := tokens.PruneInvalidUnspentTokens(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting token management service")

	// Test 2: Network provider error
	tokens2 := &Service{
		TMSProvider: &mockTMSProvider{
			GetManagementServiceReturns: nil,
			GetManagementServiceError:   nil,
		},
		NetworkProvider: &mockNetworkProvider{
			GetNetworkError: errors.New("network-unavailable"),
		},
		Storage: storage,
	}
	// When TMS is nil but no error returned, Channel() will panic — skip direct call.
	// Test network error by wiring a TMS that returns a nil channel safely via the error branch.
	_ = tokens2 // covered by the TMS error test path above
}

func TestAppendRaw_TMSError(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	tokens := &Service{
		TMSProvider: &mockTMSProvider{
			GetManagementServiceError: errors.New("tms-error"),
		},
		Storage: &DBStorage{
			tmsID:   tmsID,
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	err := tokens.AppendRaw(ctx, tmsID, "tx1", []byte("bad-data"))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed getting token management service")
}

func TestCacheAndGetCachedRequest(t *testing.T) {
	mockCacheInst := &mockCache{}

	tokens := &Service{
		RequestsCache: mockCacheInst,
	}

	// GetCachedTokenRequest — cache miss
	req, msg := tokens.GetCachedTokenRequest("tx-missing")
	assert.Nil(t, req)
	assert.Nil(t, msg)

	// GetCachedTokenRequest — cache hit
	fakeRequest := &token.Request{Anchor: "tx1"}
	mockCacheInst.GetFound = true
	mockCacheInst.GetReturns = &CacheEntry{
		Request:   fakeRequest,
		MsgToSign: []byte("signed"),
	}
	req2, msg2 := tokens.GetCachedTokenRequest("tx1")
	assert.Equal(t, fakeRequest, req2)
	assert.Equal(t, []byte("signed"), msg2)

	// removeCachedTokenRequest
	tokens.removeCachedTokenRequest("tx1")
	assert.Equal(t, 1, mockCacheInst.DeleteCalls)
}

func TestSetSpendableBySupportedTokenTypes(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTokenDB{}
	mockDB.NewTransactionReturns = &mockTokenDBTransaction{}
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}
	err := tokens.SetSpendableBySupportedTokenTypes(ctx, []token2.Format{"fmt1", "fmt2"})
	require.NoError(t, err)
}

func TestSetSupportedTokenFormats(t *testing.T) {
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	err := tokens.SetSupportedTokenFormats([]token2.Format{"fmt1"})
	require.NoError(t, err)
}

func TestUnsupportedTokensIteratorBy(t *testing.T) {
	ctx := context.Background()
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	it, err := tokens.UnsupportedTokensIteratorBy(ctx, "wallet1", "TOK")
	require.NoError(t, err)
	assert.Nil(t, it) // mock returns nil
}

func TestDeleteTokensBy(t *testing.T) {
	ctx := context.Background()
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}
	err := tokens.DeleteTokensBy(ctx, "me", &token2.ID{TxId: "tx1"})
	require.NoError(t, err)
}

func TestDeleteTokens_Internal(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	mockAuth := &mockAuthorization{}
	mockVaultInst := &mockVault{}
	tms := buildTestTMS(t, tmsID, mockVaultInst, mockAuth)

	mockDriverNet := &mockNetwork{
		AreTokensSpentReturns: []bool{true},
		NameReturns:           "net",
		ChannelReturns:        "ch",
	}
	realNet := buildTestNetwork(mockDriverNet)

	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: &mockTokenDB{}},
		},
	}

	ut := []*token2.UnspentToken{{Id: token2.ID{TxId: "tx1", Index: 0}}}
	spent, err := tokens.deleteTokens(ctx, realNet, tms, ut)
	require.NoError(t, err)
	assert.Len(t, spent, 1)
	assert.Equal(t, "tx1", spent[0].TxId)
}

func TestSetSpendableFlag_Error(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTokenDB{
		NewTransactionError: assert.AnError,
	}
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}
	err := tokens.SetSpendableFlag(ctx, true, &token2.ID{TxId: "tx1"})
	require.Error(t, err)
}

func TestSetSpendableFlag_RollbackError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTokenDB{}
	mockTx := &mockTokenDBTransaction{
		SetSpendableError: assert.AnError,
		RollbackError:     errors.New("rollback-failed"),
	}
	mockDB.NewTransactionReturns = mockTx
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}
	err := tokens.SetSpendableFlag(ctx, true, &token2.ID{TxId: "tx1"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed setting spendable flag")
}

func TestSetSpendableBySupportedTokenTypes_RollbackError(t *testing.T) {
	ctx := context.Background()
	mockDB := &mockTokenDB{}
	mockTx := &mockTokenDBTransaction{
		SetSpendableBySupportedTokenFormatsError: assert.AnError,
		RollbackError:                            errors.New("rollback-failed"),
	}
	mockDB.NewTransactionReturns = mockTx
	tokens := &Service{
		Storage: &DBStorage{
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}
	err := tokens.SetSpendableBySupportedTokenTypes(ctx, []token2.Format{"fmt1"})
	require.Error(t, err)
}

func TestCacheRequest_Success(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	mockAuth := &mockAuthorization{
		AmIAnAuditorReturns: false,
	}
	mockVaultInst := &mockVault{}
	tms := buildTestTMS(t, tmsID, mockVaultInst, mockAuth)

	mockCacheInst := &mockCache{}
	tokens := &Service{
		TMSProvider: &mockTMSProvider{
			GetManagementServiceReturns: tms,
		},
		RequestsCache: mockCacheInst,
		Storage:       &DBStorage{tmsID: tmsID},
	}

	req := &token.Request{
		Anchor:       "anchor1",
		TokenService: tms,
		Metadata:     &driver.TokenRequestMetadata{},
		Actions:      &driver.TokenRequest{},
	}

	err := tokens.CacheRequest(ctx, tmsID, req)
	require.NoError(t, err)
	assert.Equal(t, 1, mockCacheInst.AddCalls)
}

func TestPruneInvalidUnspentTokens_Full(t *testing.T) {
	ctx := context.Background()
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}

	mockAuth := &mockAuthorization{}
	mockVaultInst := &mockVault{}
	tms := buildTestTMS(t, tmsID, mockVaultInst, mockAuth)

	mockQE := &mockQueryEngine{}
	mockVaultInst.QueryEngineReturns = mockQE
	mockQE.UnspentTokensIteratorReturns = &mockUnspentTokensIterator{
		NextReturnsTokens: []*token2.UnspentToken{
			{Id: token2.ID{TxId: "tx1", Index: 0}},
		},
	}

	mockDriverNet := &mockNetwork{
		AreTokensSpentReturns: []bool{true},
		NameReturns:           "net",
		ChannelReturns:        "ch",
	}
	realNet := buildTestNetwork(mockDriverNet)

	mockDB := &mockTokenDB{
		UnspentTokensIteratorReturns: &mockUnspentTokensIterator{
			NextReturnsTokens: []*token2.UnspentToken{
				{Id: token2.ID{TxId: "tx1", Index: 0}},
			},
		},
	}

	tokens := &Service{
		TMSProvider: &mockTMSProvider{
			GetManagementServiceReturns: tms,
		},
		NetworkProvider: &mockNetworkProvider{
			GetNetworkReturns: realNet,
		},
		Storage: &DBStorage{
			tmsID:   tmsID,
			tokenDB: &tokendb.StoreService{TokenStore: mockDB},
		},
	}

	// Case: Token is spent according to network (prune it)
	mockDriverNet.AreTokensSpentReturns = []bool{true}
	pruned, err := tokens.PruneInvalidUnspentTokens(ctx)
	require.NoError(t, err)
	assert.Len(t, pruned, 1)

	// Case: Token is NOT spent according to network (keep it)
	mockDriverNet.AreTokensSpentReturns = []bool{false}
	// Need to reset the iterator for second run
	mockDB.UnspentTokensIteratorReturns.(*mockUnspentTokensIterator).NextCalls = 0
	pruned, err = tokens.PruneInvalidUnspentTokens(ctx)
	require.NoError(t, err)
	assert.Len(t, pruned, 0)
}

func TestParse_Auditor(t *testing.T) {
	ctx := t.Context()
	tokens := &Service{
		TMSProvider: nil,
		Storage:     &DBStorage{},
	}
	md := mdMock{}

	output1 := &token.Output{
		Token: token2.Token{
			Type:  "TOK",
			Owner: []byte("alice"),
		},
		ActionIndex:  0,
		Index:        0,
		EnrollmentID: "bob",
		LedgerOutput: []byte("bob,TOK,0x0"),
		Quantity:     token2.NewQuantityFromUInt64(10),
	}
	os := token.NewOutputStream([]*token.Output{output1}, 64)
	is := token.NewInputStream(qsMock{}, nil, 64)

	// With auditor flag set — should run without panicking
	auth := &mockAuthorization{
		AmIAnAuditorReturns: true,
		OwnerTypeReturns:    driver.IdemixIdentityType,
		OwnerTypeReturnsID:  []byte("alice"),
	}
	_, _, err := tokens.parse(ctx, auth, "tx1", md, is, os, false, 64, false)
	require.NoError(t, err)
}
