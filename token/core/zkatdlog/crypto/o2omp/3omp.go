/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package o2omp

import (
	"encoding/json"
	"strconv"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// Prover produces a one-out-of-many proof
// A one-out-of-many proof allows a prover who is given a list of commitments
// (c_1, ..., c_n) to show that they know that there is i \in [N] and r_i \in Z_p
// such that c_i = Q^r_i (i.e., c_i is commitment to 0)
type Prover struct {
	*Verifier
	witness *Witness
}

// Verifier checks the validity of one-out-of-many proofs
type Verifier struct {
	Commitments    []*math.G1
	Message        []byte
	PedersenParams []*math.G1 // Pedersen commitments parameters
	BitLength      int
	Curve          *math.Curve
}

// NewProver returns a Prover instantiated with the passed arguments
func NewProver(commitments []*math.G1, message []byte, pp []*math.G1, length int, index int, randomness *math.Zr, curve *math.Curve) *Prover {
	return &Prover{
		witness: &Witness{
			index:         index,
			comRandomness: randomness,
		},
		Verifier: NewVerifier(commitments, message, pp, length, curve),
	}
}

// NewVerifier returns a Verifier instantiated with the passed arguments
func NewVerifier(commitments []*math.G1, message []byte, pp []*math.G1, length int, curve *math.Curve) *Verifier {
	return &Verifier{
		Commitments:    commitments,
		Message:        message,
		PedersenParams: pp,
		BitLength:      length,
		Curve:          curve,
	}
}

// Witness represents the secret information of one-out-of-many proofs
type Witness struct {
	// index is the index of commitment c_i = Q^r_i in  the list of
	// commitments (c_1, ..., c_n)
	index int
	// comRandomness corresponds to r_i such that c_i = Q^r_i
	comRandomness *math.Zr
}

// Commitments to the randomness used in the one-out-of-many proof
type Commitments struct {
	L []*math.G1
	A []*math.G1
	B []*math.G1
	D []*math.G1
}

// Values corresponds to r+w*c, where c is the challenge in one-out-of-many proof,
// r is a random number and w is a secret information known to the prover.
// The Prover uses Values to show knowledge of (i, r_i) such that
// c_i \in {c_1, ..., c_n} corresponds to c_i = H^r_i
type Values struct {
	L []*math.Zr
	A []*math.Zr
	B []*math.Zr
	D *math.Zr
}

// Proof is a one-out-of-many proof
type Proof struct {
	Commitments *Commitments
	Values      *Values
}

// Serialize marshals Proof
func (p *Proof) Serialize() ([]byte, error) {
	return json.Marshal(p)
}

// Deserialize un-marshals Proof
func (p *Proof) Deserialize(raw []byte) error {
	return json.Unmarshal(raw, p)
}

// Prove produces a one-out-of-many proof
func (p *Prover) Prove() ([]byte, error) {
	if len(p.PedersenParams) != 2 {
		return nil, errors.Errorf("length of Pedersen parameters != 2")
	}
	if len(p.Commitments) != 1<<p.BitLength {
		return nil, errors.Errorf("number of commitments is not a power of 2, [%d][%d]", len(p.Commitments), 1<<p.BitLength)
	}
	indexBits := make([]int, p.BitLength)
	for i := 0; i < p.BitLength; i++ {
		b := (1 << uint(i)) & p.witness.index
		if b > 0 {
			indexBits[i] = 1
		} else {
			indexBits[i] = 0
		}
	}

	// generate randomness
	a := make([]*math.Zr, p.BitLength)
	r := make([]*math.Zr, p.BitLength)
	s := make([]*math.Zr, p.BitLength)
	t := make([]*math.Zr, p.BitLength)
	rho := make([]*math.Zr, p.BitLength)
	proof := &Proof{}

	// commit to randomness
	var err error
	proof.Commitments, err = p.compute3OMPCommitments(a, r, s, t, rho, indexBits)
	if err != nil {
		return nil, err
	}
	// compute challenge
	publicInput := common.GetG1Array(proof.Commitments.L, proof.Commitments.A, proof.Commitments.B, proof.Commitments.D, p.Commitments, p.PedersenParams)
	bytes, err := publicInput.Bytes()
	if err != nil {
		return nil, err
	}
	bytes = append(bytes, []byte(strconv.Itoa(p.BitLength))...)
	bytes = append(bytes, p.Message...)
	chal := p.Curve.HashToZr(bytes)
	// compute proof
	p.computeO2OMProof(proof, indexBits, chal, a, r, s, t, rho)

	return proof.Serialize()
}

// Verify checks the validity of a serialized one-out-of-many proof
func (v *Verifier) Verify(p []byte) error {
	if v.Curve == nil || v.Curve.GroupOrder == nil {
		return errors.New("cannot verify one-out-of-many proof: please initialize curve")
	}
	if len(v.PedersenParams) != 2 {
		return errors.Errorf("cannot verify one-out-of-many proof: length of Pedersen parameters != 2")
	}

	if len(v.Commitments) != 1<<v.BitLength {
		return errors.Errorf("cannot verify one-out-of-many proof: the number of commitments is not 2^bitlength [%v != %v]", len(v.Commitments), 1<<v.BitLength)
	}
	proof := &Proof{}
	err := proof.Deserialize(p)
	if err != nil {
		return err
	}

	if proof.Commitments == nil || proof.Values == nil {
		return errors.New("cannot verify one-out-of-many proof: nil elements in proof")
	}
	if len(proof.Commitments.L) != v.BitLength || len(proof.Commitments.A) != v.BitLength || len(proof.Commitments.B) != v.BitLength || len(proof.Commitments.D) != v.BitLength {
		return errors.Errorf("the size of the commitments in one out of many proof is not a multiple of %d", v.BitLength)
	}
	if len(proof.Values.L) != v.BitLength || len(proof.Values.A) != v.BitLength || len(proof.Values.B) != v.BitLength {
		return errors.Errorf("the size of the proofs in one out of many proof is not a multiple of %d", v.BitLength)
	}
	publicInput := common.GetG1Array(proof.Commitments.L, proof.Commitments.A, proof.Commitments.B, proof.Commitments.D, v.Commitments, v.PedersenParams)
	bytes, err := publicInput.Bytes() // Bytes() returns an error if one of the elements in publicInput is nil
	if err != nil {
		return errors.Wrap(err, "invalid one-out-of-many proof")
	}
	bytes = append(bytes, []byte(strconv.Itoa(v.BitLength))...)
	bytes = append(bytes, v.Message...)

	hash := v.Curve.HashToZr(bytes)

	for i := 0; i < v.BitLength; i++ {
		if proof.Values.A[i] == nil || proof.Values.B[i] == nil || proof.Values.L[i] == nil {
			return errors.New("invalid one-out-of-many proof: nil elements in proof")
		}
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
	if proof.Values.D == nil {
		return errors.New("invalid one-out-of-many proof: nil elements in proof")
	}
	if !s.Equals(v.PedersenParams[1].Mul(proof.Values.D)) {
		return errors.Errorf("verification of third equation of one out of many proof failed")
	}
	return nil
}

// monomial is one degree polynomial, used in the one-out-of-many proof
type monomial struct {
	alpha *math.Zr
	beta  *math.Zr
}

// polynomial is used in one-out-of-many proof to show
// that index i is in [N]
type polynomial struct {
	coefficients []*math.Zr
}

// compute3OMPCommitments commits to the randomness used to generate one-out-of-many proofs
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

// computeO2OMProof computes Proof.Values with the
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

// getfMonomials returns the monomials used in the one-out-of-many proofs
// Monomials are used to show a value b is a bit.
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

// getPolynomials returns an array of N polynomials (one polynomial per index i \in [N])
// Each polynomial is a function of monomials f0 and f1
func (p *Prover) getPolynomials(N, n int, f0, f1 []*monomial) []*polynomial {
	polynomials := make([]*polynomial, N)
	for i := 0; i < N; i++ {
		polynomials[i] = p.getPolynomialforIndex(i, n, f0, f1)
	}
	return polynomials
}

// getPolynomialforIndex returns a polynomial which is computed as function of
// f0 and f1 and index j
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

// getCoefficientsFromRoots takes an array of polynomial roots a returns the coefficients of
// the polynomial
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
