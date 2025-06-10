/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package setup

import (
	"os"
	"testing"

	math3 "github.com/IBM/mathlib"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/stretchr/testify/assert"
)

func TestSerialization(t *testing.T) {
	issuerPK, err := os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
	assert.NoError(t, err)
	pp, err := Setup(32, issuerPK, math3.BN254)
	assert.NoError(t, err)

	ser, err := pp.Serialize()
	assert.NoError(t, err)

	pp2, err := NewPublicParamsFromBytes(ser, DLogIdentifier)
	assert.NoError(t, err)

	ser2, err := pp2.Serialize()
	assert.NoError(t, err)

	assert.Equal(t, pp.IdemixIssuerPublicKeys, pp2.IdemixIssuerPublicKeys)
	assert.Equal(t, pp.PedersenGenerators, pp2.PedersenGenerators)
	assert.Equal(t, pp.RangeProofParams, pp2.RangeProofParams)

	assert.Equal(t, pp, pp2)
	assert.Equal(t, ser, ser2)

	assert.Error(t, pp.Validate())

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
	for i := 0; i < len(math3.Curves); i++ {
		c := math3.Curves[i]
		assert.Equal(t, c.NewG1().IsInfinity(), true)
	}
}
