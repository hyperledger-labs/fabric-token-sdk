/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
	"github.com/pkg/errors"
)

type RangeCorrectness struct {
	Proofs []*RangeProof
}

func (r *RangeCorrectness) Serialize() ([]byte, error) {
	proofs, err := asn1.NewArray(r.Proofs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal proofs")
	}
	return asn1.Marshal(proofs)
}

func (r *RangeCorrectness) Deserialize(raw []byte) error {
	proofs, err := asn1.NewArrayWithNew[*RangeProof](func() *RangeProof {
		return &RangeProof{}
	})
	if err != nil {
		return errors.Wrap(err, "failed to prepare proofs for unmarshalling")
	}
	err = asn1.Unmarshal(raw, proofs)
	if err != nil {
		return errors.Wrap(err, "failed to unmarshal proofs")
	}
	r.Proofs = proofs.Values
	return nil
}

type RangeCorrectnessProver struct {
	Commitments        []*math.G1
	Values             []uint64
	BlindingFactors    []*math.Zr
	PedersenParameters []*math.G1
	LeftGenerators     []*math.G1
	RightGenerators    []*math.G1
	BitLength          uint64
	NumberOfRounds     uint64
	P                  *math.G1
	Q                  *math.G1
	Curve              *math.Curve
}

func NewRangeCorrectnessProver(
	coms []*math.G1,
	values []uint64,
	blindingFactors []*math.Zr,
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	P, Q *math.G1,
	bitLength, rounds uint64,
	c *math.Curve,
) *RangeCorrectnessProver {
	return &RangeCorrectnessProver{
		Commitments:        coms,
		Values:             values,
		BlindingFactors:    blindingFactors,
		PedersenParameters: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		P:                  P,
		Q:                  Q,
		BitLength:          bitLength,
		NumberOfRounds:     rounds,
		Curve:              c,
	}
}

func (p *RangeCorrectnessProver) Prove() (*RangeCorrectness, error) {
	rc := &RangeCorrectness{}
	rc.Proofs = make([]*RangeProof, len(p.Commitments))
	for i := 0; i < len(p.Commitments); i++ {
		bp := NewRangeProver(
			p.Commitments[i],
			p.Values[i],
			p.PedersenParameters,
			p.BlindingFactors[i],
			p.LeftGenerators,
			p.RightGenerators,
			p.P,
			p.Q,
			p.NumberOfRounds,
			p.BitLength,
			p.Curve,
		)
		proof, err := bp.Prove()
		if err != nil {
			return nil, err
		}
		rc.Proofs[i] = proof
	}
	return rc, nil
}

type RangeCorrectnessVerifier struct {
	Commitments        []*math.G1
	PedersenParameters []*math.G1
	LeftGenerators     []*math.G1
	RightGenerators    []*math.G1
	BitLength          uint64
	NumberOfRounds     uint64
	P                  *math.G1
	Q                  *math.G1
	Curve              *math.Curve
}

func NewRangeCorrectnessVerifier(
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	P, Q *math.G1,
	bitLength, rounds uint64,
	curve *math.Curve,
) *RangeCorrectnessVerifier {
	return &RangeCorrectnessVerifier{
		PedersenParameters: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		P:                  P,
		Q:                  Q,
		BitLength:          bitLength,
		NumberOfRounds:     rounds,
		Curve:              curve,
	}

}

func (v *RangeCorrectnessVerifier) Verify(rc *RangeCorrectness) error {
	if len(rc.Proofs) != len(v.Commitments) {
		return errors.New("invalid range proof")
	}
	for i := 0; i < len(rc.Proofs); i++ {
		if rc.Proofs[i] == nil {
			return errors.Errorf("invalid range proof: nil proof at index %d", i)
		}
		bv := NewRangeVerifier(
			v.Commitments[i],
			v.PedersenParameters,
			v.LeftGenerators,
			v.RightGenerators,
			v.P,
			v.Q,
			v.NumberOfRounds,
			v.BitLength,
			v.Curve,
		)
		err := bv.Verify(rc.Proofs[i])
		if err != nil {
			return errors.Wrapf(err, "invalid range proof at index %d", i)
		}
	}
	return nil
}
