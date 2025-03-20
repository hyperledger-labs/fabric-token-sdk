/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestTokensService_NewUpgradeChallenge(t *testing.T) {
	ts, err := NewService(nil, 16, nil, nil)
	assert.NoError(t, err)
	challenge, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)
	assert.Len(t, challenge, ChallengeSize)
}

func TestTokensService_GenUpgradeProof(t *testing.T) {
	ts, err := NewService(nil, 16, nil, nil)
	assert.NoError(t, err)
	ch, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)

	invalidTokens := []token.LedgerToken{{
		ID:            token.ID{TxId: "tx1", Index: 1},
		Token:         []byte("token1"),
		TokenMetadata: []byte("meta1"),
		Format:        token.Format("token format1"),
	}}
	// validTokens := []token.LedgerToken{{
	// 	ID:            token.ID{TxId: "tx1", Index: 1},
	// 	Token:         []byte("token1"),
	// 	TokenMetadata: []byte("meta1"),
	// 	Format:        token.Format("token format1"),
	// }}

	tests := []struct {
		name         string
		ch           driver.TokensUpgradeChallenge
		ledgerTokens []token.LedgerToken
		witness      driver.TokensUpgradeWitness
		wantErr      bool
		errMsg       string
		expected     driver.TokensUpgradeProof
	}{
		{
			name:    "challenge size mismatch",
			ch:      []byte{0, 1, 2},
			wantErr: true,
			errMsg:  "invalid challenge size, got [3], expected [32]",
		},
		{
			name:    "no ledger tokens",
			ch:      ch,
			wantErr: true,
			errMsg:  "no ledger tokens provided",
		},
		{
			name:         "no witness",
			ch:           ch,
			witness:      []byte{0, 1, 2},
			ledgerTokens: invalidTokens,
			wantErr:      true,
			errMsg:       "proof witness not expected",
		},
		{
			name:         "unsupported token format",
			ch:           ch,
			ledgerTokens: invalidTokens,
			wantErr:      true,
			errMsg:       "failed to process ledgerTokens upgrade request: unsupported token format [token format1]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := NewService(nil, 16, nil, nil)
			assert.NoError(t, err)
			res, err := ts.GenUpgradeProof(tt.ch, tt.ledgerTokens, tt.witness)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}
		})
	}
}
