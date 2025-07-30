/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/pkg/errors"
)

// Verifier checks if Proof is valid
type Verifier struct {
	// SameType encodes the SameType Verifier
	SameType *SameTypeVerifier
	// RangeCorrectness encodes the range proof verifier
	RangeCorrectness *rp.RangeCorrectnessVerifier
}

func NewVerifier(tokens []*math.G1, pp *v1.PublicParams) *Verifier {
	v := &Verifier{}
	v.SameType = NewSameTypeVerifier(tokens, pp.PedersenGenerators, math.Curves[pp.Curve])
	v.RangeCorrectness = rp.NewRangeCorrectnessVerifier(pp.PedersenGenerators[1:], pp.RangeProofParams.LeftGenerators, pp.RangeProofParams.RightGenerators, pp.RangeProofParams.P, pp.RangeProofParams.Q, pp.RangeProofParams.BitLength, pp.RangeProofParams.NumberOfRounds, math.Curves[pp.Curve])
	return v
}

// Verify returns an error if Proof of an IssueAction is invalid
func (v *Verifier) Verify(proof []byte) error {
	tp := &Proof{}
	// unmarshal proof
	err := tp.Deserialize(proof)
	if err != nil {
		return err
	}
	// verify TypeAndSum proof
	err = v.SameType.Verify(tp.SameType)
	if err != nil {
		return errors.Wrapf(err, "invalid issue proof")
	}
	// verify RangeCorrectness proof
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
