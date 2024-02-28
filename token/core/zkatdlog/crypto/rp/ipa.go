/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// IPA contains the proof that the inner product argument prover
// sends to the verifier
type IPA struct {
	// Left is the result of the reduction protocol of the left vector
	Left *mathlib.Zr
	// Right is the result of the reduction protocol of the right vector
	Right *mathlib.Zr
	// L is an array that contains commitments to the intermediary values
	// of the reduction protocol. The size of L is logarithmic in the size
	// of the left/right vector
	L []*mathlib.G1
	// R is an array that contains commitments to the intermediary values
	// of the reduction protocol. The size of R is logarithmic in the size
	// of the left/right vector
	R []*mathlib.G1
}

// ipaProver is the prover for the inner product argument. It shows that a
// value c is the inner product of two committed vectors a = (a_1, ..., a_n)
// (leftVector) and b = (b_1, ..., b_n) (rightVector)
type ipaProver struct {
	// rightVector is one of the committed vectors in Commitment
	rightVector []*mathlib.Zr
	// leftVector is one of the committed vectors in Commitment
	leftVector []*mathlib.Zr
	// InnerProduct is the inner product of leftVector and rightVector
	InnerProduct *mathlib.Zr
	// Q is a random generators of G1
	Q *mathlib.G1
	// RightGenerators are the generators used to commit to rightVector
	RightGenerators []*mathlib.G1
	// LeftGenerators are the generators used to commit to leftVector
	LeftGenerators []*mathlib.G1
	// Commitment is a Pedersen commitment to leftVector and rightVector
	Commitment *mathlib.G1
	// NumberOfRounds is the number of rounds in the reduction protocol.
	// It corresponds to log_2(len(rightVector)) = log_2(len(leftVector))
	NumberOfRounds int
	// Curve is the curve over which the computations are performed
	Curve *mathlib.Curve
}

// NewIPAProver returns an ipaProver as a function of the passed arguments
func NewIPAProver(
	innerProduct *mathlib.Zr,
	leftVector, rightVector []*mathlib.Zr,
	Q *mathlib.G1,
	leftGens, rightGens []*mathlib.G1,
	Commitment *mathlib.G1,
	rounds int,
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

// ipaVerifier verifies given a proof that a value c is the inner
// product of two vectors committed in Commitment
type ipaVerifier struct {
	// InnerProduct is the value against which the proof is verified
	InnerProduct *mathlib.Zr
	// Q is a random generators of G1
	Q *mathlib.G1
	// RightGenerators are the generators used to commit to rightVector
	RightGenerators []*mathlib.G1
	// LeftGenerators are the generators used to commit to leftVector
	LeftGenerators []*mathlib.G1
	// Commitment is a Pedersen commitment to leftVector and rightVector
	Commitment *mathlib.G1
	// NumberOfRounds is the number of rounds in the reduction protocol.
	// It corresponds to log_2(len(rightVector)) = log_2(len(leftVector))
	NumberOfRounds int
	// Curve is the curve over which the computations are performed
	Curve *mathlib.Curve
}

// NewIPAVerifier returns an ipaVerifier as a function of the passed arguments
func NewIPAVerifier(innerProduct *mathlib.Zr, Q *mathlib.G1, leftGens, rightGens []*mathlib.G1, Commitment *mathlib.G1, rounds int, c *mathlib.Curve) *ipaVerifier {
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

// Prove returns an IPA proof if no error occurs, else, it returns an error
func (p *ipaProver) Prove() (*IPA, error) {
	array := common.GetG1Array(p.RightGenerators, p.LeftGenerators, []*mathlib.G1{p.Q, p.Commitment})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return nil, err
	}
	bytesToHash = append(bytesToHash, []byte(common.Separator)...)
	bytesToHash = append(bytesToHash, p.InnerProduct.Bytes()...)
	// compute first challenge
	x := p.Curve.HashToZr(bytesToHash)
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

// Verify checks if the proof passed as a parameter is a valid inner
// product argument
func (v *ipaVerifier) Verify(proof *IPA) error {
	// check that the proof is well-formed
	if proof.Left == nil || proof.Right == nil {
		return errors.New("invalid IPA proof: nil elements")
	}
	if len(proof.L) != len(proof.R) || len(proof.L) != int(v.NumberOfRounds) {

		return errors.New("invalid IPA proof")
	}
	// compute the first challenge x
	array := common.GetG1Array(v.RightGenerators, v.LeftGenerators, []*mathlib.G1{v.Q, v.Commitment})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return err
	}
	bytesToHash = append(bytesToHash, []byte(common.Separator)...)
	bytesToHash = append(bytesToHash, v.InnerProduct.Bytes()...)
	x := v.Curve.HashToZr(bytesToHash)
	// C is commitment to leftVector, rightVector and their inner product
	C := v.Q.Mul(v.Curve.ModMul(x, v.InnerProduct, v.Curve.GroupOrder))
	C.Add(v.Commitment)

	X := v.Q.Mul(x)

	var leftGen []*mathlib.G1
	var rightGen []*mathlib.G1
	leftGen = append(leftGen, v.LeftGenerators...)
	rightGen = append(rightGen, v.RightGenerators...)
	for i := 0; i < int(v.NumberOfRounds); i++ {
		// check well-formedness
		if proof.L[i] == nil || proof.R[i] == nil {
			return errors.New("invalid IPA proof: nil elements")
		}
		// compute the challenge x for each round of reduction
		array = common.GetG1Array([]*mathlib.G1{proof.L[i], proof.R[i]})
		bytesToHash, err = array.Bytes()
		if err != nil {
			return err
		}
		x = v.Curve.HashToZr(bytesToHash)
		// 1/x
		xInv := x.Copy()
		xInv.InvModP(v.Curve.GroupOrder)

		// x^2
		xSquare := v.Curve.ModMul(x, x, v.Curve.GroupOrder)
		// 1/x^2
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModP(v.Curve.GroupOrder)
		// compute a commitment to the reduced vectors and their inner product
		CPrime := proof.L[i].Mul(xSquare)
		CPrime.Add(C)
		CPrime.Add(proof.R[i].Mul(xSquareInv))
		C = CPrime.Copy()
		// reduce the generators by 1/2, as a function of the old generators and x and 1/x
		leftGen, rightGen = reduceGenerators(leftGen, rightGen, x, xInv)
	}
	// compute a commitment to left, right and their product
	CPrime := leftGen[0].Mul(proof.Left)
	CPrime.Add(rightGen[0].Mul(proof.Right))
	CPrime.Add(X.Mul(v.Curve.ModMul(proof.Left, proof.Right, v.Curve.GroupOrder)))
	if !CPrime.Equals(C) {
		return errors.New("invalid IPA")
	}
	return nil

}

// reduce returns two values left and right such that left is a function
// of the left vector and right is a function of right vector.
// Both vectors are committed in com which is passed as a parameter to reduce
func (p *ipaProver) reduce(X, com *mathlib.G1) (*mathlib.Zr, *mathlib.Zr, []*mathlib.G1, []*mathlib.G1, error) {
	var LArray, RArray []*mathlib.G1
	var leftGen, rightGen []*mathlib.G1
	var left, right []*mathlib.Zr

	leftGen = append(leftGen, p.LeftGenerators...)
	rightGen = append(rightGen, p.RightGenerators...)
	left = append(left, p.leftVector...)
	right = append(right, p.rightVector...)

	for i := 0; i < int(p.NumberOfRounds); i++ {
		// in each round the size of the vector is reduced by 2
		n := len(leftGen) / 2
		leftIP := innerProduct(left[:n], right[n:], p.Curve)
		rightIP := innerProduct(left[n:], right[:n], p.Curve)
		// L is a commitment to left[:n], right[n:] and their inner product
		L := commitVector(left[:n], right[n:], leftGen[n:], rightGen[:n], p.Curve)
		L.Add(X.Mul(leftIP))
		LArray = append(LArray, L)

		// R is a commitment to left[n:], right[:n] and their inner product
		R := commitVector(left[n:], right[:n], leftGen[:n], rightGen[n:], p.Curve)
		R.Add(X.Mul(rightIP))
		RArray = append(RArray, R)

		// compute this round's challenge x
		array := common.GetG1Array([]*mathlib.G1{L, R})
		bytesToHash, err := array.Bytes()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		x := p.Curve.HashToZr(bytesToHash)

		// compute 1/x
		xInv := x.Copy()
		xInv.InvModP(p.Curve.GroupOrder)

		// reduce the generators by 1/2, as a function of the old generators and x and 1/x
		leftGen, rightGen = reduceGenerators(leftGen, rightGen, x, xInv)

		// reduce the vectors by 1/2, a function of the old vectors and x and 1/x
		left, right = reduceVectors(left, right, x, xInv, p.Curve)

		xSquare := p.Curve.ModMul(x, x, p.Curve.GroupOrder)
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModP(p.Curve.GroupOrder)

		// compute the commitment to left, right and their inner product
		CPrime := L.Mul(xSquare)
		CPrime.Add(com)
		CPrime.Add(R.Mul(xSquareInv))
		// com = L^{x^2}*com*R^{1/x^2}
		com = CPrime.Copy()
	}
	return left[0], right[0], LArray, RArray, nil
}

// reduceVectors reduces the size of the vectors passed in the parameters by 1/2,
// as a function of the old vectors, x and 1/x
func reduceVectors(left, right []*mathlib.Zr, x, xInv *mathlib.Zr, c *mathlib.Curve) ([]*mathlib.Zr, []*mathlib.Zr) {
	leftPrime := make([]*mathlib.Zr, len(left)/2)
	rightPrime := make([]*mathlib.Zr, len(right)/2)
	for i := 0; i < len(leftPrime); i++ {
		// a_i = a_ix + a_{i+len(left)/2}x^{-1}
		leftPrime[i] = c.ModMul(left[i], x, c.GroupOrder)
		leftPrime[i] = c.ModAdd(leftPrime[i], c.ModMul(left[i+len(leftPrime)], xInv, c.GroupOrder), c.GroupOrder)

		// b_i = b_ix^{-1} + b_{i+len(right)/2}x
		rightPrime[i] = c.ModMul(right[i], xInv, c.GroupOrder)
		rightPrime[i] = c.ModAdd(rightPrime[i], c.ModMul(right[i+len(rightPrime)], x, c.GroupOrder), c.GroupOrder)
	}
	return leftPrime, rightPrime
}

// reduceGenerators reduces the number of generators passed in the parameters by 1/2,
// as a function of the old generators,  x and 1/x
func reduceGenerators(leftGen, rightGen []*mathlib.G1, x, xInv *mathlib.Zr) ([]*mathlib.G1, []*mathlib.G1) {
	leftGenPrime := make([]*mathlib.G1, len(leftGen)/2)
	rightGenPrime := make([]*mathlib.G1, len(rightGen)/2)
	for i := 0; i < len(leftGenPrime); i++ {
		// G_i = G_i^x*G_{i+len(left)/2}^{1/x}
		leftGenPrime[i] = leftGen[i].Mul(xInv)
		leftGenPrime[i].Add(leftGen[i+len(leftGenPrime)].Mul(x))

		// H_i = H_i^{1/x}*H_{i+len(right)/2}^{x}
		rightGenPrime[i] = rightGen[i].Mul(x)
		rightGenPrime[i].Add(rightGen[i+len(rightGenPrime)].Mul(xInv))
	}
	return leftGenPrime, rightGenPrime
}

func innerProduct(left []*mathlib.Zr, right []*mathlib.Zr, c *mathlib.Curve) *mathlib.Zr {
	ip := c.NewZrFromInt(0)
	for i, l := range left {
		ip = c.ModAdd(ip, c.ModMul(l, right[i], c.GroupOrder), c.GroupOrder)
	}
	return ip
}

func commitVector(left []*mathlib.Zr, right []*mathlib.Zr, leftgen []*mathlib.G1, rightgen []*mathlib.G1, c *mathlib.Curve) *mathlib.G1 {
	com := c.NewG1()
	for i := range left {
		com.Add(leftgen[i].Mul(left[i]))
		com.Add(rightgen[i].Mul(right[i]))
	}
	return com
}
