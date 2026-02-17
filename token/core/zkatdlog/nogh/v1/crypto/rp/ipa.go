/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common/encoding/asn1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
)

// IPA contains the proof for the inner product argument.
type IPA struct {
	// Left is the final reduced scalar from the left vector.
	Left *mathlib.Zr
	// Right is the final reduced scalar from the right vector.
	Right *mathlib.Zr
	// L contains the intermediate commitments for each round of the reduction.
	L []*mathlib.G1
	// R contains the intermediate commitments for each round of the reduction.
	R []*mathlib.G1
}

// Serialize marshals the IPA into a byte slice.
func (ipa *IPA) Serialize() ([]byte, error) {
	lArray, err := asn1.NewElementArray(ipa.L)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize L")
	}
	rArray, err := asn1.NewElementArray(ipa.R)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to serialize R")
	}
	return asn1.MarshalMath(ipa.Left, ipa.Right, lArray, rArray)
}

// Deserialize unmarshals a byte slice into the IPA.
func (ipa *IPA) Deserialize(raw []byte) error {
	unmarshaller, err := asn1.NewUnmarshaller(raw)
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize raw")
	}
	ipa.Left, err = unmarshaller.NextZr()
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize Left")
	}
	ipa.Right, err = unmarshaller.NextZr()
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize Right")
	}
	ipa.L, err = unmarshaller.NextG1Array()
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize L")
	}
	ipa.R, err = unmarshaller.NextG1Array()
	if err != nil {
		return errors.Wrapf(err, "failed to deserialize R")
	}
	return nil
}

// Validate checks that the IPA proof elements are valid for the given curve.
func (ipa *IPA) Validate(curve mathlib.CurveID) error {
	if ipa.Left == nil {
		return errors.New("invalid IPA proof: nil Left")
	}
	if ipa.Right == nil {
		return errors.New("invalid IPA proof: nil Right")
	}
	if ipa.L == nil {
		return errors.New("invalid IPA proof: nil L")
	}
	if ipa.R == nil {
		return errors.New("invalid IPA proof: nil R")
	}
	if err := math.CheckZrElements(ipa.L, curve, uint64(len(ipa.L))); err != nil {
		return errors.Wrapf(err, "invalid IPA proof: invalid L elements")
	}
	if err := math.CheckZrElements(ipa.R, curve, uint64(len(ipa.R))); err != nil {
		return errors.Wrapf(err, "invalid IPA proof: invalid R elements")
	}
	return nil
}

// ipaProver manages the creation of an inner product argument proof.
type ipaProver struct {
	// rightVector is the first vector of the inner product.
	rightVector []*mathlib.Zr
	// leftVector is the second vector of the inner product.
	leftVector []*mathlib.Zr
	// InnerProduct is the expected inner product result.
	InnerProduct *mathlib.Zr
	// Q is an auxiliary generator.
	Q *mathlib.G1
	// RightGenerators are the generators for the right vector.
	RightGenerators []*mathlib.G1
	// LeftGenerators are the generators for the left vector.
	LeftGenerators []*mathlib.G1
	// Commitment is the Pedersen commitment to the vectors.
	Commitment *mathlib.G1
	// NumberOfRounds is log2 of the vector length.
	NumberOfRounds uint64
	// Curve is the mathematical curve.
	Curve *mathlib.Curve
}

// NewIPAProver returns a new ipaProver instance.
func NewIPAProver(
	innerProduct *mathlib.Zr,
	leftVector, rightVector []*mathlib.Zr,
	Q *mathlib.G1,
	leftGens, rightGens []*mathlib.G1,
	Commitment *mathlib.G1,
	rounds uint64,
	c *mathlib.Curve,
) *ipaProver {
	return &ipaProver{
		InnerProduct:    innerProduct,
		rightVector:     rightVector,
		leftVector:      leftVector,
		RightGenerators: rightGens,
		LeftGenerators:  leftGens,
		Curve:           c,
		NumberOfRounds:  rounds,
		Commitment:      Commitment,
		Q:               Q,
	}
}

// Prove generates an inner product argument proof.
func (p *ipaProver) Prove() (*IPA, error) {
	array := common.GetG1Array(p.RightGenerators, p.LeftGenerators, []*mathlib.G1{p.Q, p.Commitment})
	bytesToHash := make([][]byte, 3)
	var err error
	bytesToHash[0], err = array.Bytes()
	if err != nil {
		return nil, err
	}
	bytesToHash[1] = []byte(common.Separator)
	bytesToHash[2] = p.InnerProduct.Bytes()
	raw, err := asn1.MarshalStd(bytesToHash)
	if err != nil {
		return nil, err
	}
	// compute first challenge
	x := p.Curve.HashToZr(raw)

	// compute a commitment to inner product value and the vectors
	C := p.Q.Mul(p.Curve.ModMul(x, p.InnerProduct, p.Curve.GroupOrder))
	C.Add(p.Commitment)

	X := p.Q.Mul(x)
	// reduce the left and right vectors into one value each, left and right
	// LArray and RArray contain commitments to intermediate vectors
	left, right, LArray, RArray, err := p.reduce(X, C)
	if err != nil {
		return nil, err
	}
	return &IPA{Left: left, Right: right, R: RArray, L: LArray}, nil
}

// reduce returns two values left and right such that left is a function
// of the left vector and right is a function of right vector.
// Both vectors are committed in com which is passed as a parameter to reduce
func (p *ipaProver) reduce(X, com *mathlib.G1) (*mathlib.Zr, *mathlib.Zr, []*mathlib.G1, []*mathlib.G1, error) {
	leftGen, rightGen := cloneGenerators(p.LeftGenerators, p.RightGenerators)

	left := p.leftVector
	right := p.rightVector

	LArray := make([]*mathlib.G1, p.NumberOfRounds)
	RArray := make([]*mathlib.G1, p.NumberOfRounds)
	for i := range p.NumberOfRounds {
		// in each round the size of the vector is reduced by 2
		n := len(leftGen) / 2
		leftIP := InnerProduct(left[:n], right[n:], p.Curve)
		rightIP := InnerProduct(left[n:], right[:n], p.Curve)
		// LArray[i] is a commitment to left[:n], right[n:] and their inner product
		LArray[i] = commitVectorPlusOne(left[:n], right[n:], leftGen[n:], rightGen[:n], leftIP, X, p.Curve)
		// LArray[i].Add(X.Mul(leftIP))

		// RArray[i] is a commitment to left[n:], right[:n] and their inner product
		RArray[i] = commitVectorPlusOne(left[n:], right[:n], leftGen[:n], rightGen[n:], rightIP, X, p.Curve)
		// RArray[i].Add(X.Mul(rightIP))

		// compute this round's challenge x
		array := common.GetG1Array([]*mathlib.G1{LArray[i], RArray[i]})
		bytesToHash, err := array.Bytes()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		x := p.Curve.HashToZr(bytesToHash)

		// compute 1/x
		xInv := x.Copy()
		xInv.InvModOrder()

		// reduce the generators by 1/2, as a function of the old generators and x and 1/x
		leftGen, rightGen = reduceGenerators(leftGen, rightGen, x, xInv)

		// reduce the vectors by 1/2, a function of the old vectors and x and 1/x
		left, right = reduceVectors(left, right, x, xInv, p.Curve)

		xSquare := p.Curve.ModMul(x, x, p.Curve.GroupOrder)
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModOrder()

		// compute the commitment to left, right and their inner product
		CPrime := LArray[i].Mul2(xSquare, RArray[i], xSquareInv)
		CPrime.Add(com)
		// com = L^{x^2}*com*R^{1/x^2}
		com = CPrime
	}
	return left[0], right[0], LArray, RArray, nil
}

// ipaVerifier manages the verification of an inner product argument proof.
type ipaVerifier struct {
	// InnerProduct is the value being verified.
	InnerProduct *mathlib.Zr
	// Q is an auxiliary generator.
	Q *mathlib.G1
	// RightGenerators are the generators for the right vector.
	RightGenerators []*mathlib.G1
	// LeftGenerators are the generators for the left vector.
	LeftGenerators []*mathlib.G1
	// Commitment is the Pedersen commitment to the vectors.
	Commitment *mathlib.G1
	// NumberOfRounds is log2 of the vector length.
	NumberOfRounds uint64
	// Curve is the mathematical curve.
	Curve *mathlib.Curve
}

// NewIPAVerifier returns an ipaVerifier instance.
func NewIPAVerifier(
	innerProduct *mathlib.Zr,
	Q *mathlib.G1,
	leftGens, rightGens []*mathlib.G1,
	Commitment *mathlib.G1,
	rounds uint64,
	c *mathlib.Curve,
) *ipaVerifier {
	return &ipaVerifier{
		InnerProduct:    innerProduct,
		RightGenerators: rightGens,
		LeftGenerators:  leftGens,
		Curve:           c,
		NumberOfRounds:  rounds,
		Commitment:      Commitment,
		Q:               Q,
	}
}

// Verify checks if the provided inner product argument proof is valid.
func (v *ipaVerifier) Verify(proof *IPA) error {
	// check that the proof is well-formed
	if proof.Left == nil || proof.Right == nil {
		return errors.New("invalid IPA proof: nil elements")
	}
	if len(proof.L) != len(proof.R) || uint64(len(proof.L)) != v.NumberOfRounds {
		return errors.New("invalid IPA proof")
	}
	// compute the first challenge x
	array := common.GetG1Array(v.RightGenerators, v.LeftGenerators, []*mathlib.G1{v.Q, v.Commitment})
	bytesToHash := make([][]byte, 3)
	var err error
	bytesToHash[0], err = array.Bytes()
	if err != nil {
		return err
	}
	bytesToHash[1] = []byte(common.Separator)
	bytesToHash[2] = v.InnerProduct.Bytes()
	raw, err := asn1.MarshalStd(bytesToHash)
	if err != nil {
		return err
	}
	x := v.Curve.HashToZr(raw)
	// C is commitment to leftVector, rightVector and their inner product
	C := v.Q.Mul(v.Curve.ModMul(x, v.InnerProduct, v.Curve.GroupOrder))
	C.Add(v.Commitment)

	X := v.Q.Mul(x)

	leftGen, rightGen := cloneGenerators(v.LeftGenerators, v.RightGenerators)
	xSquareList := make([]*mathlib.Zr, v.NumberOfRounds)
	xSquareInvList := make([]*mathlib.Zr, v.NumberOfRounds)
	xList := make([]*mathlib.Zr, v.NumberOfRounds)

	// Verifier will not fold the generators in each round, and will instead
	// compute the challenge determined folded generators in the final round.
	// See: Page 17, Section 3, https://eprint.iacr.org/2017/1066.pdf
	for i := range v.NumberOfRounds {
		// check well-formedness
		if proof.L[i] == nil || proof.R[i] == nil {
			return errors.New("invalid IPA proof: nil elements")
		}
		// compute the challenge x for each round of reduction
		array = common.GetG1Array([]*mathlib.G1{proof.L[i], proof.R[i]})
		raw, err = array.Bytes()
		if err != nil {
			return err
		}
		x = v.Curve.HashToZr(raw)
		xList[i] = x.Copy()
		// 1/x
		xInv := x.Copy()
		xInv.InvModOrder()

		// x^2
		xSquare := v.Curve.ModMul(x, x, v.Curve.GroupOrder)
		xSquareList[i] = xSquare.Copy()
		xSquareList[i].Neg()
		// 1/x^2
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModOrder()
		xSquareInvList[i] = xSquareInv.Copy()
		xSquareInvList[i].Neg()
	}
	// Prepare final MSM to compute folded generators
	// - generators: leftGen||rightGen||proof.L||proof.R||X
	// - scalars:    proof.Left.s || proof.Right.s^{-1} || xsquareInvList || xSqureList || proof.Left * proof.Right
	generators := make([]*mathlib.G1, len(leftGen)+len(rightGen)+len(proof.L)+len(proof.R)+1)
	scalars := make([]*mathlib.Zr, len(generators))
	s, sInv := computeSVector(1<<v.NumberOfRounds, xList, v.Curve)
	for i := 0; i < len(s); i++ {
		s[i] = v.Curve.ModMul(s[i], proof.Left, v.Curve.GroupOrder)
		sInv[i] = v.Curve.ModMul(sInv[i], proof.Right, v.Curve.GroupOrder)
	}
	idx := 0
	copy(generators[idx:], leftGen)
	copy(scalars[idx:], s)
	idx += len(leftGen)

	copy(generators[idx:], rightGen)
	copy(scalars[idx:], sInv)
	idx += len(rightGen)

	copy(generators[idx:], proof.L)
	copy(scalars[idx:], xSquareList)
	idx += len(proof.L)

	copy(generators[idx:], proof.R)
	copy(scalars[idx:], xSquareInvList)
	idx += len(proof.R)

	generators[idx] = X.Copy()
	scalars[idx] = v.Curve.ModMul(proof.Left, proof.Right, v.Curve.GroupOrder)
	CPrime := v.Curve.MultiScalarMul(generators, scalars)
	if !CPrime.Equals(C) {
		return errors.New("invalid IPA")
	}
	return nil
}

// reduceVectors reduces the size of the vectors passed in the parameters by 1/2,
// as a function of the old vectors, x and 1/x
func reduceVectors(left, right []*mathlib.Zr, x, xInv *mathlib.Zr, c *mathlib.Curve) ([]*mathlib.Zr, []*mathlib.Zr) {
	l := len(left) / 2
	leftPrime := make([]*mathlib.Zr, l)
	rightPrime := make([]*mathlib.Zr, l)
	for i := 0; i < l; i++ {
		// a_i = a_ix + a_{i+len(left)/2}x^{-1}
		leftPrime[i] = c.ModAddMul2(left[i], x, left[i+l], xInv, c.GroupOrder)

		// b_i = b_ix^{-1} + b_{i+len(right)/2}x
		rightPrime[i] = c.ModAddMul2(right[i], xInv, right[i+l], x, c.GroupOrder)
	}

	return leftPrime, rightPrime
}

// reduceGenerators reduces the number of generators passed in the parameters by 1/2,
// as a function of the old generators,  x and 1/x
func reduceGenerators(leftGen, rightGen []*mathlib.G1, x, xInv *mathlib.Zr) ([]*mathlib.G1, []*mathlib.G1) {
	l := len(leftGen) / 2
	for i := 0; i < l; i++ {
		// G_i = G_i^{x_inv}*G_{i+len(left)/2}^x
		leftGen[i].Mul2InPlace(xInv, leftGen[i+l], x)
		// H_i = H_i^{x}*H_{i+len(right)/2}^{x_inv}
		rightGen[i].Mul2InPlace(x, rightGen[i+l], xInv)
	}
	return leftGen[:l], rightGen[:l]
}

func InnerProduct(left []*mathlib.Zr, right []*mathlib.Zr, c *mathlib.Curve) *mathlib.Zr {
	return c.ModAddMul(left, right, c.GroupOrder)
}

func commitVector(
	left []*mathlib.Zr,
	right []*mathlib.Zr,
	leftgen []*mathlib.G1,
	rightgen []*mathlib.G1,
	c *mathlib.Curve,
) *mathlib.G1 {
	points := make([]*mathlib.G1, len(leftgen)+len(rightgen))
	copy(points, leftgen)
	copy(points[len(leftgen):], rightgen)

	scalars := make([]*mathlib.Zr, len(left)+len(right))
	copy(scalars, left)
	copy(scalars[len(left):], right)

	return c.MultiScalarMul(points, scalars)
}

func commitVectorPlusOne(
	left []*mathlib.Zr,
	right []*mathlib.Zr,
	leftgen []*mathlib.G1,
	rightgen []*mathlib.G1,
	a *mathlib.Zr,
	b *mathlib.G1,
	c *mathlib.Curve,
) *mathlib.G1 {
	points := make([]*mathlib.G1, len(leftgen)+len(rightgen)+1)
	copy(points, leftgen)
	copy(points[len(leftgen):], rightgen)
	points[len(points)-1] = b

	scalars := make([]*mathlib.Zr, len(left)+len(right)+1)
	copy(scalars, left)
	copy(scalars[len(left):], right)
	scalars[len(scalars)-1] = a

	return c.MultiScalarMul(points, scalars)
}

func cloneGenerators(LeftGenerators, RightGenerators []*mathlib.G1) ([]*mathlib.G1, []*mathlib.G1) {
	leftGen := make([]*mathlib.G1, len(LeftGenerators))
	for i := 0; i < len(LeftGenerators); i++ {
		leftGen[i] = LeftGenerators[i].Copy()
	}
	rightGen := make([]*mathlib.G1, len(RightGenerators))
	for i := 0; i < len(RightGenerators); i++ {
		rightGen[i] = RightGenerators[i].Copy()
	}
	return leftGen, rightGen
}

// computeSVector computes the s vector and its entry-wise inverse of s as detailed below:
// b(i,j) = 1 if (logn - j)^{th} bit of i-1 is 1, else its -1
// s[i] = \prod_{j=1}^{\log n} x_j^{bits(i,j)}
// sInv[i] = s[i].Inverse
// Input: n, challenges = [x_1,\ldots,x_{\log n}]
// Returns (s, sInv) where sInv[i] = s[i]^{-1}, computed via batch inversion.
func computeSVector(n int, challenges []*mathlib.Zr, curve *mathlib.Curve) ([]*mathlib.Zr, []*mathlib.Zr) {
	log2n := len(challenges)

	// Verify n is consistent with number of challenges
	if 1<<log2n != n {
		panic("n must equal 2^(number of challenges)")
	}

	// Precompute challenge inverses (log2n inversions instead of O(n*log2n))
	challengeInvs := BatchInverse(challenges, curve)

	s := make([]*mathlib.Zr, n)

	for i := 1; i <= n; i++ {
		// Start with s_i = 1
		si := curve.NewZrFromInt(1)

		// Compute product over j=1 to log2(n).
		// At round j the generator fold splits on bit (log2n - j) of the
		// 0-based index: first half (bit=0) picks up x_j^{-1}, second
		// half (bit=1) picks up x_j — matching reduceGenerators.
		for j := 1; j <= log2n; j++ {
			bitPosition := log2n - j
			iMinus1 := i - 1
			bitIsSet := (iMinus1 >> bitPosition) & 1

			var factor *mathlib.Zr
			if bitIsSet == 1 {
				// second half at round j => x_j
				factor = challenges[j-1]
			} else {
				// first half at round j => x_j^{-1}
				factor = challengeInvs[j-1]
			}

			// Multiply: s_i *= factor
			si = curve.ModMul(si, factor, curve.GroupOrder)
		}

		s[i-1] = si
	}
	sInv := BatchInverse(s, curve)
	return s, sInv
}

// BatchInverse computes the entry-wise modular inverse of elems using
// Montgomery's trick: a single InvModOrder call plus O(n) multiplications.
// todo! Perhaps this can be added to mathlib.
func BatchInverse(elems []*mathlib.Zr, curve *mathlib.Curve) []*mathlib.Zr {
	n := len(elems)
	if n == 0 {
		return nil
	}

	inv := make([]*mathlib.Zr, n)

	// Forward pass: build prefix products
	// prefixProd[i] = elems[0] * elems[1] * ... * elems[i]
	prefixProd := make([]*mathlib.Zr, n)
	prefixProd[0] = elems[0].Copy()
	for i := 1; i < n; i++ {
		prefixProd[i] = curve.ModMul(prefixProd[i-1], elems[i], curve.GroupOrder)
	}

	// Single inversion of the total product
	acc := prefixProd[n-1].Copy()
	acc.InvModOrder()

	// Backward pass: extract individual inverses
	for i := n - 1; i >= 1; i-- {
		inv[i] = curve.ModMul(prefixProd[i-1], acc, curve.GroupOrder)
		acc = curve.ModMul(acc, elems[i], curve.GroupOrder)
	}
	inv[0] = acc

	return inv
}
