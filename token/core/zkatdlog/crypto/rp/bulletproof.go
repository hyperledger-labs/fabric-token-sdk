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
	// T1 is a Pedersen commitment to a random tau1
	T1 *math.G1
	// T2 is a Pedersen commitment a random tau2
	T2 *math.G1
	// Tau = tau1x + tau2x^2 for a random challenge x
	Tau *math.Zr
	// C is a hiding Pedersen commitment to vectors left = (b_0, ...,b_{n-1})
	// and right = (b_0 - 1, ..., b_{n-1}-1) with randomness rho, where
	// v = \sum_{i=0}^{n-1} b_i 2^i, and n = BitLength
	C *math.G1
	// D is a hiding Pedersen commitment to two random vectors
	// with randomness eta
	D *math.G1
	// Delta = rho + x eta
	Delta *math.Zr
	// InnerProduct is the value of the inner product of the vectors committed in the non-hiding
	// commitment C*D^x/P^Delta
	InnerProduct *math.Zr
	// IPA is the proof that shows that InnerProduct is correct
	IPA *IPA
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

// NewRangeProver returns a rangeProver based on  the passed arguments
func NewRangeProver(
	com *math.G1,
	value uint64,
	commitmentGen []*math.G1,
	blindingFactor *math.Zr,
	leftGen []*math.G1,
	rightGen []*math.G1,
	P, Q *math.G1,
	numberOfRounds, bitLength int,
	curve *math.Curve,
) *rangeProver {
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
		BitLength:            bitLength,
		Curve:                curve,
	}

}

// rangeVerifier verifies that a committed value < 2^BitLength.
type rangeVerifier struct {
	// Commitment is a hiding Pedersen commitment to value: Commitment = G^vH^r
	Commitment *math.G1
	// CommitmentGenerators are the generators (G, H) used to compute Commitment
	CommitmentGenerators []*math.G1
	// LeftGenerators are the generators (G_0, ..., G_{BitLength}) that will be used to commit to
	// the bits (b_0,..., b_{BitLength-1}) of value
	LeftGenerators []*math.G1
	// RightGenerators are the generators (H_0, ..., H_{BitLength}) that will be used to commit to (b_i-1)
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

// NewRangeVerifier returns a rangeVerifier based on the passed arguments
func NewRangeVerifier(
	com *math.G1,
	commitmentGen []*math.G1,
	leftGen []*math.G1,
	rightGen []*math.G1,
	P, Q *math.G1,
	numberOfRounds, bitLength int,
	curve *math.Curve,
) *rangeVerifier {
	return &rangeVerifier{
		Commitment:           com,
		CommitmentGenerators: commitmentGen,
		LeftGenerators:       leftGen,
		RightGenerators:      rightGen,
		P:                    P,
		Q:                    Q,
		NumberOfRounds:       numberOfRounds,
		BitLength:            bitLength,
		Curve:                curve,
	}

}

// Prove produces a RangeProof that shows that a committed value
// v = \sum_{i=0}^{BitLength} b_i 2^i; b_i in {0, 1}
func (p *rangeProver) Prove() (*RangeProof, error) {
	// left = (b_i-z) + xU_i
	// right = y^i((b_i-1+z)+xV_i)+2^iz^2

	left, right, y, rp, err := p.preprocess()
	if err != nil {
		return nil, err
	}
	// compute 1/y
	yInv := y.Copy()
	yInv.InvModP(p.Curve.GroupOrder)

	rightGeneratorsPrime := make([]*math.G1, len(p.RightGenerators))
	for i := 0; i < len(p.RightGenerators); i++ {
		// compute 1/y^i
		yInv2i := yInv.PowMod(p.Curve.NewZrFromInt(int64(i)))
		// compute the new generators H'_i = H_i^{1/y^i}
		rightGeneratorsPrime[i] = p.RightGenerators[i].Mul(yInv2i)
	}
	// compute the commitment to left and right
	com := commitVector(left, right, p.LeftGenerators, rightGeneratorsPrime, p.Curve)
	rp.InnerProduct = innerProduct(left, right, p.Curve)
	// produce the IPA
	ipp := NewIPAProver(
		rp.InnerProduct,
		left,
		right,
		p.Q,
		p.LeftGenerators,
		rightGeneratorsPrime,
		com,
		p.NumberOfRounds,
		p.Curve,
	)
	rp.IPA, err = ipp.Prove()
	if err != nil {
		return nil, err
	}

	return rp, nil
}

// Verify enable a rangeVerifier to checks the validity of a RangeProof
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
	// compute x and x^2
	x := v.Curve.HashToZr(bytesToHash)
	xSquare := x.PowMod(v.Curve.NewZrFromInt(2))

	// compute y and z
	array = common.GetG1Array([]*math.G1{rp.C, rp.D, v.Commitment})
	bytesToHash, err = array.Bytes()
	if err != nil {
		return err
	}
	y := v.Curve.HashToZr(bytesToHash)
	z := v.Curve.HashToZr(y.Bytes())
	// z^2 and z^3
	zSquare := z.PowMod(v.Curve.NewZrFromInt(2))
	zCube := v.Curve.ModMul(zSquare, z, v.Curve.GroupOrder)

	yPow := make([]*math.Zr, len(v.RightGenerators))
	ipy := v.Curve.NewZrFromInt(0)
	ip2 := v.Curve.NewZrFromInt(0)
	// 2^i
	var power2 *math.Zr
	for i := 0; i < len(yPow); i++ {
		// y^i
		if i == 0 {
			yPow[0] = v.Curve.NewZrFromInt(1)
			power2 = v.Curve.NewZrFromInt(1)
		} else {
			yPow[i] = v.Curve.ModMul(y, yPow[i-1], v.Curve.GroupOrder)
			power2 = v.Curve.ModMul(v.Curve.NewZrFromInt(2), power2, v.Curve.GroupOrder)
		}
		// ipy = \sum y^i
		ipy = v.Curve.ModAdd(ipy, yPow[i], v.Curve.GroupOrder)
		// ip2 = sum 2^i
		ip2 = v.Curve.ModAdd(ip2, power2, v.Curve.GroupOrder)
	}
	// polEval = (z -z^)\sum y^i - z^3\sum 2^i
	polEval := v.Curve.ModSub(z, zSquare, v.Curve.GroupOrder)
	polEval = v.Curve.ModMul(polEval, ipy, v.Curve.GroupOrder)
	zCube = v.Curve.ModMul(zCube, ip2, v.Curve.GroupOrder)

	polEval = v.Curve.ModSub(polEval, zCube, v.Curve.GroupOrder)

	// com is should be equal to v.Commitment^{z^2} if p.Value falls within range
	com := v.CommitmentGenerators[0].Mul(rp.InnerProduct)
	com.Add(v.CommitmentGenerators[1].Mul(rp.Tau))
	com.Sub(rp.T1.Mul(x))
	com.Sub(rp.T2.Mul(xSquare))

	comPrime := v.Commitment.Mul(zSquare)
	comPrime.Add(v.CommitmentGenerators[0].Mul(polEval))

	if !com.Equals(comPrime) {
		return errors.New("invalid range proof")
	}

	// verify the IPA
	err = v.verifyIPA(rp, x, yPow, z, zSquare)
	if err != nil {
		return err
	}

	return nil
}

// preprocess prepares data for the inner product argument
func (p *rangeProver) preprocess() ([]*math.Zr, []*math.Zr, *math.Zr, *RangeProof, error) {
	var left, right []*math.Zr
	var randomLeft, randomRight []*math.Zr
	rand, err := p.Curve.Rand()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	rho := p.Curve.NewRandomZr(rand)
	eta := p.Curve.NewRandomZr(rand)
	for i := 0; i < int(p.BitLength); i++ {
		b := 1 << uint(i) & p.value
		if b > 0 {
			b = 1
		}
		// this is an array of the bits b_i of p.value
		left = append(left, p.Curve.NewZrFromInt(int64(b)))
		// this is an array of b_i - 1
		right = append(right, p.Curve.ModSub(left[i], p.Curve.NewZrFromInt(1), p.Curve.GroupOrder))
		// these are randomly generated arrays
		randomLeft = append(randomLeft, p.Curve.NewRandomZr(rand))
		randomRight = append(randomRight, p.Curve.NewRandomZr(rand))
	}

	// C commits to L = (b_0, ..., b_{p.BitLength}) and R = (b_0 - 1 , ..., b_{p.BitLength} - 1)
	C := commitVector(left, right, p.LeftGenerators, p.RightGenerators, p.Curve)
	// C is a hiding commitment thanks to rho
	C.Add(p.P.Mul(rho))

	// D commits two random vectors U and V
	D := commitVector(randomLeft, randomRight, p.LeftGenerators, p.RightGenerators, p.Curve)
	// D is a hiding commitment thanks to eta
	D.Add(p.P.Mul(eta))

	array := common.GetG1Array([]*math.G1{C, D, p.Commitment})
	// compute challenges y and z
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
	// z^2
	zSquare := z.PowMod(p.Curve.NewZrFromInt(2))
	var y2i *math.Zr
	for i := 0; i < len(left); i++ {
		// compute L_i - z
		leftPrime[i] = p.Curve.ModSub(left[i], z, p.Curve.GroupOrder)
		// compute R_i + z
		rightPrime[i] = p.Curve.ModAdd(right[i], z, p.Curve.GroupOrder)
		// compute y^i
		if i == 0 {
			y2i = p.Curve.NewZrFromInt(1)
		} else {
			y2i = p.Curve.ModMul(y, y2i, p.Curve.GroupOrder)
		}
		// compute (R_i+z)y^i
		rightPrime[i] = p.Curve.ModMul(rightPrime[i], y2i, p.Curve.GroupOrder)
		// compute V_iy^i
		randRightPrime[i] = p.Curve.ModMul(randomRight[i], y2i, p.Curve.GroupOrder)
		// compute 2^iz^2
		zPrime[i] = p.Curve.ModMul(zSquare, p.Curve.NewZrFromInt(2).PowMod(p.Curve.NewZrFromInt(int64(i))), p.Curve.GroupOrder)
	}

	// compute \sum y^iV_i(L_i-z)
	t1 := innerProduct(leftPrime, randRightPrime, p.Curve)
	// compute \sum y^i(V_i(L_i-z) + (R_i +z)U_i)
	t1 = p.Curve.ModAdd(t1, innerProduct(rightPrime, randomLeft, p.Curve), p.Curve.GroupOrder)
	// compute \sum y^i(V_i(L_i-z) + (R_i+z)U_i) + U_i2^iz^2
	t1 = p.Curve.ModAdd(t1, innerProduct(zPrime, randomLeft, p.Curve), p.Curve.GroupOrder)
	// commit to t1
	tau1 := p.Curve.NewRandomZr(rand)
	T1 := p.CommitmentGenerators[0].Mul(t1)
	T1.Add(p.CommitmentGenerators[1].Mul(tau1))

	// compute = \sum y^iU_iV_i
	t2 := innerProduct(randomLeft, randRightPrime, p.Curve)
	// commit to t2
	tau2 := p.Curve.NewRandomZr(rand)
	T2 := p.CommitmentGenerators[0].Mul(t2)
	T2.Add(p.CommitmentGenerators[1].Mul(tau2))

	// compute challenge x
	array = common.GetG1Array([]*math.G1{T1, T2})
	bytesToHash, err = array.Bytes()
	if err != nil {
		return nil, nil, nil, nil, err
	}
	x := p.Curve.HashToZr(bytesToHash)

	// compute vectors left and right against which an IPA will be produced
	// if p.Value is within the authorized range, then L_iR_i =0 and L_i-R_i-1 = 0
	// the inner product <left, right> = p.Value*z^2+t1x+t2x^2+f(z, y)
	// f(z, y) = \sum (z-z^2)*y^i - z^3*2^i
	for i := 0; i < len(left); i++ {
		// compute (L_i-z) + xU_i
		left[i] = p.Curve.ModAdd(leftPrime[i], p.Curve.ModMul(x, randomLeft[i], p.Curve.GroupOrder), p.Curve.GroupOrder)
		// compute y^i((R_i+z)+xV_i)+2^iz^2
		right[i] = p.Curve.ModAdd(rightPrime[i], p.Curve.ModMul(x, randRightPrime[i], p.Curve.GroupOrder), p.Curve.GroupOrder)
		right[i] = p.Curve.ModAdd(right[i], zPrime[i], p.Curve.GroupOrder)
	}
	// tau = t1x + t2x^2 + z^2p.blindingFactor
	tau := p.Curve.ModMul(x, tau1, p.Curve.GroupOrder)
	tau = p.Curve.ModAdd(tau, p.Curve.ModMul(tau2, x.PowMod(p.Curve.NewZrFromInt(2)), p.Curve.GroupOrder), p.Curve.GroupOrder)
	tau = p.Curve.ModAdd(tau, p.Curve.ModMul(zSquare, p.blindingFactor, p.Curve.GroupOrder), p.Curve.GroupOrder)

	// delta = rho + eta*x
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

// verifyIPA checks if the IPA within the range proof is valid
func (v *rangeVerifier) verifyIPA(rp *RangeProof, x *math.Zr, yPow []*math.Zr, z, zSquare *math.Zr) error {
	// compute com the non-hiding commitment to the vectors for which the inner product is computed
	// C commits to vectors L and R whereas D commits to vectors U and V
	// with generators (G_0, ..., G_{BitLength}, H_0, ..., H_{BitLength}, P)

	// com commits to vector L' composed of elements L'_i = (L_i-z) + xU_i and
	// vector R' composed of elements R'i = y^i((R_i+z)+xV_i)+2^iz^2
	// with generators  (G_0, ..., G_{BitLength}, H'_0, ..., H'_{BitLength})
	com := rp.D.Mul(x)
	com.Add(rp.C)
	rightGeneratorsPrime := make([]*math.G1, len(v.RightGenerators))
	for i := 0; i < len(v.LeftGenerators); i++ {
		com.Sub(v.LeftGenerators[i].Mul(z))
		// 1/y^i
		yInv2i := yPow[i].Copy()
		yInv2i.InvModP(v.Curve.GroupOrder)
		// zy^i + z^2
		zi := v.Curve.ModMul(z, yPow[i], v.Curve.GroupOrder)
		zi = v.Curve.ModAdd(zi, v.Curve.ModMul(zSquare, v.Curve.NewZrFromInt(2).PowMod(v.Curve.NewZrFromInt(int64(i))), v.Curve.GroupOrder), v.Curve.GroupOrder)
		// recompute the generators H'_i = H_i^{1/y_i}
		rightGeneratorsPrime[i] = v.RightGenerators[i].Mul(yInv2i)
		com.Add(rightGeneratorsPrime[i].Mul(zi))
	}
	com.Sub(v.P.Mul(rp.Delta))

	// run the IPA verifier
	ipv := NewIPAVerifier(rp.InnerProduct, v.Q, v.LeftGenerators, rightGeneratorsPrime, com, v.NumberOfRounds, v.Curve)
	err := ipv.Verify(rp.IPA)
	if err != nil {
		return err
	}
	return nil
}
