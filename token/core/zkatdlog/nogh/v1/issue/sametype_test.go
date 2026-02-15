/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package issue_test

import (
	"testing"

	math "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/issue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSameTypeProof(t *testing.T) {
	prover, verifier := GetSameTypeProverAndVerifier(t)
	proof, err := prover.Prove()
	require.NoError(t, err)
	assert.NotNil(t, proof)
	err = verifier.Verify(proof)
	require.NoError(t, err)
}

func prepareTokens(t *testing.T, pp []*math.G1) []*math.G1 {
	t.Helper()
	curve := math.Curves[1]
	rand, err := curve.Rand()
	require.NoError(t, err)

	bf := make([]*math.Zr, 2)
	values := make([]uint64, 2)

	for i := range 2 {
		bf[i] = curve.NewRandomZr(rand)
	}
	values[0] = 100
	values[1] = 50

	tokens := make([]*math.G1, len(values))
	for i := range values {
		tokens[i] = NewToken(curve.NewZrFromInt(int64(values[i])), bf[i], "ABC", pp, curve) // #nosec G115
	}

	return tokens
}

func GetSameTypeProverAndVerifier(t *testing.T) (*issue.SameTypeProver, *issue.SameTypeVerifier) {
	t.Helper()
	pp := preparePedersenParameters(t)
	curve := math.Curves[1]

	rand, err := curve.Rand()
	require.NoError(t, err)
	blindingFactor := curve.NewRandomZr(rand)
	com := pp[0].Mul(curve.HashToZr([]byte("ABC")))
	com.Add(pp[2].Mul(blindingFactor))

	tokens := prepareTokens(t, pp)

	return issue.NewSameTypeProver("ABC", blindingFactor, com, pp, math.Curves[1]), issue.NewSameTypeVerifier(tokens, pp, math.Curves[1])
}

func TestSameTypeDeserializeError(t *testing.T) {
	st := &issue.SameType{}
	err := st.Deserialize([]byte("invalid"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to initialize unmarshaller")

	// Partial data
	curve := math.Curves[1]
	rand, _ := curve.Rand()
	raw, err := issue.NewSameTypeProver("ABC", curve.NewRandomZr(rand), curve.GenG1.Mul(curve.NewRandomZr(rand)), preparePedersenParameters(t), curve).Prove()
	assert.NoError(t, err)
	serialized, err := raw.Serialize()
	assert.NoError(t, err)

	for i := 0; i < len(serialized)-1; i++ {
		err = st.Deserialize(serialized[:i])
		assert.Error(t, err)
	}
}

func TestSameTypeVerifyError(t *testing.T) {
	prover, verifier := GetSameTypeProverAndVerifier(t)
	proof, err := prover.Prove()
	assert.NoError(t, err)

	// Wrong challenge
	curve := math.Curves[1]
	randReader, err := curve.Rand()
	assert.NoError(t, err)

	originalChallenge := proof.Challenge
	proof.Challenge = curve.NewRandomZr(randReader)
	err = verifier.Verify(proof)
	assert.EqualError(t, err, "invalid same type proof")
	proof.Challenge = originalChallenge

	// Wrong commitment to type
	originalCommitment := proof.CommitmentToType
	proof.CommitmentToType = curve.GenG1.Mul(curve.NewRandomZr(randReader))
	err = verifier.Verify(proof)
	assert.EqualError(t, err, "invalid same type proof")
	proof.CommitmentToType = originalCommitment
}

func preparePedersenParameters(t *testing.T) []*math.G1 {
	t.Helper()
	curve := math.Curves[1]
	rand, err := curve.Rand()
	require.NoError(t, err)

	pp := make([]*math.G1, 3)

	for i := range 3 {
		pp[i] = curve.GenG1.Mul(curve.NewRandomZr(rand))
	}

	return pp
}

func NewToken(value *math.Zr, rand *math.Zr, tokenType string, pp []*math.G1, curve *math.Curve) *math.G1 {
	token := curve.NewG1()
	token.Add(pp[0].Mul(curve.HashToZr([]byte(tokenType))))
	token.Add(pp[1].Mul(value))
	token.Add(pp[2].Mul(rand))

	return token
}
