/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nfttx

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/stretchr/testify/assert"
)

func TestWrap(t *testing.T) {
	wrapped := Wrap(&ttx.Transaction{})
	assert.NotNil(t, wrapped)
}

func TestReceiveTransaction(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = ReceiveTransaction(nil)
	})
}

func TestNewAnonymousTransaction(t *testing.T) {
	assert.Panics(t, func() {
		_, _ = NewAnonymousTransaction(nil, WithAuditor(view.Identity("id")))
	})
}

func TestWithAuditor(t *testing.T) {
	opt := WithAuditor(view.Identity("id"))
	assert.NotNil(t, opt)
	opts := &ttx.TxOptions{}
	err := opt(opts)
	assert.NoError(t, err)
	assert.Equal(t, view.Identity("id"), opts.Auditor)
}

type mockLinearState struct {
	id string
}

func (m *mockLinearState) SetLinearID(id string) string { m.id = id; return id }

type mockAutoLinearState struct{}

func (m *mockAutoLinearState) GetLinearID() (string, error) { return "123", nil }

func TestTransaction_setStateID(t *testing.T) {
	tx := &Transaction{}

	l := &mockLinearState{}
	id, err := tx.setStateID(l)
	assert.NoError(t, err)
	assert.NotEmpty(t, id)
	assert.Equal(t, id, l.id)

	a := &mockAutoLinearState{}
	id, err = tx.setStateID(a)
	assert.NoError(t, err)
	assert.Equal(t, "123", id)

	id, err = tx.setStateID("unknown state")
	assert.NoError(t, err)
	assert.Empty(t, id)
}

func TestTransaction_Issue(t *testing.T) {
	tx := &Transaction{
		Transaction: &ttx.Transaction{},
	}
	assert.Panics(t, func() {
		tx.Issue(nil, &mockLinearState{id: "1"}, nil)
	})
	err := tx.Issue(nil, make(chan int), nil)
	assert.Error(t, err)
}

func TestTransaction_Transfer(t *testing.T) {
	tx := &Transaction{
		Transaction: &ttx.Transaction{},
	}
	ow := &OwnerWallet{}
	err := tx.Transfer(ow, make(chan int), nil)
	assert.Error(t, err)

	assert.Panics(t, func() {
		tx.Transfer(ow, &mockLinearState{id: "1"}, nil)
	})
}

func TestTransaction_Outputs(t *testing.T) {
	tx := &Transaction{
		Transaction: &ttx.Transaction{},
	}
	assert.Panics(t, func() {
		tx.Outputs()
	})
}
