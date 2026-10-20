/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package prover_test

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/stretchr/testify/require"

	"github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/crypto/edwards"
	"github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/prover"
)

func babyProofResult(t *testing.T, value uint64) prover.ProofResult {
	t.Helper()
	c := edwards.BabyJubjub()
	rcv, err := c.RandomScalar()
	require.NoError(t, err)
	cv, err := c.ValueCommit(value, rcv)
	require.NoError(t, err)

	var cm fr.Element
	_, err = cm.SetRandom()
	require.NoError(t, err)

	cmB, cvx, cvy, rcvB := cm.Bytes(), cv.X.Bytes(), cv.Y.Bytes(), rcv.Bytes()

	return prover.ProofResult{Commitment: cmB[:], ValueCommitmentX: cvx[:], ValueCommitmentY: cvy[:], RCV: rcvB[:]}
}

// TestBindingSignatureBabyJubjub exercises the curve-generic binding path on BN254.
func TestBindingSignatureBabyJubjub(t *testing.T) {
	c := edwards.BabyJubjub()
	tc := make([]byte, 32)

	in := []prover.ProofResult{babyProofResult(t, 60), babyProofResult(t, 40)}
	out := []prover.ProofResult{babyProofResult(t, 100)}
	sig, err := prover.ComputeBindingSignatureFor(c, "transfer", tc, in, out)
	require.NoError(t, err)
	bvk, err := prover.ComputeBVKFor(c, in, out, 0)
	require.NoError(t, err)
	require.NoError(t, c.Verify(bvk, prover.ComputeActionHash("transfer", tc, in, out), sig, c.R))

	// Issuance: no inputs, valueBalance = Σv_out.
	iss := []prover.ProofResult{babyProofResult(t, 500)}
	isig, err := prover.ComputeBindingSignatureFor(c, "issue", tc, nil, iss)
	require.NoError(t, err)
	ibvk, err := prover.ComputeBVKFor(c, nil, iss, 500)
	require.NoError(t, err)
	require.NoError(t, c.Verify(ibvk, prover.ComputeActionHash("issue", tc, nil, iss), isig, c.R))

	// Wrong balance must not verify.
	badBvk, err := prover.ComputeBVKFor(c, nil, iss, 499)
	require.NoError(t, err)
	require.ErrorIs(t, c.Verify(badBvk, prover.ComputeActionHash("issue", tc, nil, iss), isig, c.R), edwards.ErrInvalidSignature)

	// Malformed RCV is rejected.
	bad := []prover.ProofResult{{RCV: []byte{1}}}
	_, err = prover.ComputeBindingSignatureFor(c, "issue", tc, nil, bad)
	require.Error(t, err)
}
