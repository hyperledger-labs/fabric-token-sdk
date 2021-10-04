/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package common

import (
	bn256 "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// Struct for Schnorr proofs
type SchnorrProof struct {
	Statement *bn256.G1
	Proof     []*bn256.Zr
	Challenge *bn256.Zr
}

type SchnorrVerifier struct {
	PedParams []*bn256.G1
	Curve     *bn256.Curve
}

type SchnorrProver struct {
	*SchnorrVerifier
	Witness    []*bn256.Zr
	Randomness []*bn256.Zr
	Challenge  *bn256.Zr
}

func (v *SchnorrVerifier) RecomputeCommitment(zkp *SchnorrProof) *bn256.G1 {
	com := v.Curve.NewG1()
	for i, p := range zkp.Proof {
		com.Add(v.PedParams[i].Mul(p))
	}
	com.Sub(zkp.Statement.Mul(zkp.Challenge))
	return com
}

func (v *SchnorrVerifier) RecomputeCommitments(zkps []*SchnorrProof, challenge *bn256.Zr) []*bn256.G1 {
	commitments := make([]*bn256.G1, len(zkps))
	for i, zkp := range zkps {
		zkp.Challenge = challenge
		commitments[i] = v.RecomputeCommitment(zkp)
	}
	return commitments
}

func (v *SchnorrVerifier) ComputeChallenge(pub PublicInput) *bn256.Zr {
	raw := pub.Bytes()
	return v.Curve.HashToZr(raw)
}

func (p *SchnorrProver) Prove() ([]*bn256.Zr, error) {
	if len(p.Witness) != len(p.Randomness) {
		return nil, errors.Errorf("cannot compute proof")
	}
	proof := make([]*bn256.Zr, len(p.Witness))
	for i := 0; i < len(proof); i++ {
		proof[i] = p.Curve.ModMul(p.Challenge, p.Witness[i], p.Curve.GroupOrder)
		proof[i] = p.Curve.ModAdd(proof[i], p.Randomness[i], p.Curve.GroupOrder)
	}
	return proof, nil
}

func ComputePedersenCommitment(opening []*bn256.Zr, base []*bn256.G1, c *bn256.Curve) (*bn256.G1, error) {
	if len(opening) != len(base) {
		return nil, errors.Errorf("can't compute Pedersen commitment [%d]!=[%d]", len(opening), len(base))
	}
	com := c.NewG1()
	for i := 0; i < len(base); i++ {
		com.Add(base[i].Mul(opening[i]))
	}
	return com, nil
}
