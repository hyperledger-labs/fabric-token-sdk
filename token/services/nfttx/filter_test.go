/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx_test

import (
	"errors"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/nfttx/nfttxfakes"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

type dummyIterator struct {
	tokens []*token2.UnspentToken
	index  int
	err    error
}

func (d *dummyIterator) Next() (*token2.UnspentToken, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.index >= len(d.tokens) {
		return nil, nil
	}
	t := d.tokens[d.index]
	d.index++
	return t, nil
}

func (d *dummyIterator) Close() {}

type testFilter struct {
	pass bool
}

func (f *testFilter) ContainsToken(token *token2.UnspentToken) bool {
	return f.pass
}

func TestNewFilter(t *testing.T) {
	fakeQS := &nfttxfakes.QueryService{}
	f := nfttx.NewFilter("my-wallet", fakeQS, 64)
	assert.NotNil(t, f)
}

func TestFilterNilFilter(t *testing.T) {
	f := nfttx.NewFilter("my-wallet", nil, 64)
	ids, err := f.Filter(nil, "1")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "filter is nil")
}

func TestFilterIteratorError(t *testing.T) {
	fakeQS := &nfttxfakes.QueryService{}
	fakeQS.UnspentTokensIteratorByReturns(nil, errors.New("iterator error"))

	f := nfttx.NewFilter("my-wallet", fakeQS, 64)
	ids, err := f.Filter(&testFilter{pass: true}, "1")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "token selection failed")
}

func TestFilterNextError(t *testing.T) {
	fakeQS := &nfttxfakes.QueryService{}
	fakeQS.UnspentTokensIteratorByReturns(&dummyIterator{err: errors.New("next error"), tokens: nil}, nil)

	f := nfttx.NewFilter("my-wallet", fakeQS, 64)
	ids, err := f.Filter(&testFilter{pass: true}, "1")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "token selection failed")
}

func TestFilterBadQuantityTarget(t *testing.T) {
	f := nfttx.NewFilter("my-wallet", nil, 64)
	ids, err := f.Filter(&testFilter{pass: true}, "xyz")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.Contains(t, err.Error(), "failed to select tokens: failed to convert quantity")
}

func TestFilterSuccess(t *testing.T) {
	fakeQS := &nfttxfakes.QueryService{}
	tokens := []*token2.UnspentToken{
		{Id: token2.ID{TxId: "tx1", Index: 0}, Quantity: "1"},
		{Id: token2.ID{TxId: "tx2", Index: 0}, Quantity: "1"},
	}
	fakeQS.UnspentTokensIteratorByReturns(&dummyIterator{tokens: tokens}, nil)

	f := nfttx.NewFilter("my-wallet", fakeQS, 64)

	ids, err := f.Filter(&testFilter{pass: true}, "2")
	assert.NoError(t, err)
	assert.Len(t, ids, 2)

	// Filter returning false should be ignored
	fakeQS.UnspentTokensIteratorByReturns(&dummyIterator{tokens: tokens}, nil)
	ids, err = f.Filter(&testFilter{pass: false}, "1")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.ErrorIs(t, err, nfttx.ErrNoResults) // Note here we use unwrapping since err could be "failed to select tokens: no results found"
}

func TestFilterInsufficientTokens(t *testing.T) {
	fakeQS := &nfttxfakes.QueryService{}
	tokens := []*token2.UnspentToken{
		{Id: token2.ID{TxId: "tx1", Index: 0}, Quantity: "1"},
	}
	fakeQS.UnspentTokensIteratorByReturns(&dummyIterator{tokens: tokens}, nil)

	f := nfttx.NewFilter("my-wallet", fakeQS, 64)
	ids, err := f.Filter(&testFilter{pass: true}, "2")
	assert.Error(t, err)
	assert.Nil(t, ids)
	assert.ErrorIs(t, err, nfttx.ErrNoResults)
}
