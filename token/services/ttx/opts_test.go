/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"testing"
	"time"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/network"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompileOpts(t *testing.T) {
	auditor := view.Identity("auditor-id")
	timeout := 5 * time.Minute
	txID := "test-tx-id"
	tmsID := token.TMSID{Network: "net", Channel: "ch", Namespace: "ns"}
	tx := &Transaction{}
	networkTxID := network.TxID{Nonce: []byte("nonce"), Creator: []byte("creator")}

	opts, err := CompileOpts(
		WithAuditor(auditor),
		WithNetwork("net"),
		WithChannel("ch"),
		WithNamespace("ns"),
		WithNoCachingRequest(),
		WithNoTransactionVerification(),
		WithTimeout(timeout),
		WithTxID(txID),
		WithAnonymousTransaction(true),
		WithTransactions(tx),
		WithNetworkTxID(networkTxID),
	)
	require.NoError(t, err)
	assert.Equal(t, auditor, opts.Auditor)
	assert.Equal(t, "net", opts.TMSID.Network)
	assert.Equal(t, "ch", opts.TMSID.Channel)
	assert.Equal(t, "ns", opts.TMSID.Namespace)
	assert.True(t, opts.NoCachingRequest)
	assert.True(t, opts.NoTransactionVerification)
	assert.Equal(t, timeout, opts.Timeout)
	assert.Equal(t, txID, opts.TxID)
	assert.True(t, opts.AnonymousTransaction)
	assert.Equal(t, tx, opts.Transaction)
	assert.Equal(t, networkTxID, opts.NetworkTxID)

	// Additional tests for other options
	opts, err = CompileOpts(WithTMS("net2", "ch2", "ns2"))
	require.NoError(t, err)
	assert.Equal(t, "net2", opts.TMSID.Network)
	assert.Equal(t, "ch2", opts.TMSID.Channel)
	assert.Equal(t, "ns2", opts.TMSID.Namespace)

	opts, err = CompileOpts(WithTMSID(tmsID))
	require.NoError(t, err)
	assert.Equal(t, tmsID, opts.TMSID)

	opts, err = CompileOpts(WithTMSIDPointer(&tmsID))
	require.NoError(t, err)
	assert.Equal(t, tmsID, opts.TMSID)

	opts, err = CompileOpts(WithTMSIDPointer(nil))
	require.NoError(t, err)
	assert.Empty(t, opts.TMSID)

	// Test error
	expectedErr := assert.AnError
	opts, err = CompileOpts(func(o *TxOptions) error {
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, opts)
}

func TestCompileServiceOptions(t *testing.T) {
	recipientData := &RecipientData{}
	walletID := "wallet-id"

	opts, err := CompileServiceOptions(
		WithRecipientData(recipientData),
		WithRecipientWalletID(walletID),
	)
	require.NoError(t, err)
	assert.NotNil(t, opts.Params)
	assert.Equal(t, recipientData, opts.Params["RecipientData"])
	assert.Equal(t, walletID, opts.Params["RecipientWalletID"])

	// Test helper getRecipientWalletID
	assert.Equal(t, walletID, getRecipientWalletID(opts))
	assert.Equal(t, "", getRecipientWalletID(&token.ServiceOptions{}))

	// Empty wallet ID should not add anything
	opts, err = CompileServiceOptions(WithRecipientWalletID(""))
	require.NoError(t, err)
	assert.Nil(t, opts.Params)

	// Test error
	expectedErr := assert.AnError
	opts, err = CompileServiceOptions(func(o *token.ServiceOptions) error {
		return expectedErr
	})
	require.ErrorIs(t, err, expectedErr)
	assert.Nil(t, opts)
}
