/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp_test

import (
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/common/crypto/math"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	"github.com/stretchr/testify/assert"
)

func TestIPAProofVerify(t *testing.T) {
	curve := math.Curves[math2.BLS12_381_BBS_GURVY_EXT]
	nr := uint64(6)
	l := uint64(1 << nr)
	leftGens := make([]*math.G1, l)
	rightGens := make([]*math.G1, l)
	left := make([]*math.Zr, l)
	right := make([]*math.Zr, l)
	rand, err := curve.Rand()
	assert.NoError(t, err)
	com := curve.NewG1()
	Q := curve.GenG1
	for i := 0; i < len(left); i++ {
		leftGens[i] = curve.HashToG1([]byte(strconv.Itoa(i)))
		rightGens[i] = curve.HashToG1([]byte(strconv.Itoa(i + 1)))
		left[i] = curve.NewRandomZr(rand)
		right[i] = curve.NewRandomZr(rand)
		com.Add(leftGens[i].Mul(left[i]))
		com.Add(rightGens[i].Mul(right[i]))
	}

	prover := rp.NewIPAProver(innerProduct(left, right, curve), left, right, Q, leftGens, rightGens, com, nr, curve)
	proof, err := prover.Prove()
	assert.NoError(t, err)
	assert.NotNil(t, proof)
	verifier := rp.NewIPAVerifier(innerProduct(left, right, curve), Q, leftGens, rightGens, com, nr, curve)
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func innerProduct(left []*math.Zr, right []*math.Zr, c *math.Curve) *math.Zr {
	ip := c.NewZrFromInt(0)
	for i, l := range left {
		ip = c.ModAdd(ip, c.ModMul(l, right[i], c.GroupOrder), c.GroupOrder)
	}
	return ip
}
