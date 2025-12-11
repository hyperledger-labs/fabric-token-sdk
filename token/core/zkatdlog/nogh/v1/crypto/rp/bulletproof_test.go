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

type bfSetup struct {
	com       *math.G1
	Q         *math.G1
	P         *math.G1
	H         *math.G1
	G         *math.G1
	bf        *math.Zr
	leftGens  []*math.G1
	rightGens []*math.G1
	nr        uint64
	l         uint64
	curve     *math.Curve
}

func NewBfSetup(curveID math.CurveID) (*bfSetup, error) {
	curve := math.Curves[curveID]
	l := uint64(64)
	nr := 63 - uint64(bits.LeadingZeros64(l))
	leftGens := make([]*math.G1, l)
	rightGens := make([]*math.G1, l)

	rand, err := curve.Rand()
	if err != nil {
		return nil, err
	}

	Q := curve.GenG1.Mul(curve.NewRandomZr(rand))
	P := curve.GenG1.Mul(curve.NewRandomZr(rand))
	H := curve.GenG1.Mul(curve.NewRandomZr(rand))
	G := curve.GenG1.Mul(curve.NewRandomZr(rand))
	for i := 0; i < len(leftGens); i++ {
		leftGens[i] = curve.HashToG1([]byte(strconv.Itoa(2 * i)))
		rightGens[i] = curve.HashToG1([]byte(strconv.Itoa(2*i + 1)))
	}
	bf := curve.NewRandomZr(rand)
	com := G.Mul(curve.NewZrFromInt(115))
	com.Add(H.Mul(bf))

	return &bfSetup{
		com:       com,
		Q:         Q,
		P:         P,
		H:         H,
		G:         G,
		bf:        bf,
		leftGens:  leftGens,
		rightGens: rightGens,
		nr:        nr,
		l:         l,
		curve:     curve,
	}, nil
}

func TestBFProofVerify(t *testing.T) {
	setup, err := NewBfSetup(math.BLS12_381_BBS_GURVY)
	require.NoError(t, err)

	prover := rp.NewRangeProver(
		setup.com,
		115,
		[]*math.G1{setup.G, setup.H},
		setup.bf,
		setup.leftGens,
		setup.rightGens,
		setup.P,
		setup.Q,
		setup.nr,
		setup.l,
		setup.curve,
	)
	proof, err := prover.Prove()
	assert.NoError(t, err)
	assert.NotNil(t, proof)

	verifier := rp.NewRangeVerifier(
		setup.com,
		[]*math.G1{setup.G, setup.H},
		setup.leftGens,
		setup.rightGens,
		setup.P,
		setup.Q,
		setup.nr,
		setup.l,
		setup.curve,
	)
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func BenchmarkBFProver(b *testing.B) {
	pp, err := profile.New(profile.WithAll(), profile.WithPath("./profile"))
	require.NoError(b, err)
	require.NoError(b, pp.Start())
	defer pp.Stop()
	envs := make([]*bfSetup, 0, 128)
	for i := 0; i < 128; i++ {
		setup, err := NewBfSetup(math.BLS12_381_BBS_GURVY)
		require.NoError(b, err)
		envs = append(envs, setup)
	}

	b.Run("bench", func(b *testing.B) {
		for b.Loop() {
			setup := envs[rand.Intn(len(envs))]
			prover := rp.NewRangeProver(
				setup.com,
				115,
				[]*math.G1{setup.G, setup.H},
				setup.bf,
				setup.leftGens,
				setup.rightGens,
				setup.P,
				setup.Q,
				setup.nr,
				setup.l,
				setup.curve,
			)
			proof, err := prover.Prove()
			assert.NoError(b, err)
			assert.NotNil(b, proof)
		}
	})
}
