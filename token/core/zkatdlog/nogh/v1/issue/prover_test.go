/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package issue

import (
	"testing"

	math "github.com/IBM/mathlib"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/setup"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/nogh/v1/token"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProverErrors(t *testing.T) {
	curve := math.Curves[math.BN254]
	pp, err := v1.Setup(32, nil, math.BN254)
	assert.NoError(t, err)
	randReader, _ := curve.Rand()

	// tw[i] is nil
	validMeta := &token.Metadata{Type: "ABC", BlindingFactor: curve.NewRandomZr(randReader), Value: curve.NewZrFromInt(100)}
	_, err = NewProver([]*token.Metadata{validMeta, nil}, []*math.G1{curve.GenG1, curve.GenG1}, pp)
	require.ErrorIs(t, err, ErrInvalidTokenWitness)

	// tw[i].BlindingFactor is nil
	_, err = NewProver([]*token.Metadata{validMeta, {Type: "ABC"}}, []*math.G1{curve.GenG1, curve.GenG1}, pp)
	require.ErrorIs(t, err, ErrInvalidTokenWitness)

	// tw[i].Value is nil or invalid for Uint()
	tw := &token.Metadata{
		Type:           "ABC",
		BlindingFactor: curve.NewRandomZr(randReader),
		Value:          curve.NewRandomZr(randReader), // Likely out of range for uint64
	}
	// Ensure it is out of range by setting a very large value if possible,
	// but NewRandomZr is usually large enough.
	// Actually, let's just use a value that we know will fail Uint() if we want to test that specific error.
	// But the previous run showed it already fails with NewRandomZr.

	_, err = NewProver([]*token.Metadata{validMeta, tw}, []*math.G1{curve.GenG1, curve.GenG1}, pp)
	require.ErrorIs(t, err, ErrInvalidTokenWitnessValues)
}
