/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestSerializeAndDeserialize(t *testing.T) {
	// Setup
	p := &Proof{
		Challenge: []byte("test-challenge"),
		Tokens: []token.LedgerToken{{
			ID:            token.ID{TxId: "tx1", Index: 1},
			Token:         []byte("token1"),
			TokenMetadata: []byte("meta1"),
			Format:        token.Format("token format1"),
		}},
		Signatures: []Signature{
			[]byte("sig1"),
		},
	}

	// Test
	data, err := p.Serialize()
	assert.NoError(t, err)

	// deserialize fails
	p2 := &Proof{}
	err = p2.Deserialize(nil)
	assert.Error(t, err)
	p2 = &Proof{}
	err = p2.Deserialize([]byte{1, 2, 3})
	assert.Error(t, err)

	// deserialize ok
	p2 = &Proof{}
	err = p2.Deserialize(data)
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, p, p2)
}

func TestSHA256Digest(t *testing.T) {
	// Setup
	p := &Proof{
		Challenge: []byte("test-challenge"),
		Tokens: []token.LedgerToken{{
			ID:            token.ID{TxId: "tx1", Index: 1},
			Token:         []byte("token1"),
			TokenMetadata: []byte("meta1"),
			Format:        token.Format("token format1"),
		}},
	}

	// Test
	digest, err := p.SHA256Digest()
	assert.NoError(t, err)

	// Assert
	assert.Equal(t, ChallengeSize, len(digest))
	assert.Equal(
		t,
		[]byte{0xe5, 0xd3, 0x26, 0xed, 0x90, 0x21, 0x73, 0xa5, 0x1c, 0x8f, 0xef, 0xdc, 0xab, 0x4, 0xcd, 0x9c, 0xd5, 0xfe, 0x15, 0xcb, 0xe1, 0x3c, 0xb0, 0x75, 0xa5, 0xba, 0x85, 0xde, 0xc4, 0xbe, 0xd4, 0xd5},
		digest,
	)
}

func TestAddSignature(t *testing.T) {
	// Setup
	p := &Proof{}

	// Test
	p.AddSignature(Signature("sig1"))
	p.AddSignature(Signature("sig2"))

	// Assert
	assert.Equal(t, 2, len(p.Signatures))
	assert.Equal(t, Signature("sig1"), p.Signatures[0])
	assert.Equal(t, Signature("sig2"), p.Signatures[1])
}

func TestEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		proof   *Proof
		wantErr bool
	}{
		{
			name: "empty tokens",
			proof: &Proof{
				Challenge: []byte("test"),
				Tokens:    nil,
			},
			wantErr: false,
		},
		{
			name: "nil challenge",
			proof: &Proof{
				Challenge: nil,
				Tokens:    []token.LedgerToken{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.proof.SHA256Digest()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
