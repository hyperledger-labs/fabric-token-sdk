/*
Copyright IBM Corp. All Rights Reserved.
SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	bls12381fr "github.com/consensys/gnark-crypto/ecc/bls12-381/fr"
	bn254fr "github.com/consensys/gnark-crypto/ecc/bn254/fr"
	"github.com/stretchr/testify/require"

	snarktoken "github.com/LFDT-Panurus/panurus/x/token/core/zkatsnark/token"
)

var supportedCurves = []ecc.ID{ecc.BLS12_381, ecc.BN254}

func TestNoteCommitmentDeterminism(t *testing.T) {
	for _, curve := range supportedCurves {
		t.Run(curve.String(), func(t *testing.T) {
			note, err := snarktoken.NewRandomNote(100, "USD", curve)
			require.NoError(t, err)

			cm1, err := note.Commitment(curve)
			require.NoError(t, err)

			cm2, err := note.Commitment(curve)
			require.NoError(t, err)

			require.Equal(t, cm1, cm2, "commitment must be deterministic on %s", curve)
		})
	}
}

func TestNoteCommitmentDeterminism_WideRandomness(t *testing.T) {
	// A curve-agnostic note with 64 bytes of randomness must be deterministic
	// across all supported curves.
	note, err := snarktoken.NewRandomNote(100, "USD")
	require.NoError(t, err)
	require.Len(t, note.Randomness, 64)

	for _, curve := range supportedCurves {
		t.Run(curve.String(), func(t *testing.T) {
			cm1, err := note.Commitment(curve)
			require.NoError(t, err)
			require.Len(t, cm1, 32)

			cm2, err := note.Commitment(curve)
			require.NoError(t, err)

			require.Equal(t, cm1, cm2, "wide randomness commitment must be deterministic on %s", curve)
		})
	}
}

func TestEncodeTokenType_Generic(t *testing.T) {
	// BLS12-381
	tBLS := snarktoken.EncodeTokenType[bls12381fr.Element]("USD")
	var inlineBLS bls12381fr.Element
	inlineBLS.SetBytes([]byte("USD"))
	require.Equal(t, inlineBLS, tBLS)

	bBLS := snarktoken.EncodeTokenTypeBytes[bls12381fr.Element]("USD")
	expectedBLS := inlineBLS.Bytes()
	require.Equal(t, expectedBLS[:], bBLS)

	// BN254
	tBN := snarktoken.EncodeTokenType[bn254fr.Element]("USD")
	var inlineBN bn254fr.Element
	inlineBN.SetBytes([]byte("USD"))
	require.Equal(t, inlineBN, tBN)

	bBN := snarktoken.EncodeTokenTypeBytes[bn254fr.Element]("USD")
	expectedBN := inlineBN.Bytes()
	require.Equal(t, expectedBN[:], bBN)
}

func TestNoteRandomnessUniqueness(t *testing.T) {
	for _, curve := range supportedCurves {
		t.Run(curve.String(), func(t *testing.T) {
			note1, err := snarktoken.NewRandomNote(100, "USD", curve)
			require.NoError(t, err)

			note2, err := snarktoken.NewRandomNote(100, "USD", curve)
			require.NoError(t, err)

			require.NotEqual(t, note1.Randomness, note2.Randomness, "randomness must be unique")

			cm1, err := note1.Commitment(curve)
			require.NoError(t, err)
			cm2, err := note2.Commitment(curve)
			require.NoError(t, err)

			require.NotEqual(t, cm1, cm2, "different randomness must produce different commitments on %s", curve)
		})
	}
}

func TestNewRandomNoteForField(t *testing.T) {
	// Generic note creation for BLS12-381
	noteBLS, err := snarktoken.NewRandomNoteForField[bls12381fr.Element](100, "USD")
	require.NoError(t, err)
	require.Len(t, noteBLS.Randomness, 32)
	cmBLS, err := noteBLS.Commitment(ecc.BLS12_381)
	require.NoError(t, err)
	require.Len(t, cmBLS, 32)

	// Generic note creation for BN254
	noteBN, err := snarktoken.NewRandomNoteForField[bn254fr.Element](100, "USD")
	require.NoError(t, err)
	require.Len(t, noteBN.Randomness, 32)
	cmBN, err := noteBN.Commitment(ecc.BN254)
	require.NoError(t, err)
	require.Len(t, cmBN, 32)
}

func TestNoteSerializeDeserialize_AllCurvesAndLengths(t *testing.T) {
	// Test notes with 32, 48, and 64 bytes of randomness across all curves
	lengths := []int{32, 48, 64}

	for _, curve := range supportedCurves {
		t.Run(curve.String(), func(t *testing.T) {
			for _, rLen := range lengths {
				rBytes, err := snarktoken.SampleRandomness(rLen)
				require.NoError(t, err)

				note := &snarktoken.Note{
					Value:      250,
					TokenType:  "EUR",
					Randomness: rBytes,
					Issuer:     []byte("org-issuer-1"),
				}

				serialized, err := note.Serialize()
				require.NoError(t, err)

				deserialized, err := snarktoken.Deserialize(serialized)
				require.NoError(t, err)

				require.Equal(t, note.Value, deserialized.Value)
				require.Equal(t, note.TokenType, deserialized.TokenType)
				require.Equal(t, note.Randomness, deserialized.Randomness)
				require.Equal(t, note.Issuer, deserialized.Issuer)

				cm1, err := note.Commitment(curve)
				require.NoError(t, err)
				cm2, err := deserialized.Commitment(curve)
				require.NoError(t, err)
				require.Equal(t, cm1, cm2, "deserialized note must have identical commitment on %s", curve)
			}
		})
	}
}

func TestNoteCommitmentCrossCurveDistinction(t *testing.T) {
	// Same value and token type must produce distinct commitments under BLS12-381 vs BN254
	note, err := snarktoken.NewRandomNote(42, "EUR")
	require.NoError(t, err)

	cmBLS, err := note.Commitment(ecc.BLS12_381)
	require.NoError(t, err)
	require.Len(t, cmBLS, 32)

	cmBN, err := note.Commitment(ecc.BN254)
	require.NoError(t, err)
	require.Len(t, cmBN, 32)

	require.NotEqual(t, cmBLS, cmBN, "BLS12-381 and BN254 commitments must differ")
}

func TestFieldElementFromBytes_WideAndCanonical(t *testing.T) {
	// 32-byte canonical
	rBLS, rBytesBLS, err := snarktoken.SampleFieldRandomness[bls12381fr.Element]()
	require.NoError(t, err)
	parsedBLS, err := snarktoken.FieldElementFromBytes[bls12381fr.Element](rBytesBLS)
	require.NoError(t, err)
	require.Equal(t, rBLS, parsedBLS)

	rBN, rBytesBN, err := snarktoken.SampleFieldRandomness[bn254fr.Element]()
	require.NoError(t, err)
	parsedBN, err := snarktoken.FieldElementFromBytes[bn254fr.Element](rBytesBN)
	require.NoError(t, err)
	require.Equal(t, rBN, parsedBN)

	// Wide 64-byte randomness
	wide, err := snarktoken.SampleRandomness(64)
	require.NoError(t, err)

	fBLS, err := snarktoken.FieldElementFromBytes[bls12381fr.Element](wide)
	require.NoError(t, err)
	var expectedBLS bls12381fr.Element
	expectedBLS.SetBytes(wide)
	require.Equal(t, expectedBLS, fBLS)

	fBN, err := snarktoken.FieldElementFromBytes[bn254fr.Element](wide)
	require.NoError(t, err)
	var expectedBN bn254fr.Element
	expectedBN.SetBytes(wide)
	require.Equal(t, expectedBN, fBN)

	// Empty bytes error
	_, err = snarktoken.FieldElementFromBytes[bls12381fr.Element](nil)
	require.Error(t, err)
}
