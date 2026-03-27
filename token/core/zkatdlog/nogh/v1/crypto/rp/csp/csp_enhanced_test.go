/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"testing"

	math "github.com/IBM/mathlib"
	math2 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/crypto/math"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSPWithZeroWitness verifies behavior when witness contains zero elements.
func TestCSPWithZeroWitness(t *testing.T) {
	curve := math.Curves[math.BN254]
	rounds := uint64(2)
	n := int(1 << rounds)

	rand, err := curve.Rand()
	require.NoError(t, err)

	generators := make([]*math.G1, n)
	witness := make([]*math.Zr, n)
	linearForm := make([]*math.Zr, n)

	for i := range n {
		generators[i] = curve.HashToG1([]byte{byte(i)})
		witness[i] = curve.NewZrFromInt(0) // All zeros
		linearForm[i] = curve.NewRandomZr(rand)
	}

	com := curve.MultiScalarMul(generators, witness)
	value := math2.InnerProduct(linearForm, witness, curve)

	prover := &cspProver{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
		witness:        witness,
	}

	proof, err := prover.Prove()
	require.NoError(t, err)

	verifier := &cspVerifier{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
	}

	err = verifier.Verify(proof)
	require.NoError(t, err)
}

// TestCSPWithMaxFieldElements verifies behavior with maximum field elements.
func TestCSPWithMaxFieldElements(t *testing.T) {
	curve := math.Curves[math.BN254]
	rounds := uint64(2)
	n := int(1 << rounds)

	generators := make([]*math.G1, n)
	witness := make([]*math.Zr, n)
	linearForm := make([]*math.Zr, n)

	// Use maximum field element (order - 1)
	maxElem := curve.NewZrFromBytes(curve.GroupOrder.Bytes())
	maxElem = curve.ModSub(maxElem, curve.NewZrFromInt(1), curve.GroupOrder)

	for i := range n {
		generators[i] = curve.HashToG1([]byte{byte(i)})
		witness[i] = maxElem.Copy()
		linearForm[i] = maxElem.Copy()
	}

	com := curve.MultiScalarMul(generators, witness)
	value := math2.InnerProduct(linearForm, witness, curve)

	prover := &cspProver{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
		witness:        witness,
	}

	proof, err := prover.Prove()
	require.NoError(t, err)

	verifier := &cspVerifier{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
	}

	err = verifier.Verify(proof)
	require.NoError(t, err)
}

// TestCSPNonPowerOfTwoSize verifies that non-power-of-2 sizes are rejected.
func TestCSPNonPowerOfTwoSize(t *testing.T) {
	curve := math.Curves[math.BN254]
	rounds := uint64(2)
	n := 5 // Not a power of 2

	rand, err := curve.Rand()
	require.NoError(t, err)

	generators := make([]*math.G1, n)
	witness := make([]*math.Zr, n)
	linearForm := make([]*math.Zr, n)

	for i := range n {
		generators[i] = curve.HashToG1([]byte{byte(i)})
		witness[i] = curve.NewRandomZr(rand)
		linearForm[i] = curve.NewRandomZr(rand)
	}

	com := curve.MultiScalarMul(generators, witness)
	value := math2.InnerProduct(linearForm, witness, curve)

	prover := &cspProver{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
		witness:        witness,
	}

	_, err = prover.Prove()
	require.Error(t, err)
	require.Contains(t, err.Error(), "2^NumberOfRounds")
}

// TestCSPEmptyVectors verifies behavior with empty vectors.
func TestCSPEmptyVectors(t *testing.T) {
	curve := math.Curves[math.BN254]

	prover := &cspProver{
		Commitment:     curve.GenG1,
		Generators:     []*math.G1{},
		LinearForm:     []*math.Zr{},
		Value:          curve.NewZrFromInt(0),
		NumberOfRounds: 0,
		Curve:          curve,
		witness:        []*math.Zr{},
	}

	_, err := prover.Prove()
	require.Error(t, err)
}

// TestCSPMismatchedGeneratorsWitness verifies error when generators and witness sizes differ.
func TestCSPMismatchedGeneratorsWitness(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	generators := make([]*math.G1, 4)
	witness := make([]*math.Zr, 3) // Mismatch
	linearForm := make([]*math.Zr, 4)

	for i := range 4 {
		generators[i] = curve.HashToG1([]byte{byte(i)})
		linearForm[i] = curve.NewRandomZr(rand)
	}
	for i := range 3 {
		witness[i] = curve.NewRandomZr(rand)
	}

	prover := &cspProver{
		Commitment:     curve.GenG1,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          curve.NewZrFromInt(0),
		NumberOfRounds: 2,
		Curve:          curve,
		witness:        witness,
	}

	_, err = prover.Prove()
	require.Error(t, err)
}

// TestCSPTamperedMultipleRounds verifies that tampering any round invalidates the proof.
func TestCSPTamperedMultipleRounds(t *testing.T) {
	curve := math.Curves[math.BN254]
	setup := newCSPSetup(t, curve, 3) // 8 elements, 3 rounds

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	// Test tampering each round
	for round := 0; round < 3; round++ {
		t.Run("round_"+string(rune('0'+round)), func(t *testing.T) {
			tamperedProof := &CSPProof{
				Left:   make([]*math.G1, len(proof.Left)),
				Right:  make([]*math.G1, len(proof.Right)),
				VLeft:  make([]*math.Zr, len(proof.VLeft)),
				VRight: make([]*math.Zr, len(proof.VRight)),
				Curve:  proof.Curve,
			}

			// Copy all rounds
			for i := range proof.Left {
				tamperedProof.Left[i] = proof.Left[i].Copy()
				tamperedProof.Right[i] = proof.Right[i].Copy()
				tamperedProof.VLeft[i] = proof.VLeft[i].Copy()
				tamperedProof.VRight[i] = proof.VRight[i].Copy()
			}

			// Tamper the specific round
			rand, err := curve.Rand()
			require.NoError(t, err)
			tamperedProof.Left[round] = curve.GenG1.Mul(curve.NewRandomZr(rand))

			err = setup.verifier.Verify(tamperedProof)
			require.Error(t, err)
		})
	}
}

// TestCSPSVectorProperties verifies mathematical properties of the s-vector.
func TestCSPSVectorProperties(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	k := 4
	n := 1 << k
	challenges := make([]*math.Zr, k)
	for i := range challenges {
		challenges[i] = curve.NewRandomZr(rand)
	}

	s := cspSVector(n, challenges, curve)

	// Property 1: s[0] should be 1
	assert.True(t, s[0].Equals(curve.NewZrFromInt(1)), "s[0] should be 1")

	// Property 2: s[i + 2^r] = s[i] * c_{k-1-r} for all valid i, r
	for r := 0; r < k; r++ {
		halfLen := 1 << r
		c := challenges[k-1-r]
		for i := 0; i < halfLen; i++ {
			expected := curve.ModMul(s[i], c, curve.GroupOrder)
			assert.True(t, s[i+halfLen].Equals(expected),
				"s[%d] should equal s[%d] * c[%d]", i+halfLen, i, k-1-r)
		}
	}
}

// TestCSPSVectorDifferentChallenges verifies s-vector changes with different challenges.
func TestCSPSVectorDifferentChallenges(t *testing.T) {
	curve := math.Curves[math.BN254]
	rand, err := curve.Rand()
	require.NoError(t, err)

	k := 3
	n := 1 << k

	challenges1 := make([]*math.Zr, k)
	challenges2 := make([]*math.Zr, k)
	for i := range k {
		challenges1[i] = curve.NewRandomZr(rand)
		challenges2[i] = curve.NewRandomZr(rand)
	}

	s1 := cspSVector(n, challenges1, curve)
	s2 := cspSVector(n, challenges2, curve)

	// Vectors should be different (except s[0] which is always 1)
	differentCount := 0
	for i := 1; i < n; i++ {
		if !s1[i].Equals(s2[i]) {
			differentCount++
		}
	}
	assert.Greater(t, differentCount, 0, "s-vectors with different challenges should differ")
}

// TestCSPVerifierRejectsWrongNumberOfRounds verifies various malformed proof structures.
func TestCSPVerifierRejectsWrongNumberOfRounds(t *testing.T) {
	curve := math.Curves[math.BN254]
	setup := newCSPSetup(t, curve, 2)

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	testCases := []struct {
		name        string
		modifyProof func(*CSPProof)
	}{
		{
			name: "extra_left",
			modifyProof: func(p *CSPProof) {
				p.Left = append(p.Left, curve.GenG1)
			},
		},
		{
			name: "missing_right",
			modifyProof: func(p *CSPProof) {
				p.Right = p.Right[:len(p.Right)-1]
			},
		},
		{
			name: "extra_vleft",
			modifyProof: func(p *CSPProof) {
				p.VLeft = append(p.VLeft, curve.NewZrFromInt(0))
			},
		},
		{
			name: "missing_vright",
			modifyProof: func(p *CSPProof) {
				p.VRight = p.VRight[:len(p.VRight)-1]
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tamperedProof := &CSPProof{
				Left:   append([]*math.G1{}, proof.Left...),
				Right:  append([]*math.G1{}, proof.Right...),
				VLeft:  append([]*math.Zr{}, proof.VLeft...),
				VRight: append([]*math.Zr{}, proof.VRight...),
				Curve:  proof.Curve,
			}
			tc.modifyProof(tamperedProof)

			err := setup.verifier.Verify(tamperedProof)
			require.Error(t, err)
			require.Contains(t, err.Error(), "wrong number of rounds")
		})
	}
}

// TestCSPWithIdentityGenerator verifies behavior when a generator is the identity element.
func TestCSPWithIdentityGenerator(t *testing.T) {
	curve := math.Curves[math.BN254]
	rounds := uint64(2)
	n := int(1 << rounds)

	rand, err := curve.Rand()
	require.NoError(t, err)

	generators := make([]*math.G1, n)
	witness := make([]*math.Zr, n)
	linearForm := make([]*math.Zr, n)

	for i := range n {
		if i == 0 {
			// Use identity element for first generator
			generators[i] = curve.NewG1()
		} else {
			generators[i] = curve.HashToG1([]byte{byte(i)})
		}
		witness[i] = curve.NewRandomZr(rand)
		linearForm[i] = curve.NewRandomZr(rand)
	}

	com := curve.MultiScalarMul(generators, witness)
	value := math2.InnerProduct(linearForm, witness, curve)

	prover := &cspProver{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
		witness:        witness,
	}

	proof, err := prover.Prove()
	require.NoError(t, err)

	verifier := &cspVerifier{
		Commitment:     com,
		Generators:     generators,
		LinearForm:     linearForm,
		Value:          value,
		NumberOfRounds: rounds,
		Curve:          curve,
	}

	err = verifier.Verify(proof)
	require.NoError(t, err)
}

// TestCSPLargeNumberOfRounds verifies CSP works with larger vector sizes.
func TestCSPLargeNumberOfRounds(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping large test in short mode")
	}

	curve := math.Curves[math.BN254]
	rounds := uint64(8) // 256 elements
	setup := newCSPSetup(t, curve, rounds)

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	err = setup.verifier.Verify(proof)
	require.NoError(t, err)
}

// TestCSPCommitmentMismatch verifies that commitment mismatch is detected.
func TestCSPCommitmentMismatch(t *testing.T) {
	curve := math.Curves[math.BN254]
	setup := newCSPSetup(t, curve, 2)

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	// Change commitment by adding a random point
	rand, err := curve.Rand()
	require.NoError(t, err)
	setup.verifier.Commitment = setup.verifier.Commitment.Copy()
	setup.verifier.Commitment.Add(curve.GenG1.Mul(curve.NewRandomZr(rand)))

	err = setup.verifier.Verify(proof)
	require.Error(t, err)
	require.Contains(t, err.Error(), "verification failed")
}

// TestCSPLinearFormMismatch verifies that linear form mismatch is detected.
func TestCSPLinearFormMismatch(t *testing.T) {
	curve := math.Curves[math.BN254]
	setup := newCSPSetup(t, curve, 2)

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	// Change one coefficient in the linear form
	rand, err := curve.Rand()
	require.NoError(t, err)
	setup.verifier.LinearForm[0] = curve.NewRandomZr(rand)

	err = setup.verifier.Verify(proof)
	require.Error(t, err)
}

// TestCSPGeneratorMismatch verifies that generator mismatch is detected.
func TestCSPGeneratorMismatch(t *testing.T) {
	curve := math.Curves[math.BN254]
	setup := newCSPSetup(t, curve, 2)

	proof, err := setup.prover.Prove()
	require.NoError(t, err)

	// Change one generator
	rand, err := curve.Rand()
	require.NoError(t, err)
	setup.verifier.Generators[0] = curve.GenG1.Mul(curve.NewRandomZr(rand))

	err = setup.verifier.Verify(proof)
	require.Error(t, err)
}
