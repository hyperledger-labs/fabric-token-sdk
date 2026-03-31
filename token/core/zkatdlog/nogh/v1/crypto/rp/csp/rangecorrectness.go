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

// RangeCorrectness contains a set of range proofs for multiple commitments.
type RangeCorrectness struct {
	// Proofs is a slice of range proofs.
	Proofs []*RangeProof
}

// Serialize marshals the RangeCorrectness into a byte slice.
func (r *RangeCorrectness) Serialize() ([]byte, error) {
	proofs, err := asn1.NewArray(r.Proofs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal proofs")
	}

	return asn1.Marshal(proofs)
}

// Deserialize unmarshals a byte slice into the RangeCorrectness.
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

// Validate checks that all range proofs in the set are valid for the given curve.
func (r *RangeCorrectness) Validate(curve math.CurveID) error {
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

// RangeCorrectnessProver manages the generation of a set of range proofs.
type RangeCorrectnessProver struct {
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
	Curve            *math.Curve
	TranscriptHeader []byte
}

// NewRangeCorrectnessProver returns a new RangeCorrectnessProver instance.
func NewRangeCorrectnessProver(
	coms []*math.G1,
	values []uint64,
	blindingFactors []*math.Zr,
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	bitLength uint64,
	c *math.Curve,
) *RangeCorrectnessProver {
	return &RangeCorrectnessProver{
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
func (p *RangeCorrectnessProver) Prove() (*RangeCorrectness, error) {
	if len(p.TranscriptHeader) == 0 {
		return nil, errors.New("transcript header is empty")
	}
	rc := &RangeCorrectness{}
	rc.Proofs = make([]*RangeProof, len(p.Commitments))
	for i := range len(p.Commitments) {
		bp := NewRangeProver(
			p.Commitments[i],
			math2.NewCachedZrFromInt(p.Curve, p.Values[i]),
			p.BlindingFactors[i],
			p.PedersenParameters,
			p.LeftGenerators,
			p.RightGenerators,
			p.BitLength,
			p.Curve,
		).WithTranscriptHeader(p.TranscriptHeader)
		proof, err := bp.Prove()
		if err != nil {
			return nil, err
		}
		rc.Proofs[i] = proof
	}

	return rc, nil
}

func (p *RangeCorrectnessProver) WithTranscriptHeader(header []byte) *RangeCorrectnessProver {
	p.TranscriptHeader = header

	return p
}

// RangeCorrectnessVerifier manages the verification of a set of range proofs.
type RangeCorrectnessVerifier struct {
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

	TranscriptHeader []byte
}

// NewRangeCorrectnessVerifier returns a new RangeCorrectnessVerifier instance.
func NewRangeCorrectnessVerifier(
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	bitLength uint64,
	curve *math.Curve,
) *RangeCorrectnessVerifier {
	return &RangeCorrectnessVerifier{
		PedersenParameters: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		BitLength:          bitLength,
		Curve:              curve,
	}
}

// Verify checks if the provided set of range proofs is valid.
func (v *RangeCorrectnessVerifier) Verify(rc *RangeCorrectness) error {
	if len(rc.Proofs) != len(v.Commitments) {
		return errors.New("invalid range proof")
	}
	if len(v.TranscriptHeader) == 0 {
		return errors.New("transcript header is empty")
	}
	for i := range len(rc.Proofs) {
		if rc.Proofs[i] == nil {
			return errors.Errorf("invalid range proof: nil proof at index %d", i)
		}
		bv := NewRangeVerifier(
			v.PedersenParameters,
			v.LeftGenerators,
			v.RightGenerators,
			v.Commitments[i],
			v.BitLength,
			v.Curve,
		).WithTranscriptHeader(v.TranscriptHeader)
		err := bv.Verify(rc.Proofs[i])
		if err != nil {
			return errors.Wrapf(err, "invalid range proof at index %d", i)
		}
	}

	return nil
}

func (v *RangeCorrectnessVerifier) WithTranscriptHeader(header []byte) *RangeCorrectnessVerifier {
	v.TranscriptHeader = header

	return v
}
