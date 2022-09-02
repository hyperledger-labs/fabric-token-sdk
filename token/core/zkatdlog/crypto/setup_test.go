/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package crypto

import (
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	math3 "github.com/IBM/mathlib"
	"github.com/stretchr/testify/assert"
)

func TestSetup(t *testing.T) {
	s := time.Now()
	_, err := Setup(100, 2, nil, math3.FP256BN_AMCL)
	e := time.Now()
	fmt.Printf("elapsed %d", e.Sub(s).Milliseconds())
	assert.NoError(t, err)
}

func TestSerialization(t *testing.T) {
	raw, err := ioutil.ReadFile("./testdata/idemix/msp/IssuerPublicKey")
	assert.NoError(t, err)
	pp, err := Setup(100, 2, raw, math3.BN254)
	assert.NoError(t, err)
	ser, err := pp.Serialize()
	assert.NoError(t, err)

	pp2, err := NewPublicParamsFromBytes(ser, DLogPublicParameters)
	assert.NoError(t, err)
	ser2, err := pp2.Serialize()
	assert.NoError(t, err)

	assert.Equal(t, pp.IdemixIssuerPK, pp2.IdemixIssuerPK)
	assert.Equal(t, pp.PedGen, pp2.PedGen)
	assert.Equal(t, pp.PedParams, pp2.PedParams)
	assert.Equal(t, pp.RangeProofParams, pp2.RangeProofParams)

	assert.Equal(t, pp, pp2)
	assert.Equal(t, ser, ser2)
}
