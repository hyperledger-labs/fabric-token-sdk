/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/token"
)

// Proof proves that an IssueAction is valid
type Proof struct {
	// SameType is the proof that a bridge commitment is of type G_0^typeH^r
	SameType *SameType
	// RangeCorrectness is the proof that issued tokens have value in the authorized range
	RangeCorrectness *rp.RangeCorrectness
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return asn1.Marshal[asn1.Serializer](p.SameType, p.RangeCorrectness)
}

// Deserialize un-marshals Proof
func (p *Proof) Deserialize(bytes []byte) error {
	p.SameType = &SameType{}
	p.RangeCorrectness = &rp.RangeCorrectness{}
	return asn1.Unmarshal[asn1.Serializer](bytes, p.SameType, p.RangeCorrectness)
}

// Prover produces a proof of validity of an IssueAction
type Prover struct {
	// SameType encodes the SameType Prover
	SameType *SameTypeProver
	// RangeCorrectness encodes the range proof Prover
	RangeCorrectness *rp.RangeCorrectnessProver
}

func NewProver(tw []*token.TokenDataWitness, tokens []*math.G1, pp *v1.PublicParams) (*Prover, error) {
	c := math.Curves[pp.Curve]
	p := &Prover{}
	tokenType := c.HashToZr([]byte(tw[0].Type))
	commitmentToType := pp.PedersenGenerators[0].Mul(tokenType)

	rand, err := c.Rand()
	if err != nil {
		return nil, errors.Wrapf(err, "cannot get issue prover")
	}
	typeBF := c.NewRandomZr(rand)
	commitmentToType.Add(pp.PedersenGenerators[2].Mul(typeBF))
	p.SameType = NewSameTypeProver(tw[0].Type, typeBF, commitmentToType, pp.PedersenGenerators, c)

	values := make([]uint64, len(tw))
	blindingFactors := make([]*math.Zr, len(tw))
	for i := 0; i < len(tw); i++ {
		if tw[i] == nil || tw[i].BlindingFactor == nil {
			return nil, errors.New("invalid token witness")
		}
		// tw[i] = tw[i].Clone()
		values[i] = tw[i].Value
		blindingFactors[i] = c.ModSub(tw[i].BlindingFactor, p.SameType.blindingFactor, c.GroupOrder)
	}
	coms := make([]*math.G1, len(tokens))
	for i := 0; i < len(tokens); i++ {
		coms[i] = tokens[i].Copy()
		coms[i].Sub(commitmentToType)
	}
	// range prover takes commitments tokens[i]/commitmentToType
	p.RangeCorrectness = rp.NewRangeCorrectnessProver(
		coms,
		values,
		blindingFactors,
		pp.PedersenGenerators[1:],
		pp.RangeProofParams.LeftGenerators,
		pp.RangeProofParams.RightGenerators,
		pp.RangeProofParams.P,
		pp.RangeProofParams.Q,
		pp.RangeProofParams.BitLength,
		pp.RangeProofParams.NumberOfRounds,
		math.Curves[pp.Curve],
	)

	return p, nil
}

// Prove produces a Proof for an IssueAction
func (p *Prover) Prove() ([]byte, error) {
	// TypeAndSum proof
	st, err := p.SameType.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate issue proof")
	}

	// RangeCorrectness proof
	rc, err := p.RangeCorrectness.Prove()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to generate range proof for issue")
	}

	proof := &Proof{
		SameType:         st,
		RangeCorrectness: rc,
	}
	return proof.Serialize()
}
