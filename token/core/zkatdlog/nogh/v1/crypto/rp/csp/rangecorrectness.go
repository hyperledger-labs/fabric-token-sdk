/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
)

// CSPRangeCorrectness contains a set of range proofs for multiple commitments.
type CSPRangeCorrectness struct {
	// Proofs is a slice of range proofs.
	Proofs []*CspRangeProof
}

// Serialize marshals the CSPRangeCorrectness into a byte slice.
func (r *CSPRangeCorrectness) Serialize() ([]byte, error) {
	proofs, err := asn1.NewArray(r.Proofs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal proofs")
	}

	return asn1.Marshal(proofs)
}

// Deserialize unmarshals a byte slice into the CSPRangeCorrectness.
func (r *CSPRangeCorrectness) Deserialize(raw []byte) error {
	proofs, err := asn1.NewArrayWithNew[*CspRangeProof](func() *CspRangeProof {
		return &CspRangeProof{}
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

// Validate checks that all range proofs in the set are valid for the given curve.
func (r *CSPRangeCorrectness) Validate(curve math.CurveID) error {
	for i, proof := range r.Proofs {
		if proof == nil {
			return errors.Errorf("invalid range proof: nil proof at index %d", i)
		}
		err := proof.Validate(curve)
		if err != nil {
			return errors.Wrapf(err, "invalid range proof at index %d", i)
		}
	}

	return nil
}

// CSPRangeCorrectnessProver manages the generation of a set of range proofs.
type CSPRangeCorrectnessProver struct {
	// Commitments is the set of Pedersen commitments for which range proofs are generated.
	Commitments []*math.G1
	// Values is the set of underlying values.
	Values []uint64
	// BlindingFactors is the set of blinding factors for the commitments.
	BlindingFactors []*math.Zr
	// PedersenParameters are the generators (G, H).
	PedersenParameters []*math.G1
	// LeftGenerators are the generators for the left vector.
	LeftGenerators []*math.G1
	// RightGenerators are the generators for the right vector.
	RightGenerators []*math.G1
	// BitLength is the maximum bit length of the values.
	BitLength uint64
	// Curve is the mathematical curve.
	Curve *math.Curve
}

// NewCSPRangeCorrectnessProver returns a new CSPRangeCorrectnessProver instance.
func NewCSPRangeCorrectnessProver(
	coms []*math.G1,
	values []uint64,
	blindingFactors []*math.Zr,
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	bitLength uint64,
	c *math.Curve,
) *CSPRangeCorrectnessProver {
	return &CSPRangeCorrectnessProver{
		Commitments:        coms,
		Values:             values,
		BlindingFactors:    blindingFactors,
		PedersenParameters: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		BitLength:          bitLength,
		Curve:              c,
	}
}

// Prove generates a set of range proofs.
func (p *CSPRangeCorrectnessProver) Prove() (*CSPRangeCorrectness, error) {
	rc := &CSPRangeCorrectness{}
	rc.Proofs = make([]*CspRangeProof, len(p.Commitments))
	for i := range len(p.Commitments) {
		bp := NewCspRangeProver(
			p.Commitments[i],
			math2.NewCachedZrFromInt(p.Curve, p.Values[i]),
			p.BlindingFactors[i],
			p.PedersenParameters,
			p.LeftGenerators,
			p.RightGenerators,
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

// CSPRangeCorrectnessVerifier manages the verification of a set of range proofs.
type CSPRangeCorrectnessVerifier struct {
	// Commitments is the set of Pedersen commitments being verified.
	Commitments []*math.G1
	// PedersenParameters are the generators (G, H).
	PedersenParameters []*math.G1
	// LeftGenerators are the generators for the left vector.
	LeftGenerators []*math.G1
	// RightGenerators are the generators for the right vector.
	RightGenerators []*math.G1
	// BitLength is the maximum bit length of the values.
	BitLength uint64
	// Curve is the mathematical curve.
	Curve *math.Curve
}

// NewCSPRangeCorrectnessVerifier returns a new CSPRangeCorrectnessVerifier instance.
func NewCSPRangeCorrectnessVerifier(
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	bitLength uint64,
	curve *math.Curve,
) *CSPRangeCorrectnessVerifier {
	return &CSPRangeCorrectnessVerifier{
		PedersenParameters: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		BitLength:          bitLength,
		Curve:              curve,
	}
}

// Verify checks if the provided set of range proofs is valid.
func (v *CSPRangeCorrectnessVerifier) Verify(rc *CSPRangeCorrectness) error {
	if len(rc.Proofs) != len(v.Commitments) {
		return errors.New("invalid range proof")
	}
	for i := range len(rc.Proofs) {
		if rc.Proofs[i] == nil {
			return errors.Errorf("invalid range proof: nil proof at index %d", i)
		}
		bv := newCspRangeVerifier(
			v.PedersenParameters,
			v.LeftGenerators,
			v.RightGenerators,
			v.Commitments[i],
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
