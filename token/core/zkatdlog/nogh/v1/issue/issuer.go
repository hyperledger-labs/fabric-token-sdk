/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

type SigningIdentity interface {
	driver.SigningIdentity
}

// Issuer is the entity that issues tokens
type Issuer struct {
	Signer       SigningIdentity
	PublicParams *v1.PublicParams
	Type         token2.Type
}

// New returns an Issuer as a function of the passed parameters
func (i *Issuer) New(ttype token2.Type, signer common.SigningIdentity, pp *v1.PublicParams) {
	i.Signer = signer
	i.Type = ttype
	i.PublicParams = pp
}

func (i *Issuer) GenerateZKIssue(values []uint64, owners [][]byte) (*Action, []*token.Metadata, error) {
	if i.PublicParams == nil {
		return nil, nil, errors.New("failed to generate ZK Issue: nil public parameters")
	}
	if len(math.Curves) < int(i.PublicParams.Curve)+1 {
		return nil, nil, errors.New("failed to generate ZK Issue: please initialize public parameters with an admissible curve")
	}
	tokens, tw, err := token.GetTokensWithWitness(values, i.Type, i.PublicParams.PedersenGenerators, math.Curves[i.PublicParams.Curve])
	if err != nil {
		return nil, nil, err
	}

	prover, err := NewProver(tw, tokens, i.PublicParams)
	if err != nil {
		return nil, nil, err
	}
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate zero knwoledge proof for issue")
	}

	if i.Signer == nil {
		return nil, nil, errors.New("failed to generate ZK Issue: please initialize signer")
	}
	signerRaw, err := i.Signer.Serialize()
	if err != nil {
		return nil, nil, err
	}

	issue, err := NewAction(signerRaw, tokens, owners, proof)
	if err != nil {
		return nil, nil, err
	}

	inf := make([]*token.Metadata, len(values))
	for j := 0; j < len(inf); j++ {
		inf[j] = &token.Metadata{
			Type:           i.Type,
			Value:          math.Curves[i.PublicParams.Curve].NewZrFromUint64(tw[j].Value),
			BlindingFactor: tw[j].BlindingFactor,
			Issuer:         signerRaw,
		}
	}

	return issue, inf, nil
}

func (i *Issuer) SignTokenActions(raw []byte) ([]byte, error) {
	if i.Signer == nil {
		return nil, errors.New("failed to sign Token Actions: please initialize signer")
	}
	return i.Signer.Sign(raw)
}
