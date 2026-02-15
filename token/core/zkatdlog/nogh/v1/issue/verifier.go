/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
)

// Verifier coordinates the verification of zero-knowledge proofs for an issue action.
type Verifier struct {
	// SameType is the verifier for the same-type property.
	SameType *SameTypeVerifier
	// RangeCorrectness is the verifier for the range correctness property.
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

// NewVerifier instantiates a Verifier for the given token commitments and public parameters.
func NewVerifier(tokens []*math.G1, pp *v1.PublicParams) *Verifier {
	v := &Verifier{}
	v.SameType = NewSameTypeVerifier(tokens, pp.PedersenGenerators, math.Curves[pp.Curve])
	v.RangeCorrectness = rp.NewRangeCorrectnessVerifier(pp.PedersenGenerators[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])

	return v
}

// Verify checks the validity of the zero-knowledge proof for an issue action.
// It verifies both the same-type property and the range correctness of the issued tokens.
func (v *Verifier) Verify(proof []byte) error {
	tp := &Proof{}
	// Unmarshal the proof.
	err := tp.Deserialize(proof)
	if err != nil {
		return err
	}
	// Verify the same-type proof.
	err = v.SameType.Verify(tp.SameType)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	// Verify the range correctness proof.
	// The range proof is performed on tokens[i] / commitmentToType to show they commit to a positive value.
	commitmentToType := tp.SameType.CommitmentToType.Copy()
	coms := make([]*math.G1, len(v.SameType.Tokens))
	for i := range len(v.SameType.Tokens) {
		coms[i] = v.SameType.Tokens[i].Copy()
		coms[i].Sub(commitmentToType)
	}
	v.RangeCorrectness.Commitments = coms
	err = v.RangeCorrectness.Verify(tp.RangeCorrectness)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}

	return nil
}
