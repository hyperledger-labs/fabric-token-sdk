/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package o2omp

import (
	"encoding/json"
	"strconv"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// prover for the one out of many proofs
type Prover struct {
	*Verifier
	witness *Witness
}

func NewProver(commitments []*bn256.G1, message []byte, pp []*bn256.G1, length int, index int, randomness *bn256.Zr) *Prover {
	return &Prover{
		witness: &Witness{
			index:         index,
			comRandomness: randomness,
		},
		Verifier: NewVerifier(commitments, message, pp, length),
	}
}

func NewVerifier(commitments []*bn256.G1, message []byte, pp []*bn256.G1, length int) *Verifier {
	return &Verifier{
		Commitments:    commitments,
		Message:        message,
		PedersenParams: pp,
		BitLength:      length,
	}
}

// verifier for the one out of many proofs
type Verifier struct {
	Commitments    []*bn256.G1
	Message        []byte
	PedersenParams []*bn256.G1 // Pedersen commitments parameters
	BitLength      int
}

// Witness information
type Witness struct {
	index         int
	comRandomness *bn256.Zr
}

//
type Commitments struct {
	L []*bn256.G1
	A []*bn256.G1
	B []*bn256.G1
	D []*bn256.G1
}

type Values struct {
	L []*bn256.Zr
	A []*bn256.Zr
	B []*bn256.Zr
	D *bn256.Zr
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
	a := make([]*bn256.Zr, p.BitLength)
	r := make([]*bn256.Zr, p.BitLength)
	s := make([]*bn256.Zr, p.BitLength)
	t := make([]*bn256.Zr, p.BitLength)
	rho := make([]*bn256.Zr, p.BitLength)
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
	chal := bn256.HashModOrder(bytes)

	p.computeO2OMProof(proof, indexBits, chal, a, r, s, t, rho)

	return proof.Serialize()
}

func (p *Prover) SetWitness(index int, randomness *bn256.Zr) {
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
	hash := bn256.HashModOrder(bytes)

	for i := 0; i < v.BitLength; i++ {
		t := proof.Commitments.L[i].Mul(hash)
		t.Add(proof.Commitments.A[i])

		s := v.PedersenParams[0].Mul(proof.Values.L[i])
		s.Add(v.PedersenParams[1].Mul(proof.Values.A[i]))

		if !s.Equals(t) {
			return errors.Errorf("verification of first equation of one out of many proof failed")
		}

		t = proof.Commitments.L[i].Mul(bn256.ModSub(hash, proof.Values.L[i], bn256.Order))
		t.Add(proof.Commitments.B[i])

		if !t.Equals(v.PedersenParams[1].Mul(proof.Values.B[i])) {
			return errors.Errorf("verification of second equation of one out of many proof failed")
		}
	}

	s := bn256.NewG1()
	for j := 0; j < len(v.Commitments); j++ {
		f := bn256.NewZrInt(1)
		for i := 0; i < v.BitLength; i++ {
			b := (1 << uint(i)) & j // ith bit of j
			var t *bn256.Zr
			if b == 0 {
				t = bn256.ModSub(hash, proof.Values.L[i], bn256.Order)
			} else {
				t = proof.Values.L[i]
			}
			f = bn256.ModMul(t, f, bn256.Order)
		}
		s.Add(v.Commitments[j].Mul(f))
	}

	for i := 0; i < v.BitLength; i++ {
		power := hash.PowMod(bn256.NewZrInt(i), bn256.Order)
		s.Sub(proof.Commitments.D[i].Mul(power))
	}

	if !s.Equals(v.PedersenParams[1].Mul(proof.Values.D)) {
		return errors.Errorf("verification of third equation of one out of many proof failed")
	}
	return nil
}

// structs for proof
type monomial struct {
	alpha *bn256.Zr
	beta  *bn256.Zr
}

type polynomial struct {
	coefficients []*bn256.Zr
}

func (p *Prover) compute3OMPCommitments(a, r, s, t, rho []*bn256.Zr, indexBits []int) (*Commitments, error) {
	commitments := &Commitments{}
	commitments.L = make([]*bn256.G1, p.BitLength)
	commitments.A = make([]*bn256.G1, p.BitLength)
	commitments.B = make([]*bn256.G1, p.BitLength)
	commitments.D = make([]*bn256.G1, p.BitLength)

	rand, err := bn256.GetRand()
	if err != nil {
		return nil, err
	}
	for i := 0; i < p.BitLength; i++ {
		//compute commitments
		a[i] = bn256.RandModOrder(rand)
		r[i] = bn256.RandModOrder(rand)
		s[i] = bn256.RandModOrder(rand)
		t[i] = bn256.RandModOrder(rand)
		rho[i] = bn256.RandModOrder(rand)

		commitments.A[i] = p.PedersenParams[0].Mul(a[i])
		commitments.A[i].Add(p.PedersenParams[1].Mul(s[i]))

		commitments.L[i] = p.PedersenParams[1].Mul(r[i])
		commitments.B[i] = p.PedersenParams[1].Mul(t[i])

		if indexBits[i] > 0 {
			commitments.L[i].Add(p.PedersenParams[0])
			commitments.B[i].Add(p.PedersenParams[0].Mul(a[i]))
		}
	}
	f0, f1 := getfMonomials(indexBits, a)
	polynomials := getPolynomials(len(p.Commitments), p.BitLength, f0, f1)

	for i := 0; i < p.BitLength; i++ {
		commitments.D[i] = p.PedersenParams[1].Mul(rho[i])
		for j := 0; j < len(polynomials); j++ {
			if !polynomials[j].coefficients[i].IsZero() {
				commitments.D[i].Add(p.Commitments[j].Mul(polynomials[j].coefficients[i]))
			}
		}
	}
	return commitments, nil
}

func (p *Prover) computeO2OMProof(proof *Proof, indexBits []int, chal *bn256.Zr, a, r, s, t, rho []*bn256.Zr) {
	proof.Values = &Values{}
	proof.Values.L = make([]*bn256.Zr, p.BitLength)
	proof.Values.A = make([]*bn256.Zr, p.BitLength)
	proof.Values.B = make([]*bn256.Zr, p.BitLength)
	proof.Values.D = bn256.NewZrInt(0)
	for i := 0; i < p.BitLength; i++ {
		proof.Values.L[i] = a[i]
		if indexBits[i] > 0 {
			proof.Values.L[i] = bn256.ModAdd(proof.Values.L[i], chal, bn256.Order)
		}

		proof.Values.A[i] = bn256.ModMul(r[i], chal, bn256.Order)
		proof.Values.A[i] = bn256.ModAdd(proof.Values.A[i], s[i], bn256.Order)

		proof.Values.B[i] = bn256.ModSub(chal, proof.Values.L[i], bn256.Order)
		proof.Values.B[i] = bn256.ModMul(proof.Values.B[i], r[i], bn256.Order)
		proof.Values.B[i] = bn256.ModAdd(proof.Values.B[i], t[i], bn256.Order)

		power := chal.PowMod(bn256.NewZrInt(i), bn256.Order)
		proof.Values.D = bn256.ModAdd(bn256.ModMul(rho[i], power, bn256.Order), proof.Values.D, bn256.Order)
	}
	power := chal.PowMod(bn256.NewZrInt(p.BitLength), bn256.Order)
	proof.Values.D = bn256.ModSub(bn256.ModMul(p.witness.comRandomness, power, bn256.Order), proof.Values.D, bn256.Order)
}

// get monomials for proof
func getfMonomials(indexBits []int, a []*bn256.Zr) ([]*monomial, []*monomial) {
	f0 := make([]*monomial, len(a))
	f1 := make([]*monomial, len(a))
	for i, ai := range a {
		var b int
		if indexBits[i] > 0 {
			b = 1
		}
		f1[i] = &monomial{}
		f1[i].alpha = bn256.NewZrInt(b)
		f1[i].beta = ai

		f0[i] = &monomial{}
		f0[i].alpha = bn256.NewZrInt(1 - b)
		f0[i].beta = bn256.ModNeg(ai, bn256.Order)
	}
	return f0, f1
}

// get polynomials for proof
func getPolynomials(N, n int, f0, f1 []*monomial) []*polynomial {
	polynomials := make([]*polynomial, N)
	for i := 0; i < N; i++ {
		polynomials[i] = getPolynomialforIndex(i, n, f0, f1)
	}
	return polynomials
}

func getPolynomialforIndex(j, n int, f0, f1 []*monomial) *polynomial {
	g := make([]*monomial, n)
	multiplier := bn256.NewZrInt(1)
	var roots []*bn256.Zr
	for i := 0; i < n; i++ {
		b := (1 << uint(i)) & j // i^th bit of j
		if b > 0 {
			g[i] = f1[i]
		} else {
			g[i] = f0[i]
		}
		// g = alpha x + beta (we refer to beta as root)
		if g[i].alpha.Cmp(bn256.NewZrInt(1)) == 0 { // the alpha coefficient in g = 1, there is A root
			roots = append(roots, g[i].beta)
		}
		if g[i].alpha.IsZero() { // the alpha coefficient in g = 0, there is no root
			multiplier = bn256.ModMul(multiplier, g[i].beta, bn256.Order)
		}
	}
	coefficients := GetCoefficientsFromRoots(roots)

	p := &polynomial{}
	p.coefficients = make([]*bn256.Zr, n+1)
	for i := 0; i < len(coefficients); i++ {
		p.coefficients[i] = bn256.ModMul(coefficients[i], multiplier, bn256.Order)
	}
	for i := len(coefficients); i < n+1; i++ {
		p.coefficients[i] = bn256.NewZrInt(0)
	}
	return &polynomial{coefficients: p.coefficients[:n]}
}

func GetCoefficientsFromRoots(roots []*bn256.Zr) []*bn256.Zr {
	coefficients := make([]*bn256.Zr, len(roots)+1)
	if len(roots) == 0 {
		coefficients[0] = bn256.NewZrInt(1)
	} else {
		coefficients[0] = roots[0]
		coefficients[1] = bn256.NewZrInt(1)
		for i := 2; i < len(coefficients); i++ {
			coefficients[i] = coefficients[i-1]
			for j := i - 1; j > 0; j-- {
				coefficients[j] = bn256.ModMul(coefficients[j], roots[i-1], bn256.Order)
				coefficients[j] = bn256.ModAdd(coefficients[j-1], coefficients[j], bn256.Order)
			}
			coefficients[0] = bn256.ModMul(coefficients[0], roots[i-1], bn256.Order)
		}
	}

	return coefficients
}
