/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package hashescrow_test

import (
	"crypto"
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/encoding"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/hashescrow"
	"github.com/stretchr/testify/require"
)

func mkScriptForClaims(t *testing.T) (*hashescrow.Script, []byte, []byte) {
	t.Helper()

	s := &hashescrow.Script{
		Sender:    []byte("sender"),
		Recipient: []byte("recipient"),
		RecipientHashInfo: hashescrow.HashInfo{
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
		SenderHashInfo: hashescrow.HashInfo{
			HashFunc:     crypto.SHA256,
			HashEncoding: encoding.Base64,
		},
	}

	recipientPreImage := []byte("recipient-preimage")
	recipientImage, err := s.RecipientHashInfo.Image(recipientPreImage)
	require.NoError(t, err)
	s.RecipientHashInfo.Hash = recipientImage

	senderPreImage := []byte("sender-preimage")
	senderImage, err := s.SenderHashInfo.Image(senderPreImage)
	require.NoError(t, err)
	s.SenderHashInfo.Hash = senderImage

	return s, recipientPreImage, senderPreImage
}

func TestClaimSignerSign(t *testing.T) {
	cs := &hashescrow.ClaimSigner{Preimage: []byte("pre")}
	raw, err := cs.Sign([]byte("ignored-msg"))
	require.NoError(t, err)

	var sig hashescrow.ClaimSignature
	require.NoError(t, json.Unmarshal(raw, &sig))
	require.Equal(t, []byte("pre"), sig.Preimage)
}

func TestClaimVerifierVerify(t *testing.T) {
	s, recipientPreImage, senderPreImage := mkScriptForClaims(t)
	cv := &hashescrow.ClaimVerifier{Script: s}

	recipientRaw, err := json.Marshal(&hashescrow.ClaimSignature{Preimage: recipientPreImage})
	require.NoError(t, err)
	require.NoError(t, cv.Verify([]byte("msg"), recipientRaw))

	senderRaw, err := json.Marshal(&hashescrow.ClaimSignature{Preimage: senderPreImage})
	require.NoError(t, err)
	require.NoError(t, cv.Verify([]byte("msg"), senderRaw))

	wrongRaw, err := json.Marshal(&hashescrow.ClaimSignature{Preimage: []byte("wrong")})
	require.NoError(t, err)
	require.Error(t, cv.Verify([]byte("msg"), wrongRaw))

	require.Error(t, cv.Verify([]byte("msg"), []byte("bad-json")))
	require.Error(t, (&hashescrow.ClaimVerifier{}).Verify([]byte("msg"), recipientRaw))
}

func TestVerifierVerify(t *testing.T) {
	s, recipientPreImage, _ := mkScriptForClaims(t)
	v := &hashescrow.Verifier{Script: s}

	sigma, err := json.Marshal(&hashescrow.ClaimSignature{Preimage: recipientPreImage})
	require.NoError(t, err)
	require.NoError(t, v.Verify([]byte("msg"), sigma))

	badSigma, err := json.Marshal(&hashescrow.ClaimSignature{Preimage: []byte("wrong")})
	require.NoError(t, err)
	err = v.Verify([]byte("msg"), badSigma)
	require.Error(t, err)
	require.Contains(t, err.Error(), "failed verifying hash escrow claim signature")
}
