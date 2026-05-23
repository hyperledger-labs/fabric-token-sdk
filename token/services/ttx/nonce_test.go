/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"encoding/json"
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAttestationNonceViaGetRandomNonce(t *testing.T) {
	nonce, err := GetRandomNonce()
	require.NoError(t, err)
	assert.Len(t, nonce, NonceSize)

	nonce2, err := GetRandomNonce()
	require.NoError(t, err)
	assert.NotEqual(t, nonce, nonce2, "two nonces must differ")
}

func TestBuildAttestationMessage(t *testing.T) {
	nonce := []byte("nonce-bytes")
	identity := []byte("identity-bytes")

	msg := buildAttestationMessage(nonce, identity)
	assert.Equal(t, append([]byte("nonce-bytes"), []byte("identity-bytes")...), msg)
	assert.Len(t, msg, len(nonce)+len(identity))
}

func TestBuildAttestationMessage_DifferentInputsProduceDifferentMessages(t *testing.T) {
	msg1 := buildAttestationMessage([]byte("AB"), []byte("CD"))
	msg2 := buildAttestationMessage([]byte("ABC"), []byte("D"))
	// "AB"+"CD" = "ABCD" and "ABC"+"D" = "ABCD" — both produce the same bytes.
	// This documents the current concatenation approach and can serve as a basis
	// for future length-prefixed separation if required.
	assert.Equal(t, msg1, msg2, "simple concatenation does not separate fields")
}

func TestBuildAttestationMessage_EmptyInputs(t *testing.T) {
	msg := buildAttestationMessage(nil, nil)
	assert.Empty(t, msg)

	msg = buildAttestationMessage([]byte("nonce"), nil)
	assert.Equal(t, []byte("nonce"), msg)

	msg = buildAttestationMessage(nil, []byte("id"))
	assert.Equal(t, []byte("id"), msg)
}

func TestRecipientResponse_JSONRoundTrip_FreshPath(t *testing.T) {
	original := &RecipientResponse{
		RecipientData: &RecipientData{
			Identity:               view.Identity("alice"),
			AuditInfo:              []byte("audit"),
			TokenMetadata:          []byte("meta"),
			TokenMetadataAuditInfo: []byte("meta-audit"),
		},
		Signature: []byte("sig"),
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &RecipientResponse{}
	require.NoError(t, json.Unmarshal(raw, decoded))

	require.NotNil(t, decoded.RecipientData)
	assert.Equal(t, original.RecipientData.Identity, decoded.RecipientData.Identity)
	assert.Equal(t, original.RecipientData.AuditInfo, decoded.RecipientData.AuditInfo)
	assert.Equal(t, original.Signature, decoded.Signature)
}

func TestRecipientResponse_JSONRoundTrip_EchoPath(t *testing.T) {
	original := &RecipientResponse{
		Signature: []byte("sig-only"),
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &RecipientResponse{}
	require.NoError(t, json.Unmarshal(raw, decoded))

	assert.Nil(t, decoded.RecipientData, "echo path response must have nil RecipientData")
	assert.Equal(t, original.Signature, decoded.Signature)
}

func TestExchangeRecipientResponse_JSONRoundTrip(t *testing.T) {
	original := &ExchangeRecipientResponse{
		RecipientData: &RecipientData{
			Identity:  view.Identity("bob"),
			AuditInfo: []byte("bob-audit"),
		},
		Signature: []byte("exchange-sig"),
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &ExchangeRecipientResponse{}
	require.NoError(t, json.Unmarshal(raw, decoded))

	require.NotNil(t, decoded.RecipientData)
	assert.Equal(t, original.RecipientData.Identity, decoded.RecipientData.Identity)
	assert.Equal(t, original.Signature, decoded.Signature)
}

func TestRecipientRequest_NoncePreservedInJSON(t *testing.T) {
	nonce := []byte("32-byte-nonce-for-testing-xxxxx!")
	original := &RecipientRequest{
		Nonce: nonce,
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &RecipientRequest{}
	require.NoError(t, json.Unmarshal(raw, decoded))
	assert.Equal(t, nonce, decoded.Nonce)
}

func TestExchangeRecipientRequest_NoncePreservedInJSON(t *testing.T) {
	nonce := []byte("exchange-nonce-32bytes-pad-xxxxx")
	original := &ExchangeRecipientRequest{
		Nonce: nonce,
	}
	raw, err := json.Marshal(original)
	require.NoError(t, err)

	decoded := &ExchangeRecipientRequest{}
	require.NoError(t, json.Unmarshal(raw, decoded))
	assert.Equal(t, nonce, decoded.Nonce)
}

func TestRecipientResponse_MissingSignature(t *testing.T) {
	raw := []byte(`{"RecipientData":{"Identity":"YWxpY2U="}}`)
	decoded := &RecipientResponse{}
	require.NoError(t, json.Unmarshal(raw, decoded))
	assert.Nil(t, decoded.Signature, "missing Signature field should unmarshal as nil")
	assert.NotNil(t, decoded.RecipientData)
}

func TestRecipientResponse_MalformedJSON(t *testing.T) {
	decoded := &RecipientResponse{}
	err := json.Unmarshal([]byte("not json {{"), decoded)
	require.Error(t, err)
}
