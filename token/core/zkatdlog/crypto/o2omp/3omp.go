/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package o2omp

import (
	"encoding/json"
	"strconv"

	"github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// prover for the one out of many proofs
type Prover struct {
	*Verifier
	witness *Witness
}

func NewProver(commitments []*math.G1, message []byte, pp []*math.G1, length int, index int, randomness *math.Zr, curve *math.Curve) *Prover {
	return &Prover{
		witness: &Witness{
			index:         index,
			comRandomness: randomness,
		},
		Verifier: NewVerifier(commitments, message, pp, length, curve),
	}
}

func NewVerifier(commitments []*math.G1, message []byte, pp []*math.G1, length int, curve *math.Curve) *Verifier {
	return &Verifier{
		Commitments:    commitments,
		Message:        message,
		PedersenParams: pp,
		BitLength:      length,
		Curve:          curve,
	}
}

// verifier for the one out of many proofs
type Verifier struct {
	Commitments    []*math.G1
	Message        []byte
	PedersenParams []*math.G1 // Pedersen commitments parameters
	BitLength      int
	Curve          *math.Curve
}

// Witness information
type Witness struct {
	index         int
	comRandomness *math.Zr
}

//
type Commitments struct {
	L []*math.G1
	A []*math.G1
	B []*math.G1
	D []*math.G1
}

type Values struct {
	L []*math.Zr
	A []*math.Zr
	B []*math.Zr
	D *math.Zr
}

type Proof struct {
	Commitments *Commitments
	Values      *Values
}

func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

func (p *Proof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

func (p *Prover) Prove() ([]byte, error) {
	if len(p.PedersenParams) != 2 {
		return nil, errors.Errorf("length of Pedersen parameters != 2")
	}
	if len(p.Commitments) != 1<<p.BitLength {
		return nil, errors.Errorf("number of commitments is not a power of 2, [%d][%d]", len(p.Commitments), 1<<p.BitLength)
	}
	indexBits := make([]int, p.BitLength)
	for i := 0; i < p.BitLength; i++ {
		indexBits[i] = (1 << uint(i)) & p.witness.index
	}

	// randomness
	a := make([]*math.Zr, p.BitLength)
	r := make([]*math.Zr, p.BitLength)
	s := make([]*math.Zr, p.BitLength)
	t := make([]*math.Zr, p.BitLength)
	rho := make([]*math.Zr, p.BitLength)
	proof := &Proof{}

	var err error
	proof.Commitments, err = p.compute3OMPCommitments(a, r, s, t, rho, indexBits)
	if err != nil {
		return nil, err
	}
	// compute challenge
	publicInput := common.GetG1Array(proof.Commitments.L, proof.Commitments.A, proof.Commitments.B, proof.Commitments.D, p.Commitments, p.PedersenParams)
	bytes := publicInput.Bytes()
	bytes = append(bytes, []byte(strconv.Itoa(p.BitLength))...)
	bytes = append(bytes, p.Message...)
	chal := p.Curve.HashToZr(bytes)

	p.computeO2OMProof(proof, indexBits, chal, a, r, s, t, rho)

	return proof.Serialize()
}

func (p *Prover) SetWitness(index int, randomness *math.Zr) {
	p.witness = &Witness{comRandomness: randomness, index: index}
}

func (v *Verifier) Verify(p []byte) error {
	if len(v.PedersenParams) != 2 {
		return errors.Errorf("length of Pedersen parameters != 2")
	}
	proof := &Proof{}
	err := proof.Deserialize(p)
	if err != nil {
		return err
	}

	if len(v.Commitments) != 1<<v.BitLength {
		return errors.Errorf("the number of commitments is not 2^bitlength [%v != %v]", len(v.Commitments), 1<<v.BitLength)
	}
	if len(proof.Commitments.L) != v.BitLength || len(proof.Commitments.A) != v.BitLength || len(proof.Commitments.B) != v.BitLength || len(proof.Commitments.D) != v.BitLength {
		return errors.Errorf("the size of the commitments in one out of many proof is not a multiple of %d", v.BitLength)
	}
	if len(proof.Values.L) != v.BitLength || len(proof.Values.A) != v.BitLength || len(proof.Values.B) != v.BitLength {
		return errors.Errorf("the size of the proofs in one out of many proof is not a multiple of %d", v.BitLength)
	}
	publicInput := common.GetG1Array(proof.Commitments.L, proof.Commitments.A, proof.Commitments.B, proof.Commitments.D, v.Commitments, v.PedersenParams)
	bytes := publicInput.Bytes()
	bytes = append(bytes, []byte(strconv.Itoa(v.BitLength))...)
	bytes = append(bytes, v.Message...)
	hash := v.Curve.HashToZr(bytes)

	for i := 0; i < v.BitLength; i++ {
		t := proof.Commitments.L[i].Mul(hash)
		t.Add(proof.Commitments.A[i])

		s := v.PedersenParams[0].Mul(proof.Values.L[i])
		s.Add(v.PedersenParams[1].Mul(proof.Values.A[i]))

		if !s.Equals(t) {
			return errors.Errorf("verification of first equation of one out of many proof failed")
		}

		t = proof.Commitments.L[i].Mul(v.Curve.ModSub(hash, proof.Values.L[i], v.Curve.GroupOrder))
		t.Add(proof.Commitments.B[i])

		if !t.Equals(v.PedersenParams[1].Mul(proof.Values.B[i])) {
			return errors.Errorf("verification of second equation of one out of many proof failed")
		}
	}

	s := v.Curve.NewG1()
	for j := 0; j < len(v.Commitments); j++ {
		f := v.Curve.NewZrFromInt(1)
		for i := 0; i < v.BitLength; i++ {
			b := (1 << uint(i)) & j // ith bit of j
			var t *math.Zr
			if b == 0 {
				t = v.Curve.ModSub(hash, proof.Values.L[i], v.Curve.GroupOrder)
			} else {
				t = proof.Values.L[i]
			}
			f = v.Curve.ModMul(t, f, v.Curve.GroupOrder)
		}
		s.Add(v.Commitments[j].Mul(f))
	}

	for i := 0; i < v.BitLength; i++ {
		power := hash.PowMod(v.Curve.NewZrFromInt(int64(i)))
		s.Sub(proof.Commitments.D[i].Mul(power))
	}

	if !s.Equals(v.PedersenParams[1].Mul(proof.Values.D)) {
		return errors.Errorf("verification of third equation of one out of many proof failed")
	}
	return nil
}

// structs for proof
type monomial struct {
	alpha *math.Zr
	beta  *math.Zr
}

type polynomial struct {
	coefficients []*math.Zr
}

func (p *Prover) compute3OMPCommitments(a, r, s, t, rho []*math.Zr, indexBits []int) (*Commitments, error) {
	commitments := &Commitments{}
	commitments.L = make([]*math.G1, p.BitLength)
	commitments.A = make([]*math.G1, p.BitLength)
	commitments.B = make([]*math.G1, p.BitLength)
	commitments.D = make([]*math.G1, p.BitLength)

	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.BitLength; i++ {
		//compute commitments
		a[i] = p.Curve.NewRandomZr(rand)
		r[i] = p.Curve.NewRandomZr(rand)
		s[i] = p.Curve.NewRandomZr(rand)
		t[i] = p.Curve.NewRandomZr(rand)
		rho[i] = p.Curve.NewRandomZr(rand)

		commitments.A[i] = p.PedersenParams[0].Mul(a[i])
		commitments.A[i].Add(p.PedersenParams[1].Mul(s[i]))

		commitments.L[i] = p.PedersenParams[1].Mul(r[i])
		commitments.B[i] = p.PedersenParams[1].Mul(t[i])

		if indexBits[i] > 0 {
			commitments.L[i].Add(p.PedersenParams[0])
			commitments.B[i].Add(p.PedersenParams[0].Mul(a[i]))
		}
	}
	f0, f1 := p.getfMonomials(indexBits, a)
	polynomials := p.getPolynomials(len(p.Commitments), p.BitLength, f0, f1)

	for i := 0; i < p.BitLength; i++ {
		commitments.D[i] = p.PedersenParams[1].Mul(rho[i])
		for j := 0; j < len(polynomials); j++ {
			if !polynomials[j].coefficients[i].Equals(p.Curve.NewZrFromInt(0)) {
				commitments.D[i].Add(p.Commitments[j].Mul(polynomials[j].coefficients[i]))
			}
		}
	}
	return commitments, nil
}

func (p *Prover) computeO2OMProof(proof *Proof, indexBits []int, chal *math.Zr, a, r, s, t, rho []*math.Zr) {
	proof.Values = &Values{}
	proof.Values.L = make([]*math.Zr, p.BitLength)
	proof.Values.A = make([]*math.Zr, p.BitLength)
	proof.Values.B = make([]*math.Zr, p.BitLength)
	proof.Values.D = p.Curve.NewZrFromInt(0)
	for i := 0; i < p.BitLength; i++ {
		proof.Values.L[i] = a[i]
		if indexBits[i] > 0 {
			proof.Values.L[i] = p.Curve.ModAdd(proof.Values.L[i], chal, p.Curve.GroupOrder)
		}

		proof.Values.A[i] = p.Curve.ModMul(r[i], chal, p.Curve.GroupOrder)
		proof.Values.A[i] = p.Curve.ModAdd(proof.Values.A[i], s[i], p.Curve.GroupOrder)

		proof.Values.B[i] = p.Curve.ModSub(chal, proof.Values.L[i], p.Curve.GroupOrder)
		proof.Values.B[i] = p.Curve.ModMul(proof.Values.B[i], r[i], p.Curve.GroupOrder)
		proof.Values.B[i] = p.Curve.ModAdd(proof.Values.B[i], t[i], p.Curve.GroupOrder)

		power := chal.PowMod(p.Curve.NewZrFromInt(int64(i)))
		proof.Values.D = p.Curve.ModAdd(p.Curve.ModMul(rho[i], power, p.Curve.GroupOrder), proof.Values.D, p.Curve.GroupOrder)
	}
	power := chal.PowMod(p.Curve.NewZrFromInt(int64(p.BitLength)))
	proof.Values.D = p.Curve.ModSub(p.Curve.ModMul(p.witness.comRandomness, power, p.Curve.GroupOrder), proof.Values.D, p.Curve.GroupOrder)
}

// get monomials for proof
func (p *Prover) getfMonomials(indexBits []int, a []*math.Zr) ([]*monomial, []*monomial) {
	f0 := make([]*monomial, len(a))
	f1 := make([]*monomial, len(a))
	for i, ai := range a {
		var b int64
		if indexBits[i] > 0 {
			b = 1
		}
		f1[i] = &monomial{}
		f1[i].alpha = p.Curve.NewZrFromInt(b)
		f1[i].beta = ai

		f0[i] = &monomial{}
		f0[i].alpha = p.Curve.NewZrFromInt(1 - b)
		f0[i].beta = p.Curve.ModNeg(ai, p.Curve.GroupOrder)
	}
	return f0, f1
}

// get polynomials for proof
func (p *Prover) getPolynomials(N, n int, f0, f1 []*monomial) []*polynomial {
	polynomials := make([]*polynomial, N)
	for i := 0; i < N; i++ {
		polynomials[i] = p.getPolynomialforIndex(i, n, f0, f1)
	}
	return polynomials
}

func (p *Prover) getPolynomialforIndex(j, n int, f0, f1 []*monomial) *polynomial {
	g := make([]*monomial, n)
	multiplier := p.Curve.NewZrFromInt(1)
	var roots []*math.Zr
	for i := 0; i < n; i++ {
		b := (1 << uint(i)) & j // i^th bit of j
		if b > 0 {
			g[i] = f1[i]
		} else {
			g[i] = f0[i]
		}
		// g = alpha x + beta (we refer to beta as root)
		if g[i].alpha.Equals(p.Curve.NewZrFromInt(1)) { // the alpha coefficient in g = 1, there is A root
			roots = append(roots, g[i].beta)
		}
		if g[i].alpha.Equals(p.Curve.NewZrFromInt(0)) { // the alpha coefficient in g = 0, there is no root
			multiplier = p.Curve.ModMul(multiplier, g[i].beta, p.Curve.GroupOrder)
		}
	}
	coefficients := p.getCoefficientsFromRoots(roots)

	poly := &polynomial{}
	poly.coefficients = make([]*math.Zr, n+1)
	for i := 0; i < len(coefficients); i++ {
		poly.coefficients[i] = p.Curve.ModMul(coefficients[i], multiplier, p.Curve.GroupOrder)
	}
	for i := len(coefficients); i < n+1; i++ {
		poly.coefficients[i] = p.Curve.NewZrFromInt(0)
	}
	return &polynomial{coefficients: poly.coefficients[:n]}
}

func (p *Prover) getCoefficientsFromRoots(roots []*math.Zr) []*math.Zr {
	coefficients := make([]*math.Zr, len(roots)+1)
	if len(roots) == 0 {
		coefficients[0] = p.Curve.NewZrFromInt(1)
	} else {
		coefficients[0] = roots[0]
		coefficients[1] = p.Curve.NewZrFromInt(1)
		for i := 2; i < len(coefficients); i++ {
			coefficients[i] = coefficients[i-1]
			for j := i - 1; j > 0; j-- {
				coefficients[j] = p.Curve.ModMul(coefficients[j], roots[i-1], p.Curve.GroupOrder)
				coefficients[j] = p.Curve.ModAdd(coefficients[j-1], coefficients[j], p.Curve.GroupOrder)
			}
			coefficients[0] = p.Curve.ModMul(coefficients[0], roots[i-1], p.Curve.GroupOrder)
		}
	}

	return coefficients
}
