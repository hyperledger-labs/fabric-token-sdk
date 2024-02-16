/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import "C"
import (
	mathlib "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// IPA contains the proof that the inner product argument prover sends to the verifier
type IPA struct {
	Left  *mathlib.Zr
	Right *mathlib.Zr
	L     []*mathlib.G1
	R     []*mathlib.G1
}

type ipaProver struct {
	InnerProduct    *mathlib.Zr
	rightVector     []*mathlib.Zr
	leftVector      []*mathlib.Zr
	Q               *mathlib.G1
	RightGenerators []*mathlib.G1
	LeftGenerators  []*mathlib.G1
	Commitment      *mathlib.G1
	NumberOfRounds  int
	Curve           *mathlib.Curve
}

func NewIPAProver(innerProduct *mathlib.Zr, leftVector, rightVector []*mathlib.Zr, Q *mathlib.G1, leftGens, rightGens []*mathlib.G1, Commitment *mathlib.G1, rounds int, c *mathlib.Curve) *ipaProver {
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

type ipaVerifier struct {
	InnerProduct    *mathlib.Zr
	Q               *mathlib.G1
	RightGenerators []*mathlib.G1
	LeftGenerators  []*mathlib.G1
	Commitment      *mathlib.G1
	NumberOfRounds  int
	Curve           *mathlib.Curve
}

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

func (p *ipaProver) Prove() (*IPA, error) {
	array := common.GetG1Array(p.RightGenerators, p.LeftGenerators, []*mathlib.G1{p.Q, p.Commitment})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return nil, err
	}
	bytesToHash = append(bytesToHash, []byte(common.Seperator)...)
	bytesToHash = append(bytesToHash, p.InnerProduct.Bytes()...)
	// compute first challenge
	x := p.Curve.HashToZr(bytesToHash)
	com := p.Q.Mul(p.Curve.ModMul(x, p.InnerProduct, p.Curve.GroupOrder))
	com.Add(p.Commitment)
	X := p.Q.Mul(x)
	left, right, LArray, RArray, err := p.reduce(X, com)
	if err != nil {
		return nil, err
	}
	return &IPA{Left: left[0], Right: right[0], R: RArray, L: LArray}, nil
}

func (v *ipaVerifier) Verify(proof *IPA) error {
	if len(proof.L) != len(proof.R) || len(proof.L) != v.NumberOfRounds {
		return errors.New("invalid proof")
	}
	array := common.GetG1Array(v.RightGenerators, v.LeftGenerators, []*mathlib.G1{v.Q, v.Commitment})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return err
	}
	bytesToHash = append(bytesToHash, []byte(common.Seperator)...)
	bytesToHash = append(bytesToHash, v.InnerProduct.Bytes()...)
	// compute first challenge
	x := v.Curve.HashToZr(bytesToHash)
	com := v.Q.Mul(v.Curve.ModMul(x, v.InnerProduct, v.Curve.GroupOrder))
	com.Add(v.Commitment)
	X := v.Q.Mul(x)

	var leftgen []*mathlib.G1
	var rightgen []*mathlib.G1
	leftgen = append(leftgen, v.LeftGenerators...)
	rightgen = append(rightgen, v.RightGenerators...)
	for i := 0; i < v.NumberOfRounds; i++ {
		array = common.GetG1Array([]*mathlib.G1{proof.L[i], proof.R[i]})
		bytesToHash, err = array.Bytes()
		if err != nil {
			return err
		}
		x = v.Curve.HashToZr(bytesToHash)
		invx := x.Copy()
		invx.InvModP(v.Curve.GroupOrder)

		xSquare := v.Curve.ModMul(x, x, v.Curve.GroupOrder)
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModP(v.Curve.GroupOrder)

		CPrime := proof.L[i].Mul(xSquare)
		CPrime.Add(com)
		CPrime.Add(proof.R[i].Mul(xSquareInv))
		com = CPrime.Copy()
		leftgen, rightgen = prepareGenerators(leftgen, rightgen, x, invx)
	}
	CPrime := leftgen[0].Mul(proof.Left)
	CPrime.Add(rightgen[0].Mul(proof.Right))
	CPrime.Add(X.Mul(v.Curve.ModMul(proof.Left, proof.Right, v.Curve.GroupOrder)))
	if !CPrime.Equals(com) {
		return errors.New("invalid IPA")
	}
	return nil

}

func (p *ipaProver) reduce(X, com *mathlib.G1) ([]*mathlib.Zr, []*mathlib.Zr, []*mathlib.G1, []*mathlib.G1, error) {
	var LArray, RArray []*mathlib.G1
	var leftgen, rightgen []*mathlib.G1
	var left, right []*mathlib.Zr

	leftgen = append(leftgen, p.LeftGenerators...)
	rightgen = append(rightgen, p.RightGenerators...)
	left = append(left, p.leftVector...)
	right = append(right, p.rightVector...)

	for i := 0; i < p.NumberOfRounds; i++ {
		n := len(leftgen) / 2
		leftIP := innerProduct(left[:n], right[n:], p.Curve)
		rightIP := innerProduct(left[n:], right[:n], p.Curve)

		L := commitVector(left[:n], right[n:], leftgen[n:], rightgen[:n], p.Curve)
		L.Add(X.Mul(leftIP))
		LArray = append(LArray, L)

		R := commitVector(left[n:], right[:n], leftgen[:n], rightgen[n:], p.Curve)
		R.Add(X.Mul(rightIP))
		RArray = append(RArray, R)

		array := common.GetG1Array([]*mathlib.G1{L, R})
		bytesToHash, err := array.Bytes()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		x := p.Curve.HashToZr(bytesToHash)

		invx := x.Copy()
		invx.InvModP(p.Curve.GroupOrder)

		leftgen, rightgen = prepareGenerators(leftgen, rightgen, x, invx)
		left, right = prepareVectors(left, right, x, invx, p.Curve)

		xSquare := p.Curve.ModMul(x, x, p.Curve.GroupOrder)
		xSquareInv := xSquare.Copy()
		xSquareInv.InvModP(p.Curve.GroupOrder)

		CPrime := L.Mul(xSquare)
		CPrime.Add(com)
		CPrime.Add(R.Mul(xSquareInv))
		com = CPrime.Copy()
	}
	return left, right, LArray, RArray, nil

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
	for i, _ := range left {
		com.Add(leftgen[i].Mul(left[i]))
		com.Add(rightgen[i].Mul(right[i]))
	}
	return com
}

func prepareVectors(left, right []*mathlib.Zr, x, invx *mathlib.Zr, c *mathlib.Curve) ([]*mathlib.Zr, []*mathlib.Zr) {
	leftPrime := make([]*mathlib.Zr, len(left)/2)
	rightPrime := make([]*mathlib.Zr, len(right)/2)
	for i := 0; i < len(leftPrime); i++ {
		leftPrime[i] = c.ModMul(left[i], x, c.GroupOrder)
		leftPrime[i] = c.ModAdd(leftPrime[i], c.ModMul(left[i+len(leftPrime)], invx, c.GroupOrder), c.GroupOrder)

		rightPrime[i] = c.ModMul(right[i], invx, c.GroupOrder)
		rightPrime[i] = c.ModAdd(rightPrime[i], c.ModMul(right[i+len(rightPrime)], x, c.GroupOrder), c.GroupOrder)
	}
	return leftPrime, rightPrime
}

func prepareGenerators(leftGen, rightGen []*mathlib.G1, x, invx *mathlib.Zr) ([]*mathlib.G1, []*mathlib.G1) {
	leftGenPrime := make([]*mathlib.G1, len(leftGen)/2)
	rightGenPrime := make([]*mathlib.G1, len(rightGen)/2)
	for i := 0; i < len(leftGenPrime); i++ {
		leftGenPrime[i] = leftGen[i].Mul(invx)
		leftGenPrime[i].Add(leftGen[i+len(leftGenPrime)].Mul(x))
		rightGenPrime[i] = rightGen[i].Mul(x)
		rightGenPrime[i].Add(rightGen[i+len(rightGenPrime)].Mul(invx))
	}
	return leftGenPrime, rightGenPrime
}
