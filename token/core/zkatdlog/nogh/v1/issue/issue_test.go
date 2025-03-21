/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto"
	issue2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/stretchr/testify/assert"
)

func TestIssue(t *testing.T) {
	prover, verifier := prepareZKIssue(t)
	proof, err := prover.Prove()
	assert.NoError(t, err)
	assert.NotNil(t, proof)
	err = verifier.Verify(proof)
	assert.NoError(t, err)
}

func prepareZKIssue(t *testing.T) (*issue2.Prover, *issue2.Verifier) {
	pp, err := v1.Setup(32, nil, math.BN254)
	assert.NoError(t, err)
	tw, tokens := prepareInputsForZKIssue(pp)
	prover, err := issue2.NewProver(tw, tokens, pp)
	assert.NoError(t, err)
	verifier := issue2.NewVerifier(tokens, pp)
	return prover, verifier
}

func prepareInputsForZKIssue(pp *v1.PublicParams) ([]*token.TokenDataWitness, []*math.G1) {
	values := make([]uint64, 2)
	values[0] = 120
	values[1] = 190
	curve := math.Curves[pp.Curve]
	rand, _ := curve.Rand()
	bf := make([]*math.Zr, len(values))
	for i := 0; i < len(values); i++ {
		bf[i] = math.Curves[pp.Curve].NewRandomZr(rand)
	}

	tokens := make([]*math.G1, len(values))
	for i := 0; i < len(values); i++ {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp.PedersenGenerators, curve)
	}
	return token.NewTokenDataWitness("ABC", values, bf), tokens
}
