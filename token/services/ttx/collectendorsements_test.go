/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/stretchr/testify/assert"
)

// mockExternalWalletSigner is a mock implementation of ExternalWalletSigner for testing
type mockExternalWalletSigner struct {
	signFunc   func(party view.Identity, message []byte) ([]byte, error)
	doneFunc   func() error
	doneCalled bool
	signCalled bool
}

func (m *mockExternalWalletSigner) Sign(party view.Identity, message []byte) ([]byte, error) {
	m.signCalled = true
	if m.signFunc != nil {
		return m.signFunc(party, message)
	}
	return []byte("mock_signature"), nil
}

func (m *mockExternalWalletSigner) Done() error {
	m.doneCalled = true
	if m.doneFunc != nil {
		return m.doneFunc()
	}
	return nil
}

// TestCleanupExternalWallets_Success tests that CleanupExternalWallets calls Done() on all wallets
func TestCleanupExternalWallets_Success(t *testing.T) {
	wallet := &mockExternalWalletSigner{}

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet,
	}

	// Create a minimal view and context
	view := &ttx.CollectEndorsementsView{}
	ctx := &mock2.Context{}
	ctx.ContextReturns(t.Context())

	// Call the cleanup method
	view.CleanupExternalWallets(ctx, externalWallets)

	assert.True(t, wallet.doneCalled, "Done() should have been called on the wallet")
}

// TestCleanupExternalWallets_MultipleWallets tests that Done() is called on all wallets
func TestCleanupExternalWallets_MultipleWallets(t *testing.T) {
	wallet1 := &mockExternalWalletSigner{}
	wallet2 := &mockExternalWalletSigner{}
	wallet3 := &mockExternalWalletSigner{}

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet1,
		"wallet2": wallet2,
		"wallet3": wallet3,
	}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock2.Context{}
	ctx.ContextReturns(t.Context())

	view.CleanupExternalWallets(ctx, externalWallets)

	assert.True(t, wallet1.doneCalled, "Done() should have been called on wallet1")
	assert.True(t, wallet2.doneCalled, "Done() should have been called on wallet2")
	assert.True(t, wallet3.doneCalled, "Done() should have been called on wallet3")
}

// TestCleanupExternalWallets_DoneError tests that errors from Done() don't stop cleanup
func TestCleanupExternalWallets_DoneError(t *testing.T) {
	wallet1 := &mockExternalWalletSigner{
		doneFunc: func() error {
			return errors.New("wallet1 done failed")
		},
	}
	wallet2 := &mockExternalWalletSigner{}
	wallet3 := &mockExternalWalletSigner{
		doneFunc: func() error {
			return errors.New("wallet3 done failed")
		},
	}

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet1,
		"wallet2": wallet2,
		"wallet3": wallet3,
	}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock2.Context{}
	ctx.ContextReturns(t.Context())

	// Should not panic even if Done() returns errors
	view.CleanupExternalWallets(ctx, externalWallets)

	assert.True(t, wallet1.doneCalled, "Done() should have been called on wallet1 despite error")
	assert.True(t, wallet2.doneCalled, "Done() should have been called on wallet2")
	assert.True(t, wallet3.doneCalled, "Done() should have been called on wallet3 despite error")
}

// TestCleanupExternalWallets_EmptyMap tests cleanup with no wallets
func TestCleanupExternalWallets_EmptyMap(t *testing.T) {
	externalWallets := map[string]ttx.ExternalWalletSigner{}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock2.Context{}
	ctx.ContextReturns(t.Context())

	// Should not panic with empty map
	view.CleanupExternalWallets(ctx, externalWallets)
}
