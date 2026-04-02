/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ttx

import (
	"testing"

	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecipients_Identities(t *testing.T) {
	tests := []struct {
		name     string
		input    Recipients
		expected []view.Identity
	}{
		{
			name:     "empty recipients",
			input:    Recipients{},
			expected: []view.Identity{},
		},
		{
			name: "single recipient",
			input: Recipients{
				{Identity: view.Identity("alice")},
			},
			expected: []view.Identity{view.Identity("alice")},
		},
		{
			name: "multiple recipients preserve order",
			input: Recipients{
				{Identity: view.Identity("alice")},
				{Identity: view.Identity("bob")},
				{Identity: view.Identity("charlie")},
			},
			expected: []view.Identity{
				view.Identity("alice"),
				view.Identity("bob"),
				view.Identity("charlie"),
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tc.input.Identities()
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestExchangeRecipientRequest_BytesRoundTrip(t *testing.T) {
	original := &ExchangeRecipientRequest{
		TMSID:    token.TMSID{Network: "net1", Channel: "ch1", Namespace: "ns1"},
		WalletID: []byte("my-wallet"),
		RecipientData: &RecipientData{
			Identity:               view.Identity("alice"),
			AuditInfo:              []byte("audit-info"),
			TokenMetadata:          []byte("token-meta"),
			TokenMetadataAuditInfo: []byte("meta-audit"),
		},
	}

	raw, err := original.Bytes()
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	decoded := &ExchangeRecipientRequest{}
	require.NoError(t, decoded.FromBytes(raw))

	assert.Equal(t, original.TMSID, decoded.TMSID)
	assert.Equal(t, original.WalletID, decoded.WalletID)
	require.NotNil(t, decoded.RecipientData)
	assert.Equal(t, original.RecipientData.Identity, decoded.RecipientData.Identity)
	assert.Equal(t, original.RecipientData.AuditInfo, decoded.RecipientData.AuditInfo)
	assert.Equal(t, original.RecipientData.TokenMetadata, decoded.RecipientData.TokenMetadata)
	assert.Equal(t, original.RecipientData.TokenMetadataAuditInfo, decoded.RecipientData.TokenMetadataAuditInfo)
}

func TestExchangeRecipientRequest_FromBytes_InvalidInput(t *testing.T) {
	r := &ExchangeRecipientRequest{}
	err := r.FromBytes([]byte("not valid json {{"))
	require.Error(t, err)
}

func TestRecipientRequest_BytesRoundTrip(t *testing.T) {
	original := &RecipientRequest{
		TMSID:    token.TMSID{Network: "net2", Channel: "ch2", Namespace: "ns2"},
		WalletID: []byte("wallet-id"),
		RecipientData: &RecipientData{
			Identity:  view.Identity("bob"),
			AuditInfo: []byte("bob-audit"),
		},
		MultiSig: true,
	}

	raw, err := original.Bytes()
	require.NoError(t, err)
	require.NotEmpty(t, raw)

	decoded := &RecipientRequest{}
	require.NoError(t, decoded.FromBytes(raw))

	assert.Equal(t, original.TMSID, decoded.TMSID)
	assert.Equal(t, original.WalletID, decoded.WalletID)
	assert.Equal(t, original.MultiSig, decoded.MultiSig)
	require.NotNil(t, decoded.RecipientData)
	assert.Equal(t, original.RecipientData.Identity, decoded.RecipientData.Identity)
	assert.Equal(t, original.RecipientData.AuditInfo, decoded.RecipientData.AuditInfo)
}

func TestRecipientRequest_FromBytes_InvalidInput(t *testing.T) {
	r := &RecipientRequest{}
	err := r.FromBytes([]byte("not valid json {{"))
	require.Error(t, err)
}

func TestGetRecipientData(t *testing.T) {
	t.Run("nil params map returns nil", func(t *testing.T) {
		opts := &token.ServiceOptions{}
		assert.Nil(t, getRecipientData(opts))
	})

	t.Run("missing key returns nil", func(t *testing.T) {
		opts := &token.ServiceOptions{
			Params: map[string]interface{}{
				"SomeOtherKey": "value",
			},
		}
		assert.Nil(t, getRecipientData(opts))
	})

	t.Run("key present returns recipient data", func(t *testing.T) {
		rd := &RecipientData{
			Identity:  view.Identity("alice"),
			AuditInfo: []byte("audit"),
		}
		opts := &token.ServiceOptions{
			Params: map[string]interface{}{
				"RecipientData": rd,
			},
		}
		assert.Equal(t, rd, getRecipientData(opts))
	})
}

func TestGetRecipientWalletID(t *testing.T) {
	t.Run("nil params map returns empty string", func(t *testing.T) {
		opts := &token.ServiceOptions{}
		assert.Empty(t, getRecipientWalletID(opts))
	})

	t.Run("missing key returns empty string", func(t *testing.T) {
		opts := &token.ServiceOptions{
			Params: map[string]interface{}{
				"SomeOtherKey": "value",
			},
		}
		assert.Empty(t, getRecipientWalletID(opts))
	})

	t.Run("key present returns wallet id", func(t *testing.T) {
		opts := &token.ServiceOptions{
			Params: map[string]interface{}{
				"RecipientWalletID": "my-wallet-id",
			},
		}
		assert.Equal(t, "my-wallet-id", getRecipientWalletID(opts))
	})
}
