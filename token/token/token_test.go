/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
	"github.com/test-go/testify/require"
)

// ── ID ───────────────────────────────────────────────────────────────────────

func TestID_Equal(t *testing.T) {
	tests := []struct {
		name     string
		left     token.ID
		right    token.ID
		expected bool
	}{
		{"equal IDs", token.ID{TxId: "tx1", Index: 0}, token.ID{TxId: "tx1", Index: 0}, true},
		{"different TxId", token.ID{TxId: "tx1", Index: 0}, token.ID{TxId: "tx2", Index: 0}, false},
		{"different Index", token.ID{TxId: "tx1", Index: 0}, token.ID{TxId: "tx1", Index: 1}, false},
		{"both different", token.ID{TxId: "tx1", Index: 0}, token.ID{TxId: "tx2", Index: 1}, false},
		{"empty IDs are equal", token.ID{}, token.ID{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.left.Equal(tt.right))
		})
	}
}

func TestID_String(t *testing.T) {
	tests := []struct {
		name     string
		id       token.ID
		expected string
	}{
		{"non-empty ID", token.ID{TxId: "abc123", Index: 2}, "[abc123:2]"},
		{"zero index", token.ID{TxId: "tx1", Index: 0}, "[tx1:0]"},
		{"empty TxId", token.ID{TxId: "", Index: 5}, "[:5]"},
		{"empty ID", token.ID{}, "[:0]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.id.String())
		})
	}
}

// ── LedgerToken ──────────────────────────────────────────────────────────────

func TestLedgerToken_Equal(t *testing.T) {
	base := token.LedgerToken{
		ID:            token.ID{TxId: "tx1", Index: 0},
		Format:        "fabtoken",
		Token:         []byte("token-data"),
		TokenMetadata: []byte("meta-data"),
	}
	tests := []struct {
		name     string
		right    token.LedgerToken
		expected bool
	}{
		{"equal tokens", base, true},
		{"different ID", token.LedgerToken{ID: token.ID{TxId: "tx2"}, Format: "fabtoken", Token: []byte("token-data"), TokenMetadata: []byte("meta-data")}, false},
		{"different Format", token.LedgerToken{ID: token.ID{TxId: "tx1"}, Format: "comm", Token: []byte("token-data"), TokenMetadata: []byte("meta-data")}, false},
		{"different Token bytes", token.LedgerToken{ID: token.ID{TxId: "tx1"}, Format: "fabtoken", Token: []byte("other"), TokenMetadata: []byte("meta-data")}, false},
		{"different TokenMetadata", token.LedgerToken{ID: token.ID{TxId: "tx1"}, Format: "fabtoken", Token: []byte("token-data"), TokenMetadata: []byte("other")}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, base.Equal(tt.right))
		})
	}
	t.Run("two empty tokens are equal", func(t *testing.T) {
		assert.True(t, token.LedgerToken{}.Equal(token.LedgerToken{}))
	})
}

// ── UnspentToken ─────────────────────────────────────────────────────────────

func TestUnspentToken_String(t *testing.T) {
	tests := []struct {
		name     string
		tok      token.UnspentToken
		expected string
	}{
		{"standard token", token.UnspentToken{Id: token.ID{TxId: "tx1", Index: 0}}, "[tx1:0]"},
		{"token with index", token.UnspentToken{Id: token.ID{TxId: "abc", Index: 3}}, "[abc:3]"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.tok.String())
		})
	}
}

// ── UnspentTokens ────────────────────────────────────────────────────────────

func TestUnspentTokens_Count(t *testing.T) {
	assert.Equal(t, 0, (&token.UnspentTokens{}).Count())
	assert.Equal(t, 1, (&token.UnspentTokens{Tokens: []*token.UnspentToken{{}}}).Count())
	assert.Equal(t, 3, (&token.UnspentTokens{Tokens: []*token.UnspentToken{{}, {}, {}}}).Count())
}

func TestUnspentTokens_ByType(t *testing.T) {
	tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{
		{Id: token.ID{TxId: "tx1"}, Type: "USD", Quantity: "0x1"},
		{Id: token.ID{TxId: "tx2"}, Type: "EUR", Quantity: "0x2"},
		{Id: token.ID{TxId: "tx3"}, Type: "USD", Quantity: "0x3"},
	}}
	t.Run("filter USD returns 2 tokens", func(t *testing.T) {
		result := tokens.ByType("USD")
		require.Equal(t, 2, result.Count())
		assert.Equal(t, token.Type("USD"), result.Tokens[0].Type)
		assert.Equal(t, token.Type("USD"), result.Tokens[1].Type)
	})
	t.Run("filter EUR returns 1 token", func(t *testing.T) {
		result := tokens.ByType("EUR")
		require.Equal(t, 1, result.Count())
		assert.Equal(t, token.Type("EUR"), result.Tokens[0].Type)
	})
	t.Run("filter non-existent type returns empty", func(t *testing.T) {
		assert.Equal(t, 0, tokens.ByType("GBP").Count())
	})
	t.Run("filter on empty list returns empty", func(t *testing.T) {
		assert.Equal(t, 0, (&token.UnspentTokens{}).ByType("USD").Count())
	})
}

func TestUnspentTokens_Sum(t *testing.T) {
	t.Run("sum with precision 64", func(t *testing.T) {
		tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{
			{Quantity: "0x1"}, {Quantity: "0x2"}, {Quantity: "0x3"},
		}}
		assert.Equal(t, "6", tokens.Sum(64).Decimal())
	})
	t.Run("sum with precision 128", func(t *testing.T) {
		tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{
			{Quantity: "0xa"}, {Quantity: "0x5"},
		}}
		assert.Equal(t, "15", tokens.Sum(128).Decimal())
	})
	t.Run("sum of empty list is zero", func(t *testing.T) {
		assert.Equal(t, "0", (&token.UnspentTokens{}).Sum(64).Decimal())
	})
	t.Run("sum of single token", func(t *testing.T) {
		tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{{Quantity: "0x64"}}}
		assert.Equal(t, "100", tokens.Sum(64).Decimal())
	})
	t.Run("invalid quantity panics", func(t *testing.T) {
		tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{{Quantity: "invalid"}}}
		assert.Panics(t, func() { tokens.Sum(64) })
	})
}

func TestUnspentTokens_At(t *testing.T) {
	tok1 := &token.UnspentToken{Id: token.ID{TxId: "tx1"}, Type: "USD"}
	tok2 := &token.UnspentToken{Id: token.ID{TxId: "tx2"}, Type: "EUR"}
	tokens := &token.UnspentTokens{Tokens: []*token.UnspentToken{tok1, tok2}}
	assert.Equal(t, tok1, tokens.At(0))
	assert.Equal(t, tok2, tokens.At(1))
}

// ── IssuedTokens ─────────────────────────────────────────────────────────────

func TestIssuedTokens_Count(t *testing.T) {
	assert.Equal(t, 0, (&token.IssuedTokens{}).Count())
	assert.Equal(t, 1, (&token.IssuedTokens{Tokens: []*token.IssuedToken{{}}}).Count())
	assert.Equal(t, 2, (&token.IssuedTokens{Tokens: []*token.IssuedToken{{}, {}}}).Count())
}

func TestIssuedTokens_ByType(t *testing.T) {
	tokens := &token.IssuedTokens{Tokens: []*token.IssuedToken{
		{Id: token.ID{TxId: "tx1"}, Type: "USD", Quantity: "0x1"},
		{Id: token.ID{TxId: "tx2"}, Type: "EUR", Quantity: "0x2"},
		{Id: token.ID{TxId: "tx3"}, Type: "USD", Quantity: "0x3"},
	}}
	t.Run("filter USD returns 2 tokens", func(t *testing.T) {
		result := tokens.ByType("USD")
		require.Equal(t, 2, result.Count())
		assert.Equal(t, token.Type("USD"), result.Tokens[0].Type)
	})
	t.Run("filter EUR returns 1 token", func(t *testing.T) {
		require.Equal(t, 1, tokens.ByType("EUR").Count())
	})
	t.Run("filter non-existent type returns empty", func(t *testing.T) {
		assert.Equal(t, 0, tokens.ByType("GBP").Count())
	})
	t.Run("filter on empty list returns empty", func(t *testing.T) {
		assert.Equal(t, 0, (&token.IssuedTokens{}).ByType("USD").Count())
	})
}

func TestIssuedTokens_Sum(t *testing.T) {
	t.Run("sum with precision 64", func(t *testing.T) {
		tokens := &token.IssuedTokens{Tokens: []*token.IssuedToken{
			{Quantity: "0x1"}, {Quantity: "0x2"}, {Quantity: "0x3"},
		}}
		assert.Equal(t, "6", tokens.Sum(64).Decimal())
	})
	t.Run("sum with precision 128", func(t *testing.T) {
		tokens := &token.IssuedTokens{Tokens: []*token.IssuedToken{
			{Quantity: "0xa"}, {Quantity: "0x5"},
		}}
		assert.Equal(t, "15", tokens.Sum(128).Decimal())
	})
	t.Run("sum of empty list is zero", func(t *testing.T) {
		assert.Equal(t, "0", (&token.IssuedTokens{}).Sum(64).Decimal())
	})
	t.Run("invalid quantity panics", func(t *testing.T) {
		tokens := &token.IssuedTokens{Tokens: []*token.IssuedToken{{Quantity: "invalid"}}}
		assert.Panics(t, func() { tokens.Sum(64) })
	})
}
