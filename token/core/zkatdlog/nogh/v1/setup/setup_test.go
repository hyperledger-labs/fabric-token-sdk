/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package setup

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testingHelper is a test helper for creating test data
func testingHelper(t *testing.T) []byte {
	t.Helper()
	issuerPK, err := os.ReadFile("testdata/idemix/msp/IssuerPublicKey")
	require.NoError(t, err)
	require.NotEmpty(t, issuerPK)
	return issuerPK
}

func TestSerialization(t *testing.T) {
	// Use test helper instead of direct file read
	issuerPK := testingHelper(t)
	pp, err := Setup(32, issuerPK, math3.BN254)
	pp.ExtraData = map[string][]byte{
		"key1": []byte("value1"),
		"key2": []byte("value2"),
	}
	assert.NoError(t, err)
	ser, err := pp.Serialize()
	assert.NoError(t, err)

	pp2, err := NewPublicParamsFromBytes(ser, DLogNoGHDriverName, ProtocolV1)
	assert.NoError(t, err)

	ser2, err := pp2.Serialize()
	assert.NoError(t, err)

	assert.Equal(t, pp.IdemixIssuerPublicKeys, pp2.IdemixIssuerPublicKeys)
	assert.Equal(t, pp.PedersenGenerators, pp2.PedersenGenerators)
	assert.Equal(t, pp.RangeProofParams, pp2.RangeProofParams)
	assert.Equal(t, pp.ExtraData, pp2.ExtraData)

	assert.Equal(t, pp, pp2)
	assert.Equal(t, ser, ser2)

	// no issuers
	assert.NoError(t, pp.Validate())

	// with issuers
	pp.IssuerIDs = []driver.Identity{[]byte("issuer")}
	assert.NoError(t, pp.Validate())
}

func TestComputeMaxTokenValue(t *testing.T) {
	pp := PublicParams{
		RangeProofParams: &RangeProofParams{
			BitLength: 64,
		},
	}
	max := pp.ComputeMaxTokenValue()
	assert.Equal(t, uint64(18446744073709551615), max)

	pp = PublicParams{
		RangeProofParams: &RangeProofParams{
			BitLength: 16,
		},
	}
	max = pp.ComputeMaxTokenValue()
	assert.Equal(t, uint64(65535), max)
}

func TestNewG1(t *testing.T) {
	for i := range len(math3.Curves) {
		c := math3.Curves[i]
		assert.True(t, c.NewG1().IsInfinity())
	}
}

func TestRangeProofParamsValidation(t *testing.T) {
	curve := math3.BN254
	curveInst := math3.Curves[curve]

	// Create generators of the correct length
	makeGenerators := func(length uint64) []*math3.G1 {
		gen := make([]*math3.G1, length)
		for i := range gen {
			gen[i] = curveInst.GenG1
		}
		return gen
	}

	tests := []struct {
		name          string
		params        *RangeProofParams
		expectedError string
	}{
		{
			name: "valid params",
			params: &RangeProofParams{
				BitLength:       16,
				NumberOfRounds:  4,
				P:               curveInst.GenG1,
				Q:               curveInst.GenG1,
				LeftGenerators:  makeGenerators(16),
				RightGenerators: makeGenerators(16),
			},
			expectedError: "",
		},
		{
			name: "zero bit length",
			params: &RangeProofParams{
				BitLength:      0,
				NumberOfRounds: 4,
				P:              curveInst.GenG1,
				Q:              curveInst.GenG1,
			},
			expectedError: "invalid range proof parameters: bit length is zero",
		},
		{
			name: "zero rounds",
			params: &RangeProofParams{
				BitLength:      16,
				NumberOfRounds: 0,
				P:              curveInst.GenG1,
				Q:              curveInst.GenG1,
			},
			expectedError: "invalid range proof parameters: number of rounds is zero",
		},
		{
			name: "too many rounds",
			params: &RangeProofParams{
				BitLength:      16,
				NumberOfRounds: 65,
				P:              curveInst.GenG1,
				Q:              curveInst.GenG1,
			},
			expectedError: "invalid range proof parameters: number of rounds must be smaller or equal to 64",
		},
		{
			name: "mismatched generator lengths",
			params: &RangeProofParams{
				BitLength:       16,
				NumberOfRounds:  4,
				P:               curveInst.GenG1,
				Q:               curveInst.GenG1,
				LeftGenerators:  makeGenerators(15),
				RightGenerators: makeGenerators(16),
			},
			expectedError: "invalid range proof parameters: the size of the left generators does not match the size of the right generators",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate(curve)
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestIdemixIssuerPublicKeyConversion(t *testing.T) {
	original := &IdemixIssuerPublicKey{
		PublicKey: []byte("test-key"),
		Curve:     math3.BN254,
	}

	proto, err := original.ToProtos()
	require.NoError(t, err)

	recovered := &IdemixIssuerPublicKey{}
	err = recovered.FromProtos(proto)
	require.NoError(t, err)

	assert.Equal(t, original.PublicKey, recovered.PublicKey)
	assert.Equal(t, original.Curve, recovered.Curve)
}

func TestSetupWithInvalidParameters(t *testing.T) {
	tests := []struct {
		name          string
		bitLength     uint64
		expectedError string
	}{
		{
			name:          "bit length too large",
			bitLength:     65,
			expectedError: "invalid bit length [65], should be smaller than 64",
		},
		{
			name:          "zero bit length",
			bitLength:     0,
			expectedError: "invalid bit length, should be greater than 0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Setup(tt.bitLength, []byte("test-key"), math3.BN254)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestPublicParamsModification(t *testing.T) {
	pp, err := Setup(32, testingHelper(t), math3.BN254)
	require.NoError(t, err)

	// Test AddAuditor
	auditor1 := driver.Identity("auditor1")
	auditor2 := driver.Identity("auditor2")
	pp.AddAuditor(auditor1)
	pp.AddAuditor(auditor2)
	assert.Equal(t, []driver.Identity{auditor1, auditor2}, pp.Auditors())

	// Test SetAuditors
	newAuditors := []driver.Identity{driver.Identity("newAuditor")}
	pp.SetAuditors(newAuditors)
	assert.Equal(t, newAuditors, pp.Auditors())

	// Test AddIssuer
	issuer1 := driver.Identity("issuer1")
	issuer2 := driver.Identity("issuer2")
	pp.AddIssuer(issuer1)
	pp.AddIssuer(issuer2)
	assert.Equal(t, []driver.Identity{issuer1, issuer2}, pp.Issuers())

	// Test SetIssuers
	newIssuers := []driver.Identity{driver.Identity("newIssuer")}
	pp.SetIssuers(newIssuers)
	assert.Equal(t, newIssuers, pp.Issuers())
}

func TestPublicParamsValidation(t *testing.T) {
	validIssuerPK := &IdemixIssuerPublicKey{
		PublicKey: []byte("valid-key"),
		Curve:     math3.BN254,
	}

	tests := []struct {
		name          string
		setupParams   func() *PublicParams
		expectedError string
	}{
		{
			name: "valid params",
			setupParams: func() *PublicParams {
				issuerPK := testingHelper(t)
				pp, err := Setup(32, issuerPK, math3.BN254)
				require.NoError(t, err)
				return pp
			},
			expectedError: "",
		},
		{
			name: "invalid curve ID",
			setupParams: func() *PublicParams {
				pp := &PublicParams{
					Curve:                  math3.CurveID(999), // Invalid curve ID
					IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{validIssuerPK},
				}
				return pp
			},
			expectedError: "invalid public parameters: invalid curveID",
		},
		{
			name: "no idemix issuer public keys",
			setupParams: func() *PublicParams {
				pp := &PublicParams{
					Curve:                  math3.BN254,
					IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{},
				}
				return pp
			},
			expectedError: "expected one idemix issuer public key",
		},
		{
			name: "multiple idemix issuer public keys",
			setupParams: func() *PublicParams {
				pp := &PublicParams{
					Curve:                  math3.BN254,
					IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{validIssuerPK, validIssuerPK},
				}
				return pp
			},
			expectedError: "expected one idemix issuer public key",
		},
		{
			name: "nil idemix issuer public key",
			setupParams: func() *PublicParams {
				pp := &PublicParams{
					Curve:                  math3.BN254,
					IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{nil},
				}
				return pp
			},
			expectedError: "invalid idemix issuer public key, it is nil",
		},
		{
			name: "empty idemix issuer public key",
			setupParams: func() *PublicParams {
				pp := &PublicParams{
					Curve: math3.BN254,
					IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{
						{
							PublicKey: []byte{},
							Curve:     math3.BN254,
						},
					},
				}
				return pp
			},
			expectedError: "expected idemix issuer public key to be non-empty",
		},
		{
			name: "invalid idemix curve ID",
			setupParams: func() *PublicParams {
				issuerPK := testingHelper(t)
				pp, err := Setup(32, issuerPK, math3.BN254)
				require.NoError(t, err)
				pp.Curve = math3.CurveID(999) // Invalid curve ID
				return pp
			},
			expectedError: "invalid public parameters: invalid curveID [999 > 7]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := tt.setupParams()
			err := pp.Validate()
			if tt.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError)
			}
		})
	}
}

func TestComputeHash(t *testing.T) {
	pp, err := Setup(32, testingHelper(t), math3.BN254)
	require.NoError(t, err)

	hash1, err := pp.ComputeHash()
	require.NoError(t, err)
	assert.NotEmpty(t, hash1)

	// Modify parameters and check hash changes
	pp.AddIssuer(driver.Identity("newIssuer"))
	hash2, err := pp.ComputeHash()
	require.NoError(t, err)
	assert.NotEqual(t, hash1, hash2)
}

func TestWithVersion(t *testing.T) {
	version := driver.TokenDriverVersion(2)
	pp, err := WithVersion(32, testingHelper(t), math3.BN254, version)
	require.NoError(t, err)

	assert.Equal(t, version, pp.TokenDriverVersion())
	assert.Equal(t, driver.TokenDriverName(DLogNoGHDriverName), pp.TokenDriverName())
}

func TestSerializationEdgeCases(t *testing.T) {
	issuerPK := testingHelper(t)
	pp, err := Setup(32, issuerPK, math3.BN254)
	require.NoError(t, err)

	// Test invalid driver identifier
	pp.DriverName = "invalid"
	raw, err := pp.Serialize()
	require.NoError(t, err)
	_, err = NewPublicParamsFromBytes(raw, DLogNoGHDriverName, ProtocolV1)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid identifier")

	// Test corrupted data
	_, err = NewPublicParamsFromBytes([]byte("corrupted"), DLogNoGHDriverName, ProtocolV1)
	assert.Error(t, err)

	// Test deserialize with invalid proto
	_, err = NewPublicParamsFromBytes([]byte{0, 1, 2, 3}, DLogNoGHDriverName, ProtocolV1)
	assert.Error(t, err)
}

func TestPublicParamsString(t *testing.T) {
	pp := &PublicParams{
		DriverName:    DLogNoGHDriverName,
		DriverVersion: ProtocolV1,
		Curve:         math3.BN254,
		IdemixIssuerPublicKeys: []*IdemixIssuerPublicKey{
			{
				PublicKey: []byte("test-key"),
				Curve:     math3.BN254,
			},
		},
		AuditorIDs: []driver.Identity{[]byte("auditor1")},
		IssuerIDs:  []driver.Identity{[]byte("issuer1")},
		ExtraData: map[string][]byte{
			"key": []byte("value"),
		},
	}

	str := pp.String()
	res, err := json.MarshalIndent(pp, " ", "  ")
	require.NoError(t, err)
	assert.Equal(t, string(res), str)
}

func TestRangeProofParamsGenerators(t *testing.T) {
	curve := math3.BN254
	curveInst := math3.Curves[curve]

	// Test validation with nil generators
	params := &RangeProofParams{
		BitLength:      16,
		NumberOfRounds: 4,
		P:              curveInst.GenG1,
		Q:              curveInst.GenG1,
	}
	err := params.Validate(curve)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range proof parameters")

	// Test validation with nil P
	params = &RangeProofParams{
		BitLength:       16,
		NumberOfRounds:  4,
		Q:               curveInst.GenG1,
		LeftGenerators:  make([]*math3.G1, 16),
		RightGenerators: make([]*math3.G1, 16),
	}
	err = params.Validate(curve)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range proof parameters")

	// Test validation with nil Q
	params = &RangeProofParams{
		BitLength:       16,
		NumberOfRounds:  4,
		P:               curveInst.GenG1,
		LeftGenerators:  make([]*math3.G1, 16),
		RightGenerators: make([]*math3.G1, 16),
	}
	err = params.Validate(curve)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range proof parameters")
}

func TestRangeProofParamsConversion(t *testing.T) {
	curve := math3.BN254
	curveInst := math3.Curves[curve]

	original := &RangeProofParams{
		BitLength:       16,
		NumberOfRounds:  4,
		P:               curveInst.GenG1,
		Q:               curveInst.GenG1,
		LeftGenerators:  []*math3.G1{curveInst.GenG1},
		RightGenerators: []*math3.G1{curveInst.GenG1},
	}

	proto, err := original.ToProtos()
	require.NoError(t, err)

	recovered := &RangeProofParams{}
	err = recovered.FromProto(proto)
	require.NoError(t, err)

	assert.Equal(t, original.BitLength, recovered.BitLength)
	assert.Equal(t, original.NumberOfRounds, recovered.NumberOfRounds)
	assert.Equal(t, len(original.LeftGenerators), len(recovered.LeftGenerators))
	assert.Equal(t, len(original.RightGenerators), len(recovered.RightGenerators))
}

func TestGeneratePedersenParameters(t *testing.T) {
	pp := &PublicParams{
		Curve: math3.BN254,
	}

	err := pp.GeneratePedersenParameters()
	require.NoError(t, err)
	assert.Len(t, pp.PedersenGenerators, 3)

	// Test all generators are different
	for i := 0; i < len(pp.PedersenGenerators); i++ {
		for j := i + 1; j < len(pp.PedersenGenerators); j++ {
			assert.False(t, pp.PedersenGenerators[i].Equals(pp.PedersenGenerators[j]))
		}
	}
}

func TestGenerateRangeProofParameters(t *testing.T) {
	pp := &PublicParams{
		Curve: math3.BN254,
	}

	err := pp.GenerateRangeProofParameters(16)
	require.NoError(t, err)

	assert.NotNil(t, pp.RangeProofParams)
	assert.Equal(t, uint64(16), pp.RangeProofParams.BitLength)
	assert.Equal(t, uint64(4), pp.RangeProofParams.NumberOfRounds) // log2(16)
	assert.Len(t, pp.RangeProofParams.LeftGenerators, 16)
	assert.Len(t, pp.RangeProofParams.RightGenerators, 16)
	assert.NotNil(t, pp.RangeProofParams.P)
	assert.NotNil(t, pp.RangeProofParams.Q)

	// Test that P and Q are different
	assert.False(t, pp.RangeProofParams.P.Equals(pp.RangeProofParams.Q))

	// Test all generators are different
	for i, left := range pp.RangeProofParams.LeftGenerators {
		for j, right := range pp.RangeProofParams.RightGenerators {
			assert.False(t, left.Equals(right), "Left generator %d equals right generator %d", i, j)
		}
	}
}

func TestFailedSerialization(t *testing.T) {
	pp := &PublicParams{
		Curve:              math3.CurveID(999), // Invalid curve
		PedersenGenerators: []*math3.G1{nil},   // Invalid generator
	}

	_, err := pp.Serialize()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to serialize public parameters")
}

func TestLog2(t *testing.T) {
	tests := []struct {
		input    uint64
		expected uint64
	}{
		{1, 0},
		{2, 1},
		{4, 2},
		{8, 3},
		{16, 4},
		{32, 5},
		{64, 6},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("log2(%d)", tt.input), func(t *testing.T) {
			result := log2(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRangeProofParamsValidationWithMismatchedBitLength(t *testing.T) {
	curve := math3.BN254
	curveInst := math3.Curves[curve]

	// Test when BitLength doesn't match the expected value based on NumberOfRounds
	params := &RangeProofParams{
		BitLength:       32, // This should be 16 for NumberOfRounds=4
		NumberOfRounds:  4,
		P:               curveInst.GenG1,
		Q:               curveInst.GenG1,
		LeftGenerators:  make([]*math3.G1, 32),
		RightGenerators: make([]*math3.G1, 32),
	}
	err := params.Validate(curve)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid range proof parameters: bit length should be")
}

func TestRangeProofParamsInvalidPoints(t *testing.T) {
	curve := math3.BN254
	curveInst := math3.Curves[curve]

	// Create a point from another curve
	invalidPoint := math3.Curves[math3.BLS12_381_BBS_GURVY].GenG1

	issuerPK := testingHelper(t)
	pp, err := Setup(32, issuerPK, math3.BN254)
	require.NoError(t, err)

	tests := []struct {
		name          string
		params        *RangeProofParams
		expectedError string
	}{
		{
			name: "invalid P point",
			params: &RangeProofParams{
				BitLength:      pp.RangeProofParams.BitLength,
				NumberOfRounds: pp.RangeProofParams.NumberOfRounds,
				P:              invalidPoint,
				Q:              pp.RangeProofParams.Q,
			},
			expectedError: "invalid range proof parameters: generator P is invalid",
		},
		{
			name: "invalid Q point",
			params: &RangeProofParams{
				BitLength:      pp.RangeProofParams.BitLength,
				NumberOfRounds: pp.RangeProofParams.NumberOfRounds,
				P:              pp.RangeProofParams.P,
				Q:              invalidPoint,
			},
			expectedError: "invalid range proof parameters: generator Q is invalid",
		},
		{
			name: "invalid left generator",
			params: &RangeProofParams{
				BitLength:      pp.RangeProofParams.BitLength,
				NumberOfRounds: pp.RangeProofParams.NumberOfRounds,
				P:              pp.RangeProofParams.P,
				Q:              pp.RangeProofParams.Q,
				LeftGenerators: []*math3.G1{
					curveInst.GenG1,
					pp.RangeProofParams.LeftGenerators[1],
				},
				RightGenerators: make([]*math3.G1, 2),
			},
			expectedError: "invalid range proof parameters, left generators is invalid",
		},
		{
			name: "invalid right generator",
			params: &RangeProofParams{
				BitLength:       pp.RangeProofParams.BitLength,
				NumberOfRounds:  pp.RangeProofParams.NumberOfRounds,
				P:               curveInst.GenG1,
				Q:               curveInst.GenG1,
				LeftGenerators:  pp.RangeProofParams.LeftGenerators,
				RightGenerators: make([]*math3.G1, len(pp.RangeProofParams.RightGenerators)),
			},
			expectedError: "invalid range proof parameters, right generators is invalid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.params.Validate(curve)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}
