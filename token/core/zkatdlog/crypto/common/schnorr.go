/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	"github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// Struct for Schnorr proofs
type SchnorrProof struct {
	Statement *math.G1
	Proof     []*math.Zr
	Challenge *math.Zr
}

type SchnorrVerifier struct {
	PedParams []*math.G1
	Curve     *math.Curve
}

type SchnorrProver struct {
	*SchnorrVerifier
	Witness    []*math.Zr
	Randomness []*math.Zr
	Challenge  *math.Zr
}

func (v *SchnorrVerifier) RecomputeCommitment(zkp *SchnorrProof) (*math.G1, error) {
	if zkp.Challenge == nil {
		return nil, errors.Errorf("invalid zero-knowledge proof")
	}
	com := v.Curve.NewG1()
	for i, p := range zkp.Proof {
		if p == nil {
			return nil, errors.Errorf("invalid zero-knowledge proof")
		}
		com.Add(v.PedParams[i].Mul(p))
	}

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
			return nil, err
		}
	}
	return commitments, nil
}

func (v *SchnorrVerifier) ComputeChallenge(pub PublicInput) *math.Zr {
	raw := pub.Bytes()
	return v.Curve.HashToZr(raw)
}

func (p *SchnorrProver) Prove() ([]*math.Zr, error) {
	if len(p.Witness) != len(p.Randomness) {
		return nil, errors.Errorf("cannot compute proof")
	}
	proof := make([]*math.Zr, len(p.Witness))
	for i := 0; i < len(proof); i++ {
		proof[i] = p.Curve.ModMul(p.Challenge, p.Witness[i], p.Curve.GroupOrder)
		proof[i] = p.Curve.ModAdd(proof[i], p.Randomness[i], p.Curve.GroupOrder)
	}
	return proof, nil
}

func ComputePedersenCommitment(opening []*math.Zr, base []*math.G1, c *math.Curve) (*math.G1, error) {
	if len(opening) != len(base) {
		return nil, errors.Errorf("can't compute Pedersen commitment [%d]!=[%d]", len(opening), len(base))
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
