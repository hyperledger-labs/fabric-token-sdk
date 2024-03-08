/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// RangeCorrectness is a collection of range proofs. Each shows that a
// committed value is within the authorized range
type RangeCorrectness struct {
	Proofs []*RangeProof
}

// RangeCorrectnessProver proves that an array of committed values fall within the authorized range
type RangeCorrectnessProver struct {
	// values is the array of values committed in Commitments
	values []uint64
	// blindingFactors is the randomness used to compute Commitments
	blindingFactors []*math.Zr
	// Commitments is an array of hiding Pedersen commitments to values:
	// Commitments[i] = G^[values[i]]H^[blindingFactors[i]]
	Commitments []*math.G1
	// PedersenGenerators are the generators (G, H) used to compute the Commitments
	PedersenGenerators []*math.G1
	// LeftGenerators are the generators that will be used to commit to
	// the bits (b_0,..., b_{BitLength-1}) of value
	LeftGenerators []*math.G1
	// RightGenerators are the generators that will be used to commit to (b_i-1)
	RightGenerators []*math.G1
	// P is a random generator of G1
	P *math.G1
	// Q is a random generator of G1
	Q *math.G1
	// NumberOfRounds correspond to log_2(BitLength). It corresponds to the
	// number of rounds of the reduction protocol
	NumberOfRounds int
	// BitLength is the size of the binary representation of value
	BitLength int
	// Curve is the curve over which the computation is performed
	Curve *math.Curve
}

// NewRangeCorrectnessProver returns a RangeCorrectnessProver as a function of the passed arguments
func NewRangeCorrectnessProver(
	coms []*math.G1,
	values []uint64,
	blindingFactors []*math.Zr,
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	P, Q *math.G1,
	bitLength, rounds int,
	c *math.Curve,
) *RangeCorrectnessProver {
	return &RangeCorrectnessProver{
		Commitments:        coms,
		values:             values,
		blindingFactors:    blindingFactors,
		PedersenGenerators: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		P:                  P,
		Q:                  Q,
		BitLength:          bitLength,
		NumberOfRounds:     rounds,
		Curve:              c,
	}

}

// RangeCorrectnessVerifier verifies that a collection of committed values fall within the authorized range
type RangeCorrectnessVerifier struct {
	// Commitments is an array of hiding Pedersen commitments: Commitments[i] = G^{v_i}H^{r_}
	Commitments []*math.G1
	// PedersenGenerators are the generators (G, H) used to compute the Commitments
	PedersenGenerators []*math.G1
	// LeftGenerators are the generators that will be used to commit to
	// the bits (b_0,..., b_{BitLength-1}) of value
	LeftGenerators []*math.G1
	// RightGenerators are the generators that will be used to commit to (b_i-1)
	RightGenerators []*math.G1
	// P is a random generator of G1
	P *math.G1
	// Q is a random generator of G1
	Q *math.G1
	// NumberOfRounds correspond to log_2(BitLength). It corresponds to the
	// number of rounds of the reduction protocol
	NumberOfRounds int
	// BitLength is the size of the binary representation of value
	BitLength int
	// Curve is the curve over which the computation is performed
	Curve *math.Curve
}

// NewRangeCorrectnessVerifier returns a RangeCorrectnessVerifier as a function of the passed arguments
func NewRangeCorrectnessVerifier(
	pedersenParameters, leftGenerators, rightGenerators []*math.G1,
	P, Q *math.G1,
	bitLength, rounds int,
	curve *math.Curve,
) *RangeCorrectnessVerifier {
	return &RangeCorrectnessVerifier{
		PedersenGenerators: pedersenParameters,
		LeftGenerators:     leftGenerators,
		RightGenerators:    rightGenerators,
		P:                  P,
		Q:                  Q,
		BitLength:          bitLength,
		NumberOfRounds:     rounds,
		Curve:              curve,
	}

}

// Prove allows a RangeCorrectnessProver to produce a RangeCorrectness proof.
// Within Prove, RangeProver.Prove() is called for each committed value
func (p *RangeCorrectnessProver) Prove() (*RangeCorrectness, error) {
	rc := &RangeCorrectness{}
	for i := 0; i < len(p.Commitments); i++ {
		rp := NewRangeProver(
			p.Commitments[i],
			p.values[i],
			p.PedersenGenerators,
			p.blindingFactors[i],
			p.LeftGenerators,
			p.RightGenerators,
			p.P,
			p.Q,
			p.NumberOfRounds,
			p.BitLength,
			p.Curve,
		)
		proof, err := rp.Prove()
		if err != nil {
			return nil, err
		}
		rc.Proofs = append(rc.Proofs, proof)
	}

	return rc, nil
}

// Verify enables a RangeCorrectnessVerifier to check the validity of a RangeCorrectness
// proof against an array of committed values. Within Verify, RangeVerifier.Verify is
// called for each committed value
func (v *RangeCorrectnessVerifier) Verify(rc *RangeCorrectness) error {
	if len(rc.Proofs) != len(v.Commitments) {
		return errors.New("invalid range proof")
	}
	for i := 0; i < len(rc.Proofs); i++ {
		if rc.Proofs[i] == nil {
			return errors.Errorf("invalid range proof: nil proof at index %d", i)
		}
		rv := NewRangeVerifier(
			v.Commitments[i],
			v.PedersenGenerators,
			v.LeftGenerators,
			v.RightGenerators,
			v.P,
			v.Q,
			v.NumberOfRounds,
			v.BitLength,
			v.Curve,
		)
		err := rv.Verify(rc.Proofs[i])
		if err != nil {
			return errors.Wrapf(err, "invalid range proof at index %d", i)
		}
	}
	return nil
}
