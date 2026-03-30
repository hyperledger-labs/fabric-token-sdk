/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

// This file tests endorse_opts.go which provides endorsement-specific option patterns.
// Tests verify proper option application, composition, and external wallet signer management.
package ttx_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/ttx/dep/mock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompileCollectEndorsementsOpts_Empty verifies compilation with no options.
func TestCompileCollectEndorsementsOpts_Empty(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts()

	require.NoError(t, err)
	require.NotNil(t, opts)
	assert.False(t, opts.SkipAuditing)
	assert.False(t, opts.SkipAuditorSignatureVerification)
	assert.False(t, opts.SkipApproval)
	assert.False(t, opts.SkipDistributeEnv)
	assert.Nil(t, opts.ExternalWalletSigners)
}

// TestWithSkipAuditing verifies the WithSkipAuditing option.
func TestWithSkipAuditing(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts(ttx.WithSkipAuditing())

	require.NoError(t, err)
	assert.True(t, opts.SkipAuditing)
	assert.False(t, opts.SkipAuditorSignatureVerification)
	assert.False(t, opts.SkipApproval)
	assert.False(t, opts.SkipDistributeEnv)
}

// TestWithSkipAuditorSignatureVerification verifies the WithSkipAuditorSignatureVerification option.
func TestWithSkipAuditorSignatureVerification(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts(ttx.WithSkipAuditorSignatureVerification())

	require.NoError(t, err)
	assert.False(t, opts.SkipAuditing)
	assert.True(t, opts.SkipAuditorSignatureVerification)
	assert.False(t, opts.SkipApproval)
	assert.False(t, opts.SkipDistributeEnv)
}

// TestWithSkipApproval verifies the WithSkipApproval option.
func TestWithSkipApproval(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts(ttx.WithSkipApproval())

	require.NoError(t, err)
	assert.False(t, opts.SkipAuditing)
	assert.False(t, opts.SkipAuditorSignatureVerification)
	assert.True(t, opts.SkipApproval)
	assert.False(t, opts.SkipDistributeEnv)
}

// TestWithSkipDistributeEnv verifies the WithSkipDistributeEnv option.
func TestWithSkipDistributeEnv(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts(ttx.WithSkipDistributeEnv())

	require.NoError(t, err)
	assert.False(t, opts.SkipAuditing)
	assert.False(t, opts.SkipAuditorSignatureVerification)
	assert.False(t, opts.SkipApproval)
	assert.True(t, opts.SkipDistributeEnv)
}

// TestWithExternalWalletSigner verifies the WithExternalWalletSigner option.
func TestWithExternalWalletSigner(t *testing.T) {
	signer := &mock.ExternalWalletSigner{}
	walletID := "wallet-123"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner(walletID, signer),
	)

	require.NoError(t, err)
	require.NotNil(t, opts.ExternalWalletSigners)
	assert.Equal(t, signer, opts.ExternalWalletSigners[walletID])
}

// TestWithExternalWalletSigner_Multiple verifies multiple external wallet signers.
func TestWithExternalWalletSigner_Multiple(t *testing.T) {
	signer1 := &mock.ExternalWalletSigner{}
	signer2 := &mock.ExternalWalletSigner{}
	walletID1 := "wallet-1"
	walletID2 := "wallet-2"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner(walletID1, signer1),
		ttx.WithExternalWalletSigner(walletID2, signer2),
	)

	require.NoError(t, err)
	require.NotNil(t, opts.ExternalWalletSigners)
	assert.Len(t, opts.ExternalWalletSigners, 2)
	assert.Equal(t, signer1, opts.ExternalWalletSigners[walletID1])
	assert.Equal(t, signer2, opts.ExternalWalletSigners[walletID2])
}

// TestWithExternalWalletSigner_Override verifies that later signers override earlier ones.
func TestWithExternalWalletSigner_Override(t *testing.T) {
	signer1 := &mock.ExternalWalletSigner{}
	signer2 := &mock.ExternalWalletSigner{}
	walletID := "wallet-same"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner(walletID, signer1),
		ttx.WithExternalWalletSigner(walletID, signer2),
	)

	require.NoError(t, err)
	require.NotNil(t, opts.ExternalWalletSigners)
	assert.Len(t, opts.ExternalWalletSigners, 1)
	assert.Equal(t, signer2, opts.ExternalWalletSigners[walletID])
}

// TestEndorsementsOpts_ExternalWalletSigner_Found verifies retrieval of existing signer.
func TestEndorsementsOpts_ExternalWalletSigner_Found(t *testing.T) {
	signer := &mock.ExternalWalletSigner{}
	walletID := "wallet-abc"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner(walletID, signer),
	)
	require.NoError(t, err)

	result := opts.ExternalWalletSigner(walletID)
	assert.Equal(t, signer, result)
}

// TestEndorsementsOpts_ExternalWalletSigner_NotFound verifies nil for missing signer.
func TestEndorsementsOpts_ExternalWalletSigner_NotFound(t *testing.T) {
	signer := &mock.ExternalWalletSigner{}

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner("wallet-1", signer),
	)
	require.NoError(t, err)

	result := opts.ExternalWalletSigner("wallet-2")
	assert.Nil(t, result)
}

// TestEndorsementsOpts_ExternalWalletSigner_NilMap verifies nil when map is nil.
func TestEndorsementsOpts_ExternalWalletSigner_NilMap(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts()
	require.NoError(t, err)

	result := opts.ExternalWalletSigner("any-wallet")
	assert.Nil(t, result)
}

// TestCompileCollectEndorsementsOpts_AllOptions verifies all options can be combined.
func TestCompileCollectEndorsementsOpts_AllOptions(t *testing.T) {
	signer := &mock.ExternalWalletSigner{}
	walletID := "wallet-all"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithSkipAuditing(),
		ttx.WithSkipAuditorSignatureVerification(),
		ttx.WithSkipApproval(),
		ttx.WithSkipDistributeEnv(),
		ttx.WithExternalWalletSigner(walletID, signer),
	)

	require.NoError(t, err)
	assert.True(t, opts.SkipAuditing)
	assert.True(t, opts.SkipAuditorSignatureVerification)
	assert.True(t, opts.SkipApproval)
	assert.True(t, opts.SkipDistributeEnv)
	require.NotNil(t, opts.ExternalWalletSigners)
	assert.Equal(t, signer, opts.ExternalWalletSigners[walletID])
}

// TestCompileCollectEndorsementsOpts_PartialOptions verifies partial option combinations.
func TestCompileCollectEndorsementsOpts_PartialOptions(t *testing.T) {
	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithSkipAuditing(),
		ttx.WithSkipApproval(),
	)

	require.NoError(t, err)
	assert.True(t, opts.SkipAuditing)
	assert.False(t, opts.SkipAuditorSignatureVerification)
	assert.True(t, opts.SkipApproval)
	assert.False(t, opts.SkipDistributeEnv)
	assert.Nil(t, opts.ExternalWalletSigners)
}

// TestEndorsementsOpts_ExternalWalletSigner_EmptyID verifies behavior with empty wallet ID.
func TestEndorsementsOpts_ExternalWalletSigner_EmptyID(t *testing.T) {
	signer := &mock.ExternalWalletSigner{}

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner("", signer),
	)
	require.NoError(t, err)

	result := opts.ExternalWalletSigner("")
	assert.Equal(t, signer, result)
}

// TestWithExternalWalletSigner_NilSigner verifies nil signer can be set.
func TestWithExternalWalletSigner_NilSigner(t *testing.T) {
	walletID := "wallet-nil"

	opts, err := ttx.CompileCollectEndorsementsOpts(
		ttx.WithExternalWalletSigner(walletID, nil),
	)

	require.NoError(t, err)
	require.NotNil(t, opts.ExternalWalletSigners)
	assert.Nil(t, opts.ExternalWalletSigners[walletID])
}
