/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"math"
	"strconv"
	"testing"

	token2 "github.com/hyperledger-labs/fabric-token-sdk/token/token"

	"github.com/stretchr/testify/assert"
)

func TestToQuantity(t *testing.T) {
	_, err := token2.ToQuantity(ToHex(100), 0)
	assert.Equal(t, "precision be larger than 0", err.Error())

	_, err = token2.ToQuantity(IntToHex(-100), 64)
	assert.Equal(t, "invalid input [0x-64,64]", err.Error())

	_, err = token2.ToQuantity("abc", 64)
	assert.Equal(t, "invalid input [abc,64]", err.Error())

	_, err = token2.ToQuantity("0babc", 64)
	assert.Equal(t, "invalid input [0babc,64]", err.Error())

	_, err = token2.ToQuantity("0abc", 64)
	assert.Equal(t, "invalid input [0abc,64]", err.Error())

	_, err = token2.ToQuantity("0xabc", 2)
	assert.Equal(t, "0xabc has precision 12 > 2", err.Error())

	_, err = token2.ToQuantity("10231", 64)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0xABC", 64)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0XABC", 64)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0XAbC", 64)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0xAbC", 64)
	assert.NoError(t, err)

	_, err = token2.ToQuantity(IntToHex(-100), 128)
	assert.Equal(t, "invalid input [0x-64,128]", err.Error())

	_, err = token2.ToQuantity("abc", 128)
	assert.Equal(t, "invalid input [abc,128]", err.Error())

	_, err = token2.ToQuantity("0babc", 128)
	assert.Equal(t, "invalid input [0babc,128]", err.Error())

	_, err = token2.ToQuantity("0abc", 128)
	assert.Equal(t, "invalid input [0abc,128]", err.Error())

	_, err = token2.ToQuantity("0xabc", 2)
	assert.Equal(t, "0xabc has precision 12 > 2", err.Error())

	_, err = token2.ToQuantity("10231", 128)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0xABC", 128)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0XABC", 128)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0XAbC", 128)
	assert.NoError(t, err)

	_, err = token2.ToQuantity("0xAbC", 128)
	assert.NoError(t, err)
}

func TestDecimal(t *testing.T) {
	q, err := token2.ToQuantity("10231", 64)
	assert.NoError(t, err)
	assert.Equal(t, "10231", q.Decimal())

	q, err = token2.ToQuantity("10231", 128)
	assert.NoError(t, err)
	assert.Equal(t, "10231", q.Decimal())
}

func TestHex(t *testing.T) {
	q, err := token2.ToQuantity("0xabc", 64)
	assert.NoError(t, err)
	assert.Equal(t, "0xabc", q.Hex())

	q, err = token2.ToQuantity("0xabc", 128)
	assert.NoError(t, err)
	assert.Equal(t, "0xabc", q.Hex())
}

func TestOverflow(t *testing.T) {
	half := uint64(math.MaxUint64 / 2)
	assert.Equal(t, uint64(math.MaxUint64), uint64(half+half+1))

	a, err := token2.ToQuantity(ToHex(1), 64)
	assert.NoError(t, err)
	b, err := token2.ToQuantity(ToHex(uint64(math.MaxUint64)), 64)
	assert.NoError(t, err)

	assert.Panics(t, func() {
		a.Add(b)
	})
	assert.Panics(t, func() {
		b.Add(b)
	})

	a, err = token2.ToBigQuantity(ToHex(1), 64)
	assert.NoError(t, err)
	b, err = token2.ToBigQuantity(ToHex(uint64(math.MaxUint64)), 64)
	assert.NoError(t, err)

	assert.Panics(t, func() {
		a.Add(b)
	})
	assert.Panics(t, func() {
		b.Add(b)
	})
}

func ToHex(q uint64) string {
	return "0x" + strconv.FormatUint(q, 16)
}

func IntToHex(q int64) string {
	return "0x" + strconv.FormatInt(q, 16)
}
