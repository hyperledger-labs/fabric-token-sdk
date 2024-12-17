/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

//go:generate counterfeiter -o mock/signing_identity.go -fake-name SigningIdentity . SigningIdentity

type SigningIdentity interface {
	driver.SigningIdentity
}

// Issuer is the entity that issues tokens
type Issuer struct {
	Signer       SigningIdentity
	PublicParams *crypto.PublicParams
	Type         string
}

// New returns an Issuer as a function of the passed parameters
func (i *Issuer) New(ttype string, signer common.SigningIdentity, pp *crypto.PublicParams) {
	i.Signer = signer
	i.Type = ttype
	i.PublicParams = pp
}

func (i *Issuer) GenerateZKIssue(values []uint64, owners [][]byte) (*IssueAction, []*token.Metadata, error) {
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

	issue, err := NewIssue(signerRaw, tokens, owners, proof)
	if err != nil {
		return nil, nil, err
	}

	inf := make([]*token.Metadata, len(values))
	for j := 0; j < len(inf); j++ {
		inf[j] = &token.Metadata{
			Type:           i.Type,
			Value:          math.Curves[i.PublicParams.Curve].NewZrFromUint64(tw[j].Value),
			BlindingFactor: tw[j].BlindingFactor,
			Owner:          owners[j],
			Issuer:         signerRaw,
		}
	}

	return issue, inf, nil
}

func (i *Issuer) SignTokenActions(raw []byte, txID string) ([]byte, error) {
	if i.Signer == nil {
		return nil, errors.New("failed to sign Token Actions: please initialize signer")
	}
	return i.Signer.Sign(append(raw, []byte(txID)...))
}
