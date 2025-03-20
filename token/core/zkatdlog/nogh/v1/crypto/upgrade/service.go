/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package upgrade

import (
	"bytes"
	"crypto/rand"
	errors2 "errors"
	"slices"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/json"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/logging"
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

type Service struct {
	Logger                          logging.Logger
	PublicParametersManager         common.PublicParametersManager[*crypto.PublicParams]
	UpgradeSupportedTokenFormatList []token.Format
	Deserializer                    driver.Deserializer
	IdentityProvider                driver.IdentityProvider
}

func NewService(
	logger logging.Logger,
	publicParametersManager common.PublicParametersManager[*crypto.PublicParams],
	deserializer driver.Deserializer,
	identityProvider driver.IdentityProvider,
) (*Service, error) {
	// compute supported tokens
	pp := publicParametersManager.PublicParams()
	maxPrecision := pp.RangeProofParams.BitLength

	var upgradeSupportedTokenFormatList []token.Format
	for _, precision := range []uint64{16, 32, 64} {
		format, err := v1.SupportedTokenFormat(precision)
		if err != nil {
			return nil, errors.Wrapf(err, "failed computing fabtoken token format with precision [%d]", precision)
		}
		if precision > maxPrecision {
			upgradeSupportedTokenFormatList = append(upgradeSupportedTokenFormatList, format)
		}
	}

	return &Service{
		Logger:                          logger,
		PublicParametersManager:         publicParametersManager,
		UpgradeSupportedTokenFormatList: upgradeSupportedTokenFormatList,
		Deserializer:                    deserializer,
		IdentityProvider:                identityProvider,
	}, nil
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
func (s *Service) GenUpgradeProof(ch driver.TokensUpgradeChallenge, ledgerTokens []token.LedgerToken, witness driver.TokensUpgradeWitness) (driver.TokensUpgradeProof, error) {
	if len(ch) != ChallengeSize {
		return nil, errors.Errorf("invalid challenge size, got [%d], expected [%d]", len(ch), ChallengeSize)
	}
	if len(ledgerTokens) == 0 {
		return nil, errors.Errorf("no ledgerTokens provided")
	}

	proof := &Proof{
		Challenge:  ch,
		Tokens:     ledgerTokens,
		Signatures: make([]Signature, 0, len(ledgerTokens)),
	}
	digest, err := proof.SHA256Digest()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get sha256 digest")
	}

	tokens, err := s.ProcessTokens(ledgerTokens)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to process ledgerTokens upgrade request")
	}
	for _, token := range tokens {
		// get a signer for each token
		signer, err := s.IdentityProvider.GetSigner(token.Owner)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get identity signer")
		}
		sigma, err := signer.Sign(digest)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get signature")
		}
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

func (s *Service) CheckUpgradeProof(ch driver.TokensUpgradeChallenge, proofRaw driver.TokensUpgradeProof, ledgerTokens []token.LedgerToken) (bool, error) {
	_, v, err := s.checkUpgradeProof(ch, proofRaw, ledgerTokens)
	return v, err
}

func (s *Service) ProcessTokensUpgradeRequest(utp *driver.TokenUpgradeRequest) ([]token.Token, error) {
	if utp == nil {
		return nil, errors.New("nil token upgrade request")
	}

	// check that each token doesn't have a supported format
	for _, tok := range utp.Tokens {
		if !slices.Contains(s.UpgradeSupportedTokenFormatList, tok.Format) {
			return nil, errors.Errorf("upgrade of unsupported token format [%s] requested", tok.Format)
		}
	}

	// check the upgrade proof
	tokens, ok, err := s.checkUpgradeProof(utp.Challenge, utp.Proof, utp.Tokens)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check upgrade proof")
	}
	if !ok {
		return nil, errors.New("invalid upgrade proof")
	}

	// for each token, extract type and value
	return tokens, nil
}

func (s *Service) ProcessTokens(ledgerTokens []token.LedgerToken) ([]token.Token, error) {
	// for each token, extract type and value
	tokens := make([]token.Token, len(ledgerTokens))
	maxPrecision := s.PublicParametersManager.PublicParams().RangeProofParams.BitLength
	for i, tok := range ledgerTokens {
		precision, ok := token2.Precisions[tok.Format]
		if !ok {
			return nil, errors.Errorf("unsupported token format [%s]", tok.Format)
		}
		fabToken, _, err := token2.ParseFabtokenToken(tok.Token, precision, maxPrecision)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check unspent tokens")
		}
		tokens[i] = token.Token{
			Owner:    fabToken.Owner,
			Type:     fabToken.Type,
			Quantity: fabToken.Quantity,
		}
	}
	return tokens, nil
}

func (s *Service) checkUpgradeProof(ch driver.TokensUpgradeChallenge, proofRaw driver.TokensUpgradeProof, ledgerTokens []token.LedgerToken) ([]token.Token, bool, error) {
	if len(ch) != ChallengeSize {
		return nil, false, errors.Errorf("invalid challenge size, got [%d], expected [%d]", len(ch), ChallengeSize)
	}
	if len(ledgerTokens) == 0 {
		return nil, false, errors.Errorf("no ledgerTokens provided")
	}
	if len(proofRaw) == 0 {
		return nil, false, errors.Errorf("no proof provided")
	}

	// unmarshal proof
	proof := &Proof{}
	if err := proof.Deserialize(proofRaw); err != nil {
		return nil, false, errors.Wrapf(err, "failed to deserialize proof")
	}
	if len(proof.Tokens) != len(ledgerTokens) {
		return nil, false, errors.Errorf("invalid token count")
	}
	if !bytes.Equal(proof.Challenge, ch) {
		return nil, false, errors.Errorf("invalid challenge")
	}
	if len(proof.Signatures) != len(ledgerTokens) {
		return nil, false, errors.Errorf("invalid number of token signatures")
	}
	proof.Challenge = ch
	proof.Tokens = ledgerTokens

	digest, err := proof.SHA256Digest()
	if err != nil {
		return nil, false, errors.Wrapf(err, "failed to get sha256 digest")
	}

	// verify signatures
	tokens, err := s.ProcessTokens(proof.Tokens)
	if err != nil {
		return nil, false, errors.Wrap(err, "failed to process ledgerTokens")
	}
	for i, token := range tokens {
		verifier, err := s.Deserializer.GetOwnerVerifier(token.Owner)
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to get owner verifier")
		}
		err = verifier.Verify(digest, proof.Signatures[i])
		if err != nil {
			return nil, false, errors.Wrapf(err, "failed to verify signature at index [%d]", i)
		}
	}

	// all good
	return tokens, true, nil
}
