/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
)

// cspProver instantiates a CSP prover
type cspProver struct {
	Commitment     *mathlib.G1    // Commitment to the witness
	Generators     []*mathlib.G1  // Generators for pedersen commitment
	LinearForm     []*mathlib.Zr  // Linear function
	Value          *mathlib.Zr    // Claimed value of linear function over witness
	NumberOfRounds uint64         // log(length of vectors), assume power of 2
	Curve          *mathlib.Curve // Curve identifier

	witness []*mathlib.Zr // opening for the commitment
}

// CSPProof is the non-interactive compressed sigma-protocol proof for a linear
// form evaluation over a Pedersen commitment.
//
// At each of the NumberOfRounds folding steps the prover supplies:
//
//	Left[i]   = MSM(gen_L, wit_R)   — cross-commitment
//	Right[i]  = MSM(gen_R, wit_L)   — cross-commitment
//	VLeft[i]  = ⟨f_L, wit_R⟩       — cross scalar (ModAddMul)
//	VRight[i] = ⟨f_R, wit_L⟩       — cross scalar (ModAddMul)
//
// The verifier reproduces all Fiat-Shamir challenges c_0,…,c_{k-1} and tracks
//
//	C' = c·C + Left  + c²·Right
//	v' = c·v + VLeft + c²·VRight
//
// Then computes gen_f and f_f via a single final MSM / inner-product using the
// coefficient vector s (see cspSVector), and checks
//
//	com_f^{f_f} == gen_f^{val_f}
type CSPProof struct {
	Left   []*mathlib.G1 // cross commitment MSM(gen_L, wit_R)
	Right  []*mathlib.G1 // cross commitment MSM(gen_R, wit_L)
	VLeft  []*mathlib.Zr // cross scalar <f_L, wit_R>
	VRight []*mathlib.Zr // cross scalar <f_R, wit_L>

	Curve *mathlib.Curve
}

// Prove generates a CSP proof. Generators, LinearForm, and witness must all
// have length exactly 2^NumberOfRounds.
//
// Per-round folding rules (c is the Fiat-Shamir challenge):
//
//	gen'_j = gen_L[j] + c · gen_R[j]
//	f'_j   = f_L[j]   + c · f_R[j]
//	w'_j   = c · w_L[j] + w_R[j]
func (p *cspProver) Prove() (*CSPProof, error) {
	// Check that sizes of Generators, LinearForm and witness is 2^{NumberOfRounds}.
	expected := uint64(1) << p.NumberOfRounds
	if uint64(len(p.Generators)) != expected ||
		uint64(len(p.LinearForm)) != expected ||
		uint64(len(p.witness)) != expected {
		return nil, errors.New("inconsistent vector sizes: each must have length 2^NumberOfRounds")
	}

	// Initialize transcript.
	tr := Transcript{Curve: p.Curve}
	tr.InitHasher()

	// Absorb Commitment || Generators || LinearForm || Value into transcript.
	tr.Absorb(p.Commitment.Bytes())
	for _, g := range p.Generators {
		tr.Absorb(g.Bytes())
	}
	for _, f := range p.LinearForm {
		tr.Absorb(f.Bytes())
	}
	tr.Absorb(p.Value.Bytes())

	left := make([]*mathlib.G1, p.NumberOfRounds)
	right := make([]*mathlib.G1, p.NumberOfRounds)
	vLeft := make([]*mathlib.Zr, p.NumberOfRounds)
	vRight := make([]*mathlib.Zr, p.NumberOfRounds)

	// Working copies — G1 Add mutates the receiver in-place, so deep-copy generators.
	// Zr values are replaced by ModAdd/ModMul (pointer swap), so shallow copy suffices.
	generators := make([]*mathlib.G1, len(p.Generators))
	for i, g := range p.Generators {
		generators[i] = g.Copy()
	}
	linearForm := make([]*mathlib.Zr, len(p.LinearForm))
	copy(linearForm, p.LinearForm)
	witness := make([]*mathlib.Zr, len(p.witness))
	copy(witness, p.witness)

	for i := range p.NumberOfRounds {
		n := len(generators) / 2

		// Cross-commitments via MSM:
		//   left[i]  = MSM(gen_L, wit_R)   gen_L = generators[:n], wit_R = witness[n:]
		//   right[i] = MSM(gen_R, wit_L)   gen_R = generators[n:], wit_L = witness[:n]
		left[i] = p.Curve.MultiScalarMul(generators[:n], witness[n:])
		right[i] = p.Curve.MultiScalarMul(generators[n:], witness[:n])

		// Cross scalar products via ModAddMul (scalar-field MSM):
		//   vLeft[i]  = ⟨f_L, wit_R⟩
		//   vRight[i] = ⟨f_R, wit_L⟩
		vLeft[i] = math.InnerProduct(linearForm[:n], witness[n:], p.Curve)
		vRight[i] = math.InnerProduct(linearForm[n:], witness[:n], p.Curve)

		// Absorb cross terms into transcript, then squeeze challenge.
		tr.Absorb(left[i].Bytes())
		tr.Absorb(right[i].Bytes())
		tr.Absorb(vLeft[i].Bytes())
		tr.Absorb(vRight[i].Bytes())

		c, err := tr.Squeeze()
		if err != nil {
			return nil, errors.New("unable to generate random challenge")
		}

		// Fold generators:  gen'[j] = gen_L[j] + c · gen_R[j]
		// Fold linear form: f'[j]   = f_L[j]   + c · f_R[j]
		// Fold witness:     w'[j]   = c · w_L[j] + w_R[j]
		for j := range n {
			generators[j].Add(generators[n+j].Mul(c))

			linearForm[j] = p.Curve.ModAdd(
				linearForm[j],
				p.Curve.ModMul(c, linearForm[n+j], p.Curve.GroupOrder),
				p.Curve.GroupOrder,
			)

			witness[j] = p.Curve.ModAdd(
				p.Curve.ModMul(c, witness[j], p.Curve.GroupOrder),
				witness[n+j],
				p.Curve.GroupOrder,
			)
		}

		generators = generators[:n]
		linearForm = linearForm[:n]
		witness = witness[:n]
	}

	return &CSPProof{
		Left:   left,
		Right:  right,
		VLeft:  vLeft,
		VRight: vRight,
		Curve:  p.Curve,
	}, nil
}

// cspVerifier verifies a CSPProof against a public statement.
type cspVerifier struct {
	Commitment     *mathlib.G1   // Pedersen commitment C = MSM(gen, w)
	Generators     []*mathlib.G1 // Commitment generators
	LinearForm     []*mathlib.Zr // Coefficients of the linear form f
	Value          *mathlib.Zr   // Claimed evaluation v = ⟨f, w⟩
	NumberOfRounds uint64        // log₂(vector length)
	Curve          *mathlib.Curve
}

// Verify checks that proof is a valid CSP proof for the statement
// (Commitment, Generators, LinearForm, Value).
// Then it computes gen_final and f_final via a single final MSM using the coefficient
// vector s (see cspSVector), and checks:
//
//	com_f^{f_f} == gen_f^{val_f}
//
// This holds iff there exists w_f : com_f = MSM(gen_f, w_f) ∧ val_f = f_f · w_f.
// This is similar to the optimisation used in Bulletproof verifier:
// See: Page 17, Section 3, https://eprint.iacr.org/2017/1066.pdf
func (v *cspVerifier) Verify(proof *CSPProof) error {
	if uint64(len(proof.Left)) != v.NumberOfRounds ||
		uint64(len(proof.Right)) != v.NumberOfRounds ||
		uint64(len(proof.VLeft)) != v.NumberOfRounds ||
		uint64(len(proof.VRight)) != v.NumberOfRounds {
		return errors.New("malformed proof: wrong number of rounds")
	}

	// Initialize transcript — must mirror Prove() exactly.
	tr := Transcript{Curve: v.Curve}
	tr.InitHasher()
	tr.Absorb(v.Commitment.Bytes())
	for _, g := range v.Generators {
		tr.Absorb(g.Bytes())
	}
	for _, f := range v.LinearForm {
		tr.Absorb(f.Bytes())
	}
	tr.Absorb(v.Value.Bytes())

	// Replay transcript to collect all challenges, tracking commitment and
	// value updates. Generators are NOT folded per round.
	challenges := make([]*mathlib.Zr, v.NumberOfRounds)
	com := v.Commitment.Copy()
	val := v.Value.Copy()

	for i := range v.NumberOfRounds {
		// Absorb cross terms (same order as Prove()).
		tr.Absorb(proof.Left[i].Bytes())
		tr.Absorb(proof.Right[i].Bytes())
		tr.Absorb(proof.VLeft[i].Bytes())
		tr.Absorb(proof.VRight[i].Bytes())

		c, err := tr.Squeeze()
		if err != nil {
			return errors.New("unable to recompute challenge")
		}
		challenges[i] = c
		cSq := v.Curve.ModMul(c, c, v.Curve.GroupOrder)

		// Update commitment: C' = c·C + Left[i] + c²·Right[i]
		newCom := com.Mul(c)
		newCom.Add(proof.Left[i])
		newCom.Add(proof.Right[i].Mul(cSq))
		com = newCom

		// Update value: v' = c·v + VLeft[i] + c²·VRight[i]
		val = v.Curve.ModAdd(
			v.Curve.ModMul(c, val, v.Curve.GroupOrder),
			v.Curve.ModAdd(
				proof.VLeft[i],
				v.Curve.ModMul(cSq, proof.VRight[i], v.Curve.GroupOrder),
				v.Curve.GroupOrder,
			),
			v.Curve.GroupOrder,
		)
	}

	// Compute the coefficient vector s such that
	//   gen_f = sum_i s[i] · gen[i]   and   f_f = sum_i s[i] · f[i]
	// then evaluate both via a single MSM and a single inner product.
	n := 1 << v.NumberOfRounds
	s := cspSVector(n, challenges, v.Curve)

	// gen_f = MSM(Generators, s)
	genF := v.Curve.MultiScalarMul(v.Generators, s)

	// f_f = ⟨s, LinearForm⟩  (scalar-field MSM via ModAddMul)
	fF := math.InnerProduct(s, v.LinearForm, v.Curve)

	// Final check: com_f^{f_f} == gen_f^{val_f}
	lhs := com.Mul(fF)
	rhs := genF.Mul(val)
	if !lhs.Equals(rhs) {
		return errors.New("CSP proof verification failed")
	}

	return nil
}

// cspSVector computes the length-n coefficient vector s such that the
// final folded generator gen_f = sum_i s[i] · gen[i].
//
// Under the folding rule gen'_j = gen_L[j] + c_r · gen_R[j], after k=log₂(n)
// rounds with challenges c_0,…,c_{k-1}:
//
//	s[i] = ∏_{r=0}^{k-1} c_r^{ bit(i, k-1-r) }
//
// where bit(i,m) is the m-th bit of i (0 = LSB). The vector is built in O(n)
// multiplications using the recurrence: s[i + 2^r] = s[i] · c_{k-1-r}.
func cspSVector(n int, challenges []*mathlib.Zr, curve *mathlib.Curve) []*mathlib.Zr {
	k := len(challenges)
	s := make([]*mathlib.Zr, n)
	s[0] = math.One(curve)
	for r := range k {
		halfLen := 1 << r      // number of entries already filled
		c := challenges[k-1-r] // reverse order matches bit(i, k-1-r)
		for i := range halfLen {
			s[i+halfLen] = curve.ModMul(s[i], c, curve.GroupOrder)
		}
	}

	return s
}
