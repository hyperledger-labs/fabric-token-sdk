/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package anonym

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/services/flogging"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/token"
	"github.com/pkg/errors"
)

var logger = flogging.MustGetLogger("token-sdk.zkatdlog.issue")

type Issuer struct {
	Signer       common.SigningIdentity
	PublicParams *crypto.PublicParams
	Type         string
}

func (i *Issuer) New(ttype string, signer common.SigningIdentity, pp *crypto.PublicParams) {
	i.Signer = signer
	i.Type = ttype
	i.PublicParams = pp
}

func (i *Issuer) GenerateZKIssue(values []uint64, owners [][]byte) (*issue2.IssueAction, []*token.TokenInformation, error) {
	tokens, tw, err := token.GetTokensWithWitness(values, i.Type, i.PublicParams.ZKATPedParams)
	if err != nil {
		return nil, nil, err
	}

	prover := issue2.NewProver(tw, tokens, true, i.PublicParams)
	proof, err := prover.Prove()
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate zero knwoledge proof for issue")
	}

	i.Signer, err = CreateSigner(
		tokens[0],
		tw[0].Value,
		tw[0].BlindingFactor,
		bn256.HashModOrder([]byte(i.Type)),
		i.Signer.(*Signer).Witness.Sk,
		i.Signer.(*Signer).Witness.Index,
		i.PublicParams,
	)

	if err != nil {
		return nil, nil, err
	}
	issue, err := issue2.NewIssue(i.Signer.GetPublicVersion(), tokens, owners, proof, true)
	if err != nil {
		return nil, nil, err
	}

	inf := make([]*token.TokenInformation, len(values))
	for j := 0; j < len(inf); j++ {
		inf[j] = &token.TokenInformation{
			Type:           i.Type,
			Value:          tw[j].Value,
			BlindingFactor: tw[j].BlindingFactor,
			Owner:          owners[j],
		}
	}

	return issue, inf, nil
}

func (i *Issuer) SignTokenActions(raw []byte, txID string) ([]byte, error) {
	return i.Signer.Sign(append(raw, []byte(txID)...))
}

func CreateSigner(token *bn256.G1, value, tokenBF, ttype, sk *bn256.Zr, index int, pp *crypto.PublicParams) (*Signer, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, errors.Errorf("failed to get random generator for issuer's signer")
	}

	// compute issuer pseudonym
	tnymbf := bn256.RandModOrder(rand)
	typeNym, err := common.ComputePedersenCommitment([]*bn256.Zr{sk, ttype, tnymbf}, pp.ZKATPedParams)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create issuer signing pseudonym")
	}
	// get issuing policy
	ip := &crypto.IssuingPolicy{}
	err = ip.Deserialize(pp.IssuingPolicy)
	if err != nil {
		return nil, errors.Errorf("failed to get issuer's signer")
	}

	// initialize issuer witness
	auth := NewAuthorization(typeNym, token)
	witness := NewWitness(sk, ttype, value, tnymbf, tokenBF, index)

	logger.Debugf("NewIssuerAuthSigner [%d,%d,%d]", len(ip.Issuers), ip.IssuersNumber, ip.BitLength)

	return NewSigner(witness, ip.Issuers, auth, ip.BitLength, pp.ZKATPedParams), nil
}

func GenerateKeyPair(ttype string, pp *crypto.PublicParams) (*bn256.Zr, *bn256.G1, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, nil, errors.Errorf("failed to generate the secret key of the issuer")
	}

	sk := bn256.RandModOrder(rand)

	pk, err := common.ComputePedersenCommitment([]*bn256.Zr{sk, bn256.HashModOrder([]byte(ttype))}, pp.ZKATPedParams[:2])
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to generate the public key of the issuer")
	}
	return sk, pk, nil
}
