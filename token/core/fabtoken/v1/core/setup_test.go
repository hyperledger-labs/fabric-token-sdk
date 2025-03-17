/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetup_ValidPrecision(t *testing.T) {
	pp, err := Setup(32)
	assert.NoError(t, err)
	assert.Equal(t, uint64(32), pp.QuantityPrecision)
	assert.Equal(t, uint64(1<<32)-1, pp.MaxToken)
}

func TestSetup_InvalidPrecision(t *testing.T) {
	pp, err := Setup(65)
	assert.Error(t, err)
	assert.Nil(t, pp)
	assert.Equal(t, "invalid precision [65], must be smaller or equal than 64", err.Error())
}

func TestNewPublicParamsFromBytes_Valid(t *testing.T) {
	pp, err := Setup(32)
	assert.NoError(t, err)
	raw, err := pp.Serialize()
	assert.NoError(t, err)
	pp, err = NewPublicParamsFromBytes(raw, "fabtoken")
	assert.NoError(t, err)
	assert.Equal(t, "fabtoken", pp.Label)
	assert.Equal(t, uint64(32), pp.QuantityPrecision)
	assert.Equal(t, uint64(4294967295), pp.MaxToken)
}

func TestPublicParams_Validate_Valid(t *testing.T) {
	pp := &PublicParams{
		Label:             "fabtoken",
		QuantityPrecision: 32,
		MaxToken:          1<<32 - 1,
	}
	err := pp.Validate()
	assert.NoError(t, err)
}

func TestPublicParams_Validate_InvalidPrecision(t *testing.T) {
	pp := &PublicParams{
		Label:             "fabtoken",
		QuantityPrecision: 65,
		MaxToken:          1<<64 - 1,
	}
	err := pp.Validate()
	assert.Error(t, err)
	assert.Equal(t, "invalid precision [65], must be less than 64", err.Error())
}

func TestPublicParams_Validate_InvalidMaxToken(t *testing.T) {
	pp := &PublicParams{
		Label:             "fabtoken",
		QuantityPrecision: 32,
		MaxToken:          1 << 32,
	}
	err := pp.Validate()
	assert.Error(t, err)
	assert.Equal(t, "max token value is invalid [4294967296]>[4294967295]", err.Error())
}
