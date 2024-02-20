/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp

import (
	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto/common"
	"github.com/pkg/errors"
)

// RangeProof proves that a committed value < max
type RangeProof struct {
	T1           *math.G1
	T2           *math.G1
	Tau          *math.Zr
	Delta        *math.Zr
	C            *math.G1
	D            *math.G1
	InnerProduct *math.Zr
	IPA          *IPA
}

// rangeProver proves that a committed value < 2^BitLength.
type rangeProver struct {
	// value is the value committed in Commitment
	value uint64
	// blindingFactor is the randomness used to compute Commitment
	blindingFactor *math.Zr
	// Commitment is a hiding Pedersen commitment to value: Commitment = G^vH^r
	Commitment *math.G1
	// CommitmentGenerators are the generators (G, H) used to compute Commitment
	CommitmentGenerators []*math.G1
	// LeftGenerators are the generators that will be used to commit to
	// the bits (b_0,..., b_{BitLength-1}) of value
	LeftGenerators []*math.G1
	// RightGenerators are the generators that will be used to commit to (b_i-1)
	RightGenerators []*math.G1
	// P is a random generator of G1
	P *math.G1
	// Q is a random generator of G1
	Q *math.G1
	// NumberOfRounds correspond to log_2(BitLength). It corresponds to the
	// number of rounds of the reduction protocol
	NumberOfRounds int
	// BitLength is the size of the binary representation of value
	BitLength int
	// Curve is the curve over which the computation is performed
	Curve *math.Curve
}

func NewRangeProver(com *math.G1, value uint64, commitmentGen []*math.G1, blindingFactor *math.Zr, leftGen []*math.G1, rightGen []*math.G1, P, Q *math.G1, numberOfRounds, bitlength int, curve *math.Curve) *rangeProver {
	return &rangeProver{
		Commitment:           com,
		value:                value,
		CommitmentGenerators: commitmentGen,
		blindingFactor:       blindingFactor,
		LeftGenerators:       leftGen,
		RightGenerators:      rightGen,
		P:                    P,
		Q:                    Q,
		NumberOfRounds:       numberOfRounds,
		BitLength:            bitlength,
		Curve:                curve,
	}

}

// rangeVerifier verifies that a committed value < 2^BitLength.
type rangeVerifier struct {
	// Commitment is a hiding Pedersen commitment to value: Commitment = G^vH^r
	Commitment *math.G1
	// CommitmentGenerators are the generators (G, H) used to compute Commitment
	CommitmentGenerators []*math.G1
	// LeftGenerators are the generators that will be used to commit to
	// the bits (b_0,..., b_{BitLength-1}) of value
	LeftGenerators []*math.G1
	// RightGenerators are the generators that will be used to commit to (b_i-1)
	RightGenerators []*math.G1
	// P is a random generator of G1
	P *math.G1
	// Q is a random generator of G1
	Q *math.G1
	// NumberOfRounds correspond to log_2(BitLength). It corresponds to the
	// number of rounds of the reduction protocol
	NumberOfRounds int
	// BitLength is the size of the binary representation of value
	BitLength int
	// Curve is the curve over which the computation is performed
	Curve *math.Curve
}

func NewRangeVerifier(com *math.G1, commitmentGen []*math.G1, leftGen []*math.G1, rightGen []*math.G1, P, Q *math.G1, numberOfRounds, bitlength int, curve *math.Curve) *rangeVerifier {
	return &rangeVerifier{
		Commitment:           com,
		CommitmentGenerators: commitmentGen,
		LeftGenerators:       leftGen,
		RightGenerators:      rightGen,
		P:                    P,
		Q:                    Q,
		NumberOfRounds:       numberOfRounds,
		BitLength:            bitlength,
		Curve:                curve,
	}

}

func (p *rangeProver) Prove() (*RangeProof, error) {
	left, right, y, rp, err := p.preprocess()
	if err != nil {
		return nil, err
	}
	invy := y.Copy()
	invy.InvModP(p.Curve.GroupOrder)

	rightGeneratorsPrime := make([]*math.G1, len(p.RightGenerators))
	for i := 0; i < len(p.RightGenerators); i++ {
		invy2i := invy.PowMod(p.Curve.NewZrFromInt(int64(i)))
		rightGeneratorsPrime[i] = p.RightGenerators[i].Mul(invy2i)
	}

	com := commitVector(left, right, p.LeftGenerators, rightGeneratorsPrime, p.Curve)
	rp.InnerProduct = innerProduct(left, right, p.Curve)
	ipp := NewIPAProver(rp.InnerProduct, left, right, p.Q, p.LeftGenerators, rightGeneratorsPrime, com, p.NumberOfRounds, p.Curve)
	rp.IPA, err = ipp.Prove()
	if err != nil {
		return nil, err
	}

	return rp, nil
}

func (v *rangeVerifier) Verify(rp *RangeProof) error {
	// check that the proof is well-formed
	if rp.InnerProduct == nil || rp.C == nil || rp.D == nil {
		return errors.New("invalid range proof: nil elements")
	}
	if rp.T1 == nil || rp.T2 == nil {
		return errors.New("invalid range proof: nil elements")
	}
	if rp.Tau == nil || rp.Delta == nil {
		return errors.New("invalid range proof: nil elements")
	}
	if rp.IPA == nil {
		return errors.New("invalid range proof: nil elements")
	}
	array := common.GetG1Array([]*math.G1{rp.T1, rp.T2})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return err
	}
	x := v.Curve.HashToZr(bytesToHash)
	xSquare := x.PowMod(v.Curve.NewZrFromInt(2))

	array = common.GetG1Array([]*math.G1{rp.C, rp.D})
	bytesToHash, err = array.Bytes()
	if err != nil {
		return err
	}
	y := v.Curve.HashToZr(bytesToHash)
	z := v.Curve.HashToZr(y.Bytes())

	zSquare := z.PowMod(v.Curve.NewZrFromInt(2))
	zCube := v.Curve.ModMul(zSquare, z, v.Curve.GroupOrder)

	powy := make([]*math.Zr, len(v.RightGenerators))
	ipy := v.Curve.NewZrFromInt(0)
	ip2 := v.Curve.NewZrFromInt(0)
	for i := 0; i < len(powy); i++ {
		powy[i] = y.PowMod(v.Curve.NewZrFromInt(int64(i)))
		power2 := v.Curve.NewZrFromInt(2).PowMod(v.Curve.NewZrFromInt(int64(i)))
		ipy = v.Curve.ModAdd(ipy, powy[i], v.Curve.GroupOrder)
		ip2 = v.Curve.ModAdd(ip2, power2, v.Curve.GroupOrder)
	}
	polEval := v.Curve.ModSub(z, zSquare, v.Curve.GroupOrder)
	polEval = v.Curve.ModMul(polEval, ipy, v.Curve.GroupOrder)
	zCube = v.Curve.ModMul(zCube, ip2, v.Curve.GroupOrder)

	polEval = v.Curve.ModSub(polEval, zCube, v.Curve.GroupOrder)

	com := v.CommitmentGenerators[0].Mul(rp.InnerProduct)
	com.Add(v.CommitmentGenerators[1].Mul(rp.Tau))
	com.Sub(rp.T1.Mul(x))
	com.Sub(rp.T2.Mul(xSquare))

	comPrime := v.Commitment.Mul(zSquare)
	comPrime.Add(v.CommitmentGenerators[0].Mul(polEval))

	if !com.Equals(comPrime) {
		return errors.New("invalid range proof")
	}

	err = v.verifyIPA(rp, x, powy, z, zSquare)
	if err != nil {
		return err
	}

	return nil
}

func (p *rangeProver) preprocess() ([]*math.Zr, []*math.Zr, *math.Zr, *RangeProof, error) {
	var left, right []*math.Zr
	var randomLeft, randomRight []*math.Zr
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	rho := p.Curve.NewRandomZr(rand)
	eta := p.Curve.NewRandomZr(rand)
	for i := 0; i < p.BitLength; i++ {
		b := 1 << uint(i) & p.value
		if b > 0 {
			b = 1
		}
		left = append(left, p.Curve.NewZrFromInt(int64(b)))
		right = append(right, p.Curve.ModSub(left[i], p.Curve.NewZrFromInt(1), p.Curve.GroupOrder))

		randomLeft = append(randomLeft, p.Curve.NewRandomZr(rand))
		randomRight = append(randomRight, p.Curve.NewRandomZr(rand))
	}

	C := commitVector(left, right, p.LeftGenerators, p.RightGenerators, p.Curve)
	C.Add(p.P.Mul(rho))

	D := commitVector(randomLeft, randomRight, p.LeftGenerators, p.RightGenerators, p.Curve)
	D.Add(p.P.Mul(eta))

	array := common.GetG1Array([]*math.G1{C, D})
	bytesToHash, err := array.Bytes()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	y := p.Curve.HashToZr(bytesToHash)
	z := p.Curve.HashToZr(y.Bytes())

	leftPrime := make([]*math.Zr, len(left))
	rightPrime := make([]*math.Zr, len(right))

	randRightPrime := make([]*math.Zr, len(randomRight))

	zPrime := make([]*math.Zr, len(left))
	zSquare := z.PowMod(p.Curve.NewZrFromInt(2))

	for i := 0; i < len(left); i++ {
		leftPrime[i] = p.Curve.ModSub(left[i], z, p.Curve.GroupOrder)

		rightPrime[i] = p.Curve.ModAdd(right[i], z, p.Curve.GroupOrder)
		y2i := y.PowMod(p.Curve.NewZrFromInt(int64(i)))
		rightPrime[i] = p.Curve.ModMul(rightPrime[i], y2i, p.Curve.GroupOrder)

		randRightPrime[i] = p.Curve.ModMul(randomRight[i], y2i, p.Curve.GroupOrder)

		zPrime[i] = p.Curve.ModMul(zSquare, p.Curve.NewZrFromInt(2).PowMod(p.Curve.NewZrFromInt(int64(i))), p.Curve.GroupOrder)
	}

	t1 := innerProduct(leftPrime, randRightPrime, p.Curve)
	t1 = p.Curve.ModAdd(t1, innerProduct(rightPrime, randomLeft, p.Curve), p.Curve.GroupOrder)
	t1 = p.Curve.ModAdd(t1, innerProduct(zPrime, randomLeft, p.Curve), p.Curve.GroupOrder)
	tau1 := p.Curve.NewRandomZr(rand)
	T1 := p.CommitmentGenerators[0].Mul(t1)
	T1.Add(p.CommitmentGenerators[1].Mul(tau1))

	t2 := innerProduct(randomLeft, randRightPrime, p.Curve)
	tau2 := p.Curve.NewRandomZr(rand)
	T2 := p.CommitmentGenerators[0].Mul(t2)
	T2.Add(p.CommitmentGenerators[1].Mul(tau2))

	array = common.GetG1Array([]*math.G1{T1, T2})
	bytesToHash, err = array.Bytes()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	x := p.Curve.HashToZr(bytesToHash)

	for i := 0; i < len(left); i++ {
		left[i] = p.Curve.ModAdd(leftPrime[i], p.Curve.ModMul(x, randomLeft[i], p.Curve.GroupOrder), p.Curve.GroupOrder)
		right[i] = p.Curve.ModAdd(rightPrime[i], p.Curve.ModMul(x, randRightPrime[i], p.Curve.GroupOrder), p.Curve.GroupOrder)
		right[i] = p.Curve.ModAdd(right[i], zPrime[i], p.Curve.GroupOrder)
	}

	tau := p.Curve.ModMul(x, tau1, p.Curve.GroupOrder)
	tau = p.Curve.ModAdd(tau, p.Curve.ModMul(tau2, x.PowMod(p.Curve.NewZrFromInt(2)), p.Curve.GroupOrder), p.Curve.GroupOrder)
	tau = p.Curve.ModAdd(tau, p.Curve.ModMul(zSquare, p.blindingFactor, p.Curve.GroupOrder), p.Curve.GroupOrder)

	delta := p.Curve.ModAdd(rho, p.Curve.ModMul(eta, x, p.Curve.GroupOrder), p.Curve.GroupOrder)

	rp := &RangeProof{
		T1:    T1,
		T2:    T2,
		C:     C,
		D:     D,
		Tau:   tau,
		Delta: delta,
	}

	return left, right, y, rp, nil
}

func (v *rangeVerifier) verifyIPA(rp *RangeProof, x *math.Zr, powy []*math.Zr, z, zSquare *math.Zr) error {

	com := rp.D.Mul(x)
	com.Add(rp.C)

	rightGeneratorsPrime := make([]*math.G1, len(v.RightGenerators))
	for i := 0; i < len(v.LeftGenerators); i++ {
		com.Sub(v.LeftGenerators[i].Mul(z))

		invy2i := powy[i].Copy()
		invy2i.InvModP(v.Curve.GroupOrder)

		zi := v.Curve.ModMul(z, powy[i], v.Curve.GroupOrder)
		zi = v.Curve.ModAdd(zi, v.Curve.ModMul(zSquare, v.Curve.NewZrFromInt(2).PowMod(v.Curve.NewZrFromInt(int64(i))), v.Curve.GroupOrder), v.Curve.GroupOrder)

		rightGeneratorsPrime[i] = v.RightGenerators[i].Mul(invy2i)
		com.Add(rightGeneratorsPrime[i].Mul(zi))
	}
	com.Sub(v.P.Mul(rp.Delta))

	ipv := NewIPAVerifier(rp.InnerProduct, v.Q, v.LeftGenerators, rightGeneratorsPrime, com, v.NumberOfRounds, v.Curve)
	err := ipv.Verify(rp.IPA)
	if err != nil {
		return err
	}
	return nil
}
