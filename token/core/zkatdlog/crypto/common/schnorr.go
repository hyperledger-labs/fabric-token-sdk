/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// SchnorrProof carries a ZKP for statement (w_1, ..., w_n): Com = \Prod_{j=1}^n P_i^w_i
type SchnorrProof struct {
	Statement *math.G1
	Proof     []*math.Zr
	Challenge *math.Zr
}

// SchnorrProver produces a Schnorr proof
type SchnorrProver struct {
	*SchnorrVerifier
	Witness    []*math.Zr
	Randomness []*math.Zr
	Challenge  *math.Zr
}

// SchnorrVerifier verifies a SchnorrProof
type SchnorrVerifier struct {
	PedParams []*math.G1
	Curve     *math.Curve
}

// Prove produces an array of Zr elements that match the passed
// challenge, witnesses and randomness.
func (p *SchnorrProver) Prove() ([]*math.Zr, error) {
	if p.Curve == nil || p.Curve.GroupOrder == nil {
		return nil, errors.New("cannot compute Schnorr proof: please initialize curve properly")
	}
	if len(p.Witness) != len(p.Randomness) {
		return nil, errors.New("cannot compute Schnorr proof: please initialize witness and randomness correctly")
	}
	if p.Challenge == nil {
		return nil, errors.New("cannot compute Schnorr proof: please initialize challenge")
	}
	proof := make([]*math.Zr, len(p.Witness))
	// p_i = r_i + c*w_i mod q
	for i := 0; i < len(proof); i++ {
		if p.Witness[i] == nil || p.Randomness == nil {
			return nil, errors.New("cannot compute Schnorr proof: please initialize nil elements")
		}
		proof[i] = p.Curve.ModMul(p.Challenge, p.Witness[i], p.Curve.GroupOrder)
		proof[i] = p.Curve.ModAdd(proof[i], p.Randomness[i], p.Curve.GroupOrder)
	}
	return proof, nil
}

// ComputePedersenCommitment returns the commitment to opening relative to base in passed curve.
func ComputePedersenCommitment(opening []*math.Zr, base []*math.G1, c *math.Curve) (*math.G1, error) {
	if len(opening) != len(base) {
		return nil, errors.Errorf("can't compute Pedersen commitment [%d]!=[%d]", len(opening), len(base))
	}
	if c == nil {
		return nil, errors.Errorf("can't compute Pedersen commitment: please initialize curve")
	}
	com := c.NewG1()
	for i := 0; i < len(base); i++ {
		if base[i] == nil || opening[i] == nil {
			return nil, errors.Errorf("can't compute Pedersen commitment: nil EC points")
		}
		com.Add(base[i].Mul(opening[i]))
	}
	return com, nil
}

// RecomputeCommitment is called by the verifier.
// It takes a SchnorrProof and returns the corresponding randomness commitment.
func (v *SchnorrVerifier) RecomputeCommitment(zkp *SchnorrProof) (*math.G1, error) {
	// safety checks
	if zkp.Challenge == nil || zkp.Statement == nil {
		return nil, errors.Errorf("invalid zero-knowledge proof: nil challenge or statement")
	}
	if v.Curve == nil {
		return nil, errors.New("please initialize curve")
	}
	if len(zkp.Proof) > len(v.PedParams) {
		return nil, errors.Errorf("please initialize Pedersen parameters correctly")
	}
	com := v.Curve.NewG1()
	// compute commitment com = \Prod_{i=1}^n P_i^{proof_i}/Statement^{challenge}
	for i, p := range zkp.Proof {
		// more safety checks
		if v.PedParams == nil {
			return nil, errors.New("please initialize Pedersen parameters")
		}
		if p == nil {
			return nil, errors.New("invalid zero-knowledge proof: nil proof")
		}
		com.Add(v.PedParams[i].Mul(p))
	}
	// subtract Statement^{challenge}
	com.Sub(zkp.Statement.Mul(zkp.Challenge))
	return com, nil
}

func (v *SchnorrVerifier) RecomputeCommitments(zkps []*SchnorrProof, challenge *math.Zr) ([]*math.G1, error) {
	commitments := make([]*math.G1, len(zkps))
	var err error
	for i, zkp := range zkps {
		zkp.Challenge = challenge
		commitments[i], err = v.RecomputeCommitment(zkp)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compute commitment at index [%d]", i)
		}
	}
	return commitments, nil
}

// ComputeChallenge takes an array of bytes and returns the corresponding hash.
func (v *SchnorrVerifier) ComputeChallenge(raw []byte) (*math.Zr, error) {
	if v.Curve == nil {
		return nil, errors.New("failed to compute challenge: please initialize curve")
	}
	return v.Curve.HashToZr(raw), nil
}
