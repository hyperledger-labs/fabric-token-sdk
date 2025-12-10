/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rp_test

import (
	"math/bits"
	"math/rand"
	"strconv"
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-smart-client/node/start/profile"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/rp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ipaSetup struct {
	left      []*math.Zr
	right     []*math.Zr
	Q         *math.G1
	leftGens  []*math.G1
	rightGens []*math.G1
	curve     *math.Curve
	com       *math.G1
	nr        uint64
}

func NewIpaSetup(curveID math.CurveID) (*ipaSetup, error) {
	curve := math.Curves[curveID]
	l := uint64(64)
	nr := 63 - uint64(bits.LeadingZeros64(l))
	leftGens := make([]*math.G1, l)
	rightGens := make([]*math.G1, l)
	left := make([]*math.Zr, l)
	right := make([]*math.Zr, l)
	rand, err := curve.Rand()
	if err != nil {
		return nil, err
	}
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
	return &ipaSetup{
		left:      left,
		right:     right,
		Q:         Q,
		leftGens:  leftGens,
		rightGens: rightGens,
		curve:     curve,
		com:       com,
		nr:        nr,
	}, nil
}

func TestIPAProofVerify(t *testing.T) {
	setup, err := NewIpaSetup(math.BLS12_381_BBS_GURVY)
	require.NoError(t, err)

	prover := rp.NewIPAProver(
		rp.InnerProduct(setup.left, setup.right, setup.curve),
		setup.left,
		setup.right,
		setup.Q,
		setup.leftGens,
		setup.rightGens,
		setup.com,
		setup.nr,
		setup.curve,
	)
	proof, err := prover.Prove()
	require.NoError(t, err)
	assert.NotNil(t, proof)

	verifier := rp.NewIPAVerifier(
		rp.InnerProduct(setup.left, setup.right, setup.curve),
		setup.Q,
		setup.leftGens,
		setup.rightGens,
		setup.com,
		setup.nr,
		setup.curve,
	)
	err = verifier.Verify(proof)
	require.NoError(t, err)
}

func BenchmarkIPAProver(b *testing.B) {
	pp, err := profile.New(profile.WithAll(), profile.WithPath("./profile"))
	require.NoError(b, err)
	require.NoError(b, pp.Start())
	defer pp.Stop()
	envs := make([]*ipaSetup, 0, 128)
	for i := 0; i < 128; i++ {
		setup, err := NewIpaSetup(math.BLS12_381_BBS_GURVY)
		require.NoError(b, err)
		envs = append(envs, setup)
	}

	b.Run("bench", func(b *testing.B) {
		for b.Loop() {
			setup := envs[rand.Intn(len(envs))]
			prover := rp.NewIPAProver(
				rp.InnerProduct(setup.left, setup.right, setup.curve),
				setup.left,
				setup.right,
				setup.Q,
				setup.leftGens,
				setup.rightGens,
				setup.com,
				setup.nr,
				setup.curve,
			)
			proof, err := prover.Prove()
			require.NoError(b, err)
			assert.NotNil(b, proof)
		}
	})
}
