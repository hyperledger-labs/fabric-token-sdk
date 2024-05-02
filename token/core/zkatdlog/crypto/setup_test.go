/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"fmt"
	"os"
	"testing"
	"time"

	math3 "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	s := time.Now()
	_, err := Setup(32, []byte("issuerPK"), math3.FP256BN_AMCL)
	e := time.Now()
	fmt.Printf("elapsed %d", e.Sub(s).Milliseconds())
	assert.NoError(t, err)

}

func TestSerialization(t *testing.T) {
	issuerPK, err := os.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
	assert.NoError(t, err)
	pp, err := Setup(32, issuerPK, math3.BN254)
	assert.NoError(t, err)

	ser, err := pp.Serialize()
	assert.NoError(t, err)

	pp2, err := NewPublicParamsFromBytes(ser, DLogPublicParameters)
	assert.NoError(t, err)

	ser2, err := pp2.Serialize()
	assert.NoError(t, err)

	assert.Equal(t, pp.IdemixIssuerPK, pp2.IdemixIssuerPK)
	assert.Equal(t, pp.PedersenGenerators, pp2.PedersenGenerators)
	assert.Equal(t, pp.RangeProofParams, pp2.RangeProofParams)

	assert.Equal(t, pp, pp2)
	assert.Equal(t, ser, ser2)

	assert.NoError(t, pp.Validate())

	pp.Issuers = [][]byte{[]byte("issuer")}
	assert.NoError(t, pp.Validate())

}

func TestNewG1(t *testing.T) {
	for i := 0; i < len(math3.Curves); i++ {
		c := math3.Curves[i]
		assert.Equal(t, c.NewG1().IsInfinity(), true)
	}
}
