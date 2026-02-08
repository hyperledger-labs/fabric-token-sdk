/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/stretchr/testify/assert"
)

// TestCleanupExternalWallets_Success tests that CleanupExternalWallets calls Done() on all wallets
func TestCleanupExternalWallets_Success(t *testing.T) {
	wallet := &mock.ExternalWalletSigner{}

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet,
	}

	// Create a minimal view and context
	view := &ttx.CollectEndorsementsView{}
	ctx := &mock.Context{}
	ctx.ContextReturns(t.Context())

	// Call the cleanup method
	view.CleanupExternalWallets(ctx, externalWallets)

	assert.Equal(t, 1, wallet.DoneCallCount(), "Done() should have been called once on the wallet")
}

// TestCleanupExternalWallets_MultipleWallets tests that Done() is called on all wallets
func TestCleanupExternalWallets_MultipleWallets(t *testing.T) {
	wallet1 := &mock.ExternalWalletSigner{}
	wallet2 := &mock.ExternalWalletSigner{}
	wallet3 := &mock.ExternalWalletSigner{}

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet1,
		"wallet2": wallet2,
		"wallet3": wallet3,
	}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock.Context{}
	ctx.ContextReturns(t.Context())

	view.CleanupExternalWallets(ctx, externalWallets)

	assert.Equal(t, 1, wallet1.DoneCallCount(), "Done() should have been called once on wallet1")
	assert.Equal(t, 1, wallet2.DoneCallCount(), "Done() should have been called once on wallet2")
	assert.Equal(t, 1, wallet3.DoneCallCount(), "Done() should have been called once on wallet3")
}

// TestCleanupExternalWallets_MultipleWalletsDoneError tests that errors from Done() don't stop cleanup of multiple wallets
func TestCleanupExternalWallets_MultipleWalletsDoneError(t *testing.T) {
	wallet1 := &mock.ExternalWalletSigner{}
	wallet1.DoneReturns(errors.New("wallet1 done failed"))

	wallet2 := &mock.ExternalWalletSigner{}

	wallet3 := &mock.ExternalWalletSigner{}
	wallet3.DoneReturns(errors.New("wallet3 done failed"))

	externalWallets := map[string]ttx.ExternalWalletSigner{
		"wallet1": wallet1,
		"wallet2": wallet2,
		"wallet3": wallet3,
	}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock.Context{}
	ctx.ContextReturns(t.Context())

	// Should not panic even if Done() returns errors
	view.CleanupExternalWallets(ctx, externalWallets)

	assert.Equal(t, 1, wallet1.DoneCallCount(), "Done() should have been called once on wallet1 despite error")
	assert.Equal(t, 1, wallet2.DoneCallCount(), "Done() should have been called once on wallet2")
	assert.Equal(t, 1, wallet3.DoneCallCount(), "Done() should have been called once on wallet3 despite error")
}

// TestCleanupExternalWallets_EmptyMap tests cleanup with no wallets
func TestCleanupExternalWallets_EmptyMap(t *testing.T) {
	externalWallets := map[string]ttx.ExternalWalletSigner{}

	view := &ttx.CollectEndorsementsView{}
	ctx := &mock.Context{}
	ctx.ContextReturns(t.Context())

	// Should not panic with empty map
	view.CleanupExternalWallets(ctx, externalWallets)
}
