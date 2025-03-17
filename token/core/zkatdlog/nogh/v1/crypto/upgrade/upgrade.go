/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"bytes"
	"crypto/rand"
	errors2 "errors"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/utils"
	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
)

const (
	ChallengeSize = 32
)

type (
	Signature = []byte
)

type Proof struct {
	Challenge  driver.TokensUpgradeChallenge
	Tokens     []token.LedgerToken
	Signatures []Signature
}

func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Proof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

func (p *Proof) SHA256Digest() ([]byte, error) {
	h := utils.NewSHA256Hasher()
	err := h.AddBytes(p.Challenge)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to write challenge to hash")
	}
	for _, token := range p.Tokens {
		if err := errors2.Join(
			h.AddString(token.ID.TxId),
			h.AddUInt64(token.ID.Index),
			h.AddBytes(token.Token),
			h.AddBytes(token.TokenMetadata),
			h.AddString(string(token.Format)),
		); err != nil {
			return nil, errors.Wrapf(err, "failed to write token to hash")
		}
	}
	return h.Digest(), nil
}

func (p *Proof) AddSignature(sigma Signature) {
	p.Signatures = append(p.Signatures, sigma)
}

type Service struct{}

func NewService() *Service {
	return &Service{}
}

func (s *Service) NewUpgradeChallenge() (driver.TokensUpgradeChallenge, error) {
	// generate a 32 bytes secure random slice
	key := make([]byte, ChallengeSize)
	_, err := rand.Read(key)
	if err != nil {
		return nil, errors.Wrap(err, "error getting random bytes")
	}
	// rand.Read guarantees that len(key) == ChallengeSize, let's check it anyway
	if len(key) != ChallengeSize {
		return nil, errors.Errorf("invalid key size, got only [%d], expected [%d]", len(key), ChallengeSize)
	}
	return key, nil
}

// GenUpgradeProof does the following: For each token in input, it signs the concatenation of the challenge and the tokens to be upgraded.
// These signatures are then added to the proof
func (s *Service) GenUpgradeProof(ch driver.TokensUpgradeChallenge, tokens []token.LedgerToken, witness driver.TokensUpgradeWitness) (driver.TokensUpgradeProof, error) {
	if len(ch) != ChallengeSize {
		return nil, errors.Errorf("invalid challenge size, got [%d], expected [%d]", len(ch), ChallengeSize)
	}
	if len(tokens) == 0 {
		return nil, errors.Errorf("no tokens provided")
	}

	proof := &Proof{
		Challenge:  ch,
		Tokens:     tokens,
		Signatures: make([]Signature, 0, len(tokens)),
	}
	digest, err := proof.SHA256Digest()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sha256 digest")
	}

	for _, token := range tokens {
		// get a signer for each token

		// sign the digest
		var sigma Signature

		// add the signature to the proof
		proof.AddSignature(sigma)
	}

	// marshal proof
	raw, err := proof.Serialize()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize proof")
	}
	return raw, nil
}

func (s *Service) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proofRaw driver.TokensUpgradeProof, tokens []token.LedgerToken) (bool, error) {
	if len(ch) != ChallengeSize {
		return false, errors.Errorf("invalid challenge size, got [%d], expected [%d]", len(ch), ChallengeSize)
	}
	if len(tokens) == 0 {
		return false, errors.Errorf("no tokens provided")
	}
	if len(proofRaw) == 0 {
		return false, errors.Errorf("no proof provided")
	}

	// unmarshal proof
	proof := &Proof{}
	if err := proof.Deserialize(proofRaw); err != nil {
		return false, errors.Wrapf(err, "failed to deserialize proof")
	}
	if len(proof.Tokens) != len(tokens) {
		return false, errors.Errorf("invalid token count")
	}
	if !bytes.Equal(proof.Challenge, ch) {
		return false, errors.Errorf("invalid challenge")
	}
	proof.Challenge = ch
	proof.Tokens = tokens

	digest, err := proof.SHA256Digest()
	if err != nil {
		return false, errors.Wrapf(err, "failed to get sha256 digest")
	}

	// verify signatures
	for i, token := range proof.Tokens {

	}

	// all good
	return true, nil
}
