/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade_test

import (
	"testing"

	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1/actions"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/upgrade"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/upgrade/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	mock2 "github.com/hyperledger-labs/fabric-token-sdk/token/driver/mock"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTokensService_NewUpgradeChallenge(t *testing.T) {
	ts, err := upgrade.NewService(nil, 16, nil, nil)
	assert.NoError(t, err)
	challenge, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)
	assert.Len(t, challenge, upgrade.ChallengeSize)
}

func TestTokensService_GenUpgradeProof(t *testing.T) {
	ts, err := upgrade.NewService(nil, 16, nil, nil)
	assert.NoError(t, err)
	ch, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)

	invalidTokens := []token.LedgerToken{{
		ID:            token.ID{TxId: "tx1", Index: 1},
		Token:         []byte("token1"),
		TokenMetadata: []byte("meta1"),
		Format:        token.Format("token format1"),
	}}
	fabtokenOutput := actions.Output{
		Owner:    []byte("owner1"),
		Type:     "token type",
		Quantity: "10",
	}
	fabtokenOutputRaw, err := fabtokenOutput.Serialize()
	assert.NoError(t, err)
	formatFabtoken16, err := v1.SupportedTokenFormat(16)
	assert.NoError(t, err)
	validTokens := []token.LedgerToken{{
		ID:            token.ID{TxId: "tx1", Index: 1},
		Token:         fabtokenOutputRaw,
		TokenMetadata: nil,
		Format:        formatFabtoken16,
	}}

	nilgetIdentityProvider := func() upgrade.IdentityProvider {
		return nil
	}
	tests := []struct {
		name                string
		ch                  driver.TokensUpgradeChallenge
		ledgerTokens        []token.LedgerToken
		witness             driver.TokensUpgradeWitness
		wantErr             bool
		errMsg              string
		expected            func() driver.TokensUpgradeProof
		getIdentityProvider func() upgrade.IdentityProvider
	}{
		{
			name:                "challenge size mismatch",
			ch:                  []byte{0, 1, 2},
			wantErr:             true,
			errMsg:              "invalid challenge size, got [3], expected [32]",
			getIdentityProvider: nilgetIdentityProvider,
		},
		{
			name:                "no ledger tokens",
			ch:                  ch,
			wantErr:             true,
			errMsg:              "no ledger tokens provided",
			getIdentityProvider: nilgetIdentityProvider,
		},
		{
			name:                "no witness",
			ch:                  ch,
			witness:             []byte{0, 1, 2},
			ledgerTokens:        invalidTokens,
			wantErr:             true,
			errMsg:              "proof witness not expected",
			getIdentityProvider: nilgetIdentityProvider,
		},
		{
			name:                "unsupported token format",
			ch:                  ch,
			ledgerTokens:        invalidTokens,
			wantErr:             true,
			errMsg:              "failed to process ledgerTokens upgrade request: unsupported token format [token format1]",
			getIdentityProvider: nilgetIdentityProvider,
		},
		{
			name:         "get signer fails",
			ch:           ch,
			ledgerTokens: validTokens,
			wantErr:      true,
			errMsg:       "failed to get identity signer: get signer error",
			getIdentityProvider: func() upgrade.IdentityProvider {
				mip := &mock.IdentityProvider{}
				mip.GetSignerReturns(nil, errors.New("get signer error"))
				return mip
			},
		},
		{
			name:         "get signer fails",
			ch:           ch,
			ledgerTokens: validTokens,
			wantErr:      false,
			expected: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     validTokens,
					Signatures: []upgrade.Signature{[]byte("a signature")},
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			getIdentityProvider: func() upgrade.IdentityProvider {
				signer := &mock2.Signer{}
				signer.SignReturns([]byte("a signature"), nil)
				mip := &mock.IdentityProvider{}
				mip.GetSignerReturns(signer, nil)
				return mip
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := upgrade.NewService(nil, 16, nil, tt.getIdentityProvider())
			assert.NoError(t, err)
			res, err := ts.GenUpgradeProof(tt.ch, tt.ledgerTokens, tt.witness)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected(), res)
			}
		})
	}
}

func TestTokensService_CheckUpgradeProof(t *testing.T) {
	ts, err := upgrade.NewService(nil, 16, nil, nil)
	assert.NoError(t, err)
	ch, err := ts.NewUpgradeChallenge()
	assert.NoError(t, err)

	nilDeserializer := func() upgrade.Deserializer {
		return nil
	}
	nilProof := func() driver.TokensUpgradeProof {
		return nil
	}
	invalidTokens := []token.LedgerToken{{
		ID:            token.ID{TxId: "tx1", Index: 1},
		Token:         []byte("token1"),
		TokenMetadata: []byte("meta1"),
		Format:        token.Format("token format1"),
	}}
	fabtokenOutput := actions.Output{
		Owner:    []byte("owner1"),
		Type:     "token type",
		Quantity: "10",
	}
	fabtokenOutputRaw, err := fabtokenOutput.Serialize()
	assert.NoError(t, err)
	formatFabtoken16, err := v1.SupportedTokenFormat(16)
	assert.NoError(t, err)
	validTokens := []token.LedgerToken{{
		ID:            token.ID{TxId: "tx1", Index: 1},
		Token:         fabtokenOutputRaw,
		TokenMetadata: nil,
		Format:        formatFabtoken16,
	}}

	tests := []struct {
		name            string
		ch              driver.TokensUpgradeChallenge
		ledgerTokens    []token.LedgerToken
		proof           func() driver.TokensUpgradeProof
		wantErr         bool
		errMsg          string
		expected        bool
		wantErrProcess  bool
		processErrMsg   string
		getDeserializer func() upgrade.Deserializer
	}{
		{
			name:            "challenge size mismatch",
			ch:              []byte{0, 1, 2},
			wantErr:         true,
			errMsg:          "invalid challenge size, got [3], expected [32]",
			getDeserializer: nilDeserializer,
			proof:           nilProof,
		},
		{
			name:            "no ledger tokens provided",
			ch:              ch,
			wantErr:         true,
			errMsg:          "no ledger tokens provided",
			getDeserializer: nilDeserializer,
			proof:           nilProof,
		},
		{
			name:            "no proof provided",
			ch:              ch,
			ledgerTokens:    invalidTokens,
			wantErr:         true,
			errMsg:          "no proof provided",
			getDeserializer: nilDeserializer,
			proof:           nilProof,
		},
		{
			name:         "failed to deserialize proof",
			ch:           ch,
			ledgerTokens: invalidTokens,
			proof: func() driver.TokensUpgradeProof {
				return []byte{1, 2}
			},
			wantErr:         true,
			errMsg:          "failed to deserialize proof: invalid character '\\x01' looking for beginning of value",
			getDeserializer: nilDeserializer,
		},
		{
			name:         "proof with invalid token count",
			ch:           ch,
			ledgerTokens: invalidTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr:         true,
			errMsg:          "proof with invalid token count",
			getDeserializer: nilDeserializer,
		},
		{
			name:         "proof with invalid challenge",
			ch:           ch,
			ledgerTokens: invalidTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  nil,
					Tokens:     invalidTokens,
					Signatures: nil,
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr:         true,
			errMsg:          "proof with invalid challenge",
			getDeserializer: nilDeserializer,
		},
		{
			name:         "proof with invalid number of token signatures",
			ch:           ch,
			ledgerTokens: invalidTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     invalidTokens,
					Signatures: nil,
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr:         true,
			errMsg:          "proof with invalid number of token signatures",
			getDeserializer: nilDeserializer,
		},
		{
			name:         "tokens do not match at index [0]",
			ch:           ch,
			ledgerTokens: invalidTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     validTokens,
					Signatures: []upgrade.Signature{[]byte("a signature")},
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr:         true,
			errMsg:          "tokens do not match at index [0]",
			getDeserializer: nilDeserializer,
		},
		{
			name:         "invalid verifier",
			ch:           ch,
			ledgerTokens: validTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     validTokens,
					Signatures: []upgrade.Signature{[]byte("a signature")},
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr: true,
			errMsg:  "failed to get owner verifier: invalid verifier",
			getDeserializer: func() upgrade.Deserializer {
				d := &mock.Deserializer{}
				d.GetOwnerVerifierReturns(nil, errors.New("invalid verifier"))
				return d
			},
		},
		{
			name:         "invalid signature",
			ch:           ch,
			ledgerTokens: validTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     validTokens,
					Signatures: []upgrade.Signature{[]byte("a signature")},
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr: true,
			errMsg:  "failed to verify signature at index [0]: invalid signature",
			getDeserializer: func() upgrade.Deserializer {
				v := &mock2.Verifier{}
				v.VerifyReturns(errors.New("invalid signature"))
				d := &mock.Deserializer{}
				d.GetOwnerVerifierReturns(v, nil)
				return d
			},
		},
		{
			name:         "valid but process fails",
			ch:           ch,
			ledgerTokens: validTokens,
			proof: func() driver.TokensUpgradeProof {
				proof := &upgrade.Proof{
					Challenge:  ch,
					Tokens:     validTokens,
					Signatures: []upgrade.Signature{[]byte("a signature")},
				}
				raw, err := proof.Serialize()
				assert.NoError(t, err)
				return raw
			},
			wantErr: false,
			getDeserializer: func() upgrade.Deserializer {
				v := &mock2.Verifier{}
				v.VerifyReturns(nil)
				d := &mock.Deserializer{}
				d.GetOwnerVerifierReturns(v, nil)
				return d
			},
			expected:       true,
			wantErrProcess: true,
			processErrMsg:  "upgrade of unsupported token format [baff495e067aea1a0a5e6a37d72689316c457251e359a6796329761ca3227648] requested",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts, err := upgrade.NewService(nil, 16, tt.getDeserializer(), nil)
			assert.NoError(t, err)
			proof := tt.proof()
			res, err := ts.CheckUpgradeProof(tt.ch, proof, tt.ledgerTokens)
			if tt.wantErr {
				assert.Error(t, err)
				assert.EqualError(t, err, tt.errMsg)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, res)
			}

			_, err = ts.ProcessTokensUpgradeRequest(&driver.TokenUpgradeRequest{
				Challenge: tt.ch,
				Tokens:    tt.ledgerTokens,
				Proof:     proof,
			})
			if tt.wantErrProcess {
				assert.Error(t, err)
				if len(tt.processErrMsg) != 0 {
					assert.EqualError(t, err, tt.processErrMsg)
				}
			} else {
				if tt.wantErr {
					assert.Error(t, err)
				} else {
					assert.NoError(t, err)
				}
			}
		})
	}
}
