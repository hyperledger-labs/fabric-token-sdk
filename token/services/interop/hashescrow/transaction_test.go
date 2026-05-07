/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow

import (
	"crypto"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/stretchr/testify/require"
)

func TestCompileTransferOptions(t *testing.T) {
	opts, err := compileTransferOptions(
		WithHash([]byte("h")),
		WithHashFunc(crypto.SHA512),
		WithHashEncoding(encoding.Hex),
	)
	require.NoError(t, err)
	require.NotNil(t, opts.Attributes)
	require.Equal(t, []byte("h"), opts.Attributes["hashescrow.hash"])
	require.Equal(t, crypto.SHA512, opts.Attributes["hashescrow.hashFunc"])
	require.Equal(t, encoding.Hex, opts.Attributes["hashescrow.hashEncoding"])
}

func TestRecipientAsScript(t *testing.T) {
	tx := &Transaction{}
	sender := []byte("sender")
	recipient := []byte("recipient")

	raw, preimage, script, err := tx.recipientAsScript(sender, recipient, []byte("custom-hash"), crypto.SHA256, encoding.Base64)
	require.NoError(t, err)
	require.Empty(t, preimage)
	require.NotNil(t, raw)
	require.Equal(t, sender, []byte(script.Sender))
	require.Equal(t, recipient, []byte(script.Recipient))
	require.Equal(t, []byte("custom-hash"), script.HashInfo.Hash)

	raw, preimage, script, err = tx.recipientAsScript(sender, recipient, nil, crypto.SHA256, encoding.Base64)
	require.NoError(t, err)
	require.NotNil(t, raw)
	require.NotEmpty(t, preimage)
	require.NotEmpty(t, script.HashInfo.Hash)
}

func TestRecipientAsScriptBadEncoding(t *testing.T) {
	tx := &Transaction{}
	_, _, _, err := tx.recipientAsScript(
		[]byte("sender"),
		[]byte("recipient"),
		nil,
		crypto.SHA256,
		encoding.Encoding(999),
	)
	require.EqualError(t, err, "hashEncoding.New() returned nil")
}

func TestCreateNonce(t *testing.T) {
	nonce, err := CreateNonce()
	require.NoError(t, err)
	require.Len(t, nonce, 24)
}
