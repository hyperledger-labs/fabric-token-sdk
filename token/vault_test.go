/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/common/utils/collections/iterators"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestQueryEngine_IsMine(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedID := &token.ID{TxId: "a_transaction", Index: 0}
	mockQE.IsMineReturns(true, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	isMine, err := queryEngine.IsMine(expectedID)
	assert.NoError(t, err)
	assert.True(t, isMine)
}

func TestQueryEngine_IsMine_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedErr := errors.New("mock error")
	mockQE.IsMineReturns(false, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	isMine, err := queryEngine.IsMine(nil)
	assert.Error(t, err)
	assert.False(t, isMine)
	assert.Equal(t, expectedErr, err)
}

func TestQueryEngine_ListAuditTokens(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedIDs := []*token.ID{{TxId: "a_transaction", Index: 0}}
	expectedTokens := []*token.Token{{
		Owner:    []byte("some_owner"),
		Type:     "some_type",
		Quantity: "some_quantity",
	}}
	mockQE.ListAuditTokensReturns(expectedTokens, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	tokens, err := queryEngine.ListAuditTokens(expectedIDs...)
	assert.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
	mockQE.ListAuditTokensReturnsOnCall(0, nil, errors.New("pending transactions"))
	mockQE.ListAuditTokensReturnsOnCall(1, expectedTokens, nil)

	tokens, err = queryEngine.ListAuditTokens(expectedIDs...)
	assert.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
	mockQE.ListAuditTokensReturns(nil, errors.New("pending transactions"))

	tokens, err = queryEngine.ListAuditTokens(expectedIDs...)
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.EqualError(t, err, "failed to get audit tokens: pending transactions")
}

func TestQueryEngine_ListAuditTokens_IsPendingTrue(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedIDs := []*token.ID{{TxId: "a_transaction", Index: 0}}
	expectedTokens := []*token.Token{{
		Owner:    []byte("some_owner"),
		Type:     "some_type",
		Quantity: "some_quantity",
	}}
	mockQE.ListAuditTokensReturnsOnCall(0, nil, errors.New("not found"))
	mockQE.ListAuditTokensReturnsOnCall(1, expectedTokens, nil)
	mockQE.IsPendingReturnsOnCall(0, true, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	tokens, err := queryEngine.ListAuditTokens(expectedIDs...)
	assert.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
	assert.Equal(t, 1, mockQE.IsPendingCallCount())
	assert.Equal(t, expectedIDs[0], mockQE.IsPendingArgsForCall(0))
	assert.Equal(t, 2, mockQE.ListAuditTokensCallCount())
}

func TestQueryEngine_ListAuditTokens_IsPendingTrueNumRetries(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedIDs := []*token.ID{{TxId: "a_transaction", Index: 0}}
	mockQE.ListAuditTokensReturnsOnCall(0, nil, errors.New("not found"))
	mockQE.ListAuditTokensReturnsOnCall(1, nil, errors.New("not found"))
	mockQE.ListAuditTokensReturnsOnCall(2, nil, errors.New("not found"))
	mockQE.ListAuditTokensReturnsOnCall(3, nil, errors.New("not found"))
	mockQE.IsPendingReturnsOnCall(0, true, nil)
	mockQE.IsPendingReturnsOnCall(1, true, nil)
	mockQE.IsPendingReturnsOnCall(2, true, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	tokens, err := queryEngine.ListAuditTokens(expectedIDs...)
	assert.Error(t, err)
	assert.Empty(t, tokens)
	assert.Equal(t, 3, mockQE.IsPendingCallCount())
	assert.Equal(t, expectedIDs[0], mockQE.IsPendingArgsForCall(0))
	assert.Equal(t, expectedIDs[0], mockQE.IsPendingArgsForCall(1))
	assert.Equal(t, expectedIDs[0], mockQE.IsPendingArgsForCall(2))
	assert.Equal(t, 3, mockQE.ListAuditTokensCallCount())
}

func TestQueryEngine_UnspentTokensIterator_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedErr := errors.New("mock error")
	mockQE.UnspentTokensIteratorReturns(nil, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	iterator, err := queryEngine.UnspentTokensIterator()
	assert.Error(t, err)
	assert.Nil(t, iterator)
	assert.Equal(t, expectedErr, err)
}

func TestQueryEngine_ListUnspentTokens(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedUnspentTokens := &token.UnspentTokens{}
	mockQE.ListUnspentTokensReturns(expectedUnspentTokens, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	unspentTokens, err := queryEngine.ListUnspentTokens()
	assert.NoError(t, err)
	assert.Equal(t, expectedUnspentTokens, unspentTokens)
}

func TestQueryEngine_ListUnspentTokens_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedErr := errors.New("mock error")
	mockQE.ListUnspentTokensReturns(nil, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	unspentTokens, err := queryEngine.ListUnspentTokens()
	assert.Error(t, err)
	assert.Nil(t, unspentTokens)
	assert.Equal(t, expectedErr, err)
}

func TestQueryEngine_ListHistoryIssuedTokens(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedIssuedTokens := &token.IssuedTokens{}
	mockQE.ListHistoryIssuedTokensReturns(expectedIssuedTokens, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	issuedTokens, err := queryEngine.ListHistoryIssuedTokens()
	assert.NoError(t, err)
	assert.Equal(t, expectedIssuedTokens, issuedTokens)
}

func TestQueryEngine_ListHistoryIssuedTokens_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedErr := errors.New("mock error")
	mockQE.ListHistoryIssuedTokensReturns(nil, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	issuedTokens, err := queryEngine.ListHistoryIssuedTokens()
	assert.Error(t, err)
	assert.Nil(t, issuedTokens)
	assert.Equal(t, expectedErr, err)
}

func TestQueryEngine_PublicParams(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedParams := []byte("public parameters")
	mockQE.PublicParamsReturns(expectedParams, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	params, err := queryEngine.PublicParams()
	assert.NoError(t, err)
	assert.Equal(t, expectedParams, params)
}

func TestQueryEngine_PublicParams_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedErr := errors.New("mock error")
	mockQE.PublicParamsReturns(nil, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	params, err := queryEngine.PublicParams()
	assert.Error(t, err)
	assert.Nil(t, params)
	assert.Equal(t, expectedErr, err)
}

func TestQueryEngine_GetTokens(t *testing.T) {
	mockQE := &mock.QueryEngine{}
	expectedTokens := []*token.Token{{
		Owner:    []byte("some_owner"),
		Type:     "some_type",
		Quantity: "some_quantity",
	}}
	mockQE.GetTokensReturns(expectedTokens, nil)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	tokens, err := queryEngine.GetTokens(nil)
	assert.NoError(t, err)
	assert.Equal(t, expectedTokens, tokens)
}

func TestQueryEngine_GetTokens_Error(t *testing.T) {
	mockQE := &mock.QueryEngine{}

	expectedErr := errors.New("mock error")
	mockQE.GetTokensReturns(nil, expectedErr)

	queryEngine := NewQueryEngine(logging.MustGetLogger("test"), mockQE, 3, time.Second)
	tokens, err := queryEngine.GetTokens(nil)
	assert.Error(t, err)
	assert.Nil(t, tokens)
	assert.Equal(t, expectedErr, err)
}

func TestCertificationStorage_Exists(t *testing.T) {
	mockStorage := &mock.CertificationStorage{}
	id := &token.ID{}
	mockStorage.ExistsReturns(true)

	certStorage := &CertificationStorage{c: mockStorage}
	exists := certStorage.Exists(id)
	assert.True(t, exists, "Expected certification to exist")
	assert.Equal(t, 1, mockStorage.ExistsCallCount(), "Exists method should be called once")
	assert.Equal(t, id, mockStorage.ExistsArgsForCall(0), "Exists method should be called with the correct argument")
}

func TestCertificationStorage_Exists_NotExist(t *testing.T) {
	mockStorage := &mock.CertificationStorage{}
	expectedID := &token.ID{TxId: "a_transaction", Index: 0}
	mockStorage.ExistsReturns(false)

	certStorage := &CertificationStorage{c: mockStorage}
	exists := certStorage.Exists(expectedID)
	assert.False(t, exists, "Expected certification not to exist")
	assert.Equal(t, 1, mockStorage.ExistsCallCount(), "Exists method should be called once")
	assert.Equal(t, expectedID, mockStorage.ExistsArgsForCall(0), "Exists method should be called with the correct argument")
}

func TestCertificationStorage_Store(t *testing.T) {
	mockStorage := &mock.CertificationStorage{}
	certifications := map[*token.ID][]byte{
		{TxId: "a_transaction", Index: 0}:       []byte("cert1"),
		{TxId: "another_transaction", Index: 0}: []byte("cert2"),
	}
	mockStorage.StoreReturns(nil)

	certStorage := &CertificationStorage{c: mockStorage}
	err := certStorage.Store(certifications)
	assert.NoError(t, err, "Expected no error while storing certifications")
	assert.Equal(t, 1, mockStorage.StoreCallCount(), "Store method should be called once")
	assert.Equal(t, certifications, mockStorage.StoreArgsForCall(0), "Store method should be called with the correct argument")
}

func TestCertificationStorage_Store_Error(t *testing.T) {
	mockStorage := &mock.CertificationStorage{}
	certifications := map[*token.ID][]byte{
		{TxId: "a_transaction", Index: 0}:       []byte("cert1"),
		{TxId: "another_transaction", Index: 0}: []byte("cert2"),
	}
	mockErr := errors.New("storage error")
	mockStorage.StoreReturns(mockErr)

	certStorage := &CertificationStorage{c: mockStorage}
	err := certStorage.Store(certifications)
	assert.Error(t, err, "Expected an error while storing certifications")
	assert.EqualError(t, err, mockErr.Error(), "Expected the same error returned by the storage")
	assert.Equal(t, 1, mockStorage.StoreCallCount(), "Store method should be called once")
	assert.Equal(t, certifications, mockStorage.StoreArgsForCall(0), "Store method should be called with the correct argument")
}

func TestUnspentTokensIterator_Sum(t *testing.T) {
	mockIterator := &mock.UnspentTokensIterator{}
	staticToken := &token.UnspentToken{Quantity: "10"}
	staticToken2 := &token.UnspentToken{Quantity: "20"}
	mockIterator.NextReturnsOnCall(0, staticToken, nil)
	mockIterator.NextReturnsOnCall(1, staticToken2, nil)
	mockIterator.NextReturnsOnCall(2, nil, nil)

	sum, err := iterators.Reduce[token.UnspentToken](mockIterator, token.ToQuantitySum(64))
	assert.NoError(t, err, "Expected no error while summing tokens")
	assert.NotNil(t, sum, "Expected a non-nil sum")
	expectedSum := token.NewQuantityFromUInt64(30)
	assert.Equal(t, 0, expectedSum.Cmp(sum), "Expected sum to be equal to 30")
	assert.Equal(t, 3, mockIterator.NextCallCount(), "Next method should be called three times")
	assert.Equal(t, 1, mockIterator.CloseCallCount(), "Close method should be called once")
}

func TestUnspentTokensIterator_Sum_ErrorInNext(t *testing.T) {
	mockIterator := &mock.UnspentTokensIterator{}
	mockErr := errors.New("iterator error")
	mockIterator.NextReturns(nil, mockErr)
	mockIterator.CloseCalls(func() {})

	sum, err := iterators.Reduce[token.UnspentToken](mockIterator, token.ToQuantitySum(64))
	assert.Nil(t, sum, "Expected a nil sum when Next returns an error")
	assert.Error(t, err, "Expected an error when Next returns an error")
	assert.EqualError(t, err, mockErr.Error(), "Expected the same error returned by Next")
	assert.Equal(t, 1, mockIterator.NextCallCount(), "Next method should be called once")
	assert.Equal(t, 1, mockIterator.CloseCallCount(), "Close method should be called")
}

func TestUnspentTokensIterator_Sum_ErrorInToQuantity(t *testing.T) {
	mockIterator := &mock.UnspentTokensIterator{}
	mockIterator.NextReturns(&token.UnspentToken{Quantity: "invalid"}, nil)
	mockIterator.CloseCalls(func() {})

	sum, err := iterators.Reduce[token.UnspentToken](mockIterator, token.ToQuantitySum(64))
	assert.Nil(t, sum, "Expected a nil sum when ToQuantity fails")
	assert.Error(t, err, "Expected an error when ToQuantity fails")
	assert.Equal(t, 1, mockIterator.NextCallCount(), "Next method should be called once")
	assert.Equal(t, 1, mockIterator.CloseCallCount(), "Close method should be called")
}
