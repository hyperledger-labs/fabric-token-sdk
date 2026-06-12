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
		WithRecipientHash([]byte("h")),
		WithSenderHash([]byte("h2")),
		WithHashFunc(crypto.SHA512),
		WithHashEncoding(encoding.Hex),
	)
	require.NoError(t, err)
	require.NotNil(t, opts.Attributes)
	require.Equal(t, []byte("h"), opts.Attributes["hashescrow.recipientHash"])
	require.Equal(t, []byte("h2"), opts.Attributes["hashescrow.senderHash"])
	require.Equal(t, crypto.SHA512, opts.Attributes["hashescrow.hashFunc"])
	require.Equal(t, encoding.Hex, opts.Attributes["hashescrow.hashEncoding"])
}

func TestRecipientAsScript(t *testing.T) {
	tx := &Transaction{}
	sender := []byte("sender")
	recipient := []byte("recipient")

	raw, script, err := tx.recipientAsScript(sender, recipient, []byte("recipient-hash"), []byte("sender-hash"), crypto.SHA256, encoding.Base64)
	require.NoError(t, err)
	require.NotNil(t, raw)
	require.Equal(t, sender, []byte(script.Sender))
	require.Equal(t, recipient, []byte(script.Recipient))
	require.Equal(t, []byte("recipient-hash"), script.RecipientHashInfo.Hash)
	require.Equal(t, []byte("sender-hash"), script.SenderHashInfo.Hash)
}

func TestRecipientAsScriptBadHashInfo(t *testing.T) {
	tx := &Transaction{}
	_, _, err := tx.recipientAsScript(
		[]byte("sender"),
		[]byte("recipient"),
		[]byte("recipient-hash"),
		[]byte("sender-hash"),
		crypto.SHA256,
		encoding.Encoding(999),
	)
	require.Error(t, err)
}
