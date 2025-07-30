/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token_test

import (
	"math"
	"strconv"
	"testing"

	"github.com/hyperledger-labs/fabric-token-sdk/token/token"
	"github.com/stretchr/testify/assert"
)

func TestToQuantity(t *testing.T) {
	_, err := token.ToQuantity(ToHex(100), 0)
	assert.Equal(t, "precision must be larger than 0", err.Error())

	_, err = token.ToQuantity(IntToHex(-100), 64)
	assert.Equal(t, "invalid input [0x-64,64]", err.Error())

	_, err = token.ToQuantity("abc", 64)
	assert.Equal(t, "invalid input [abc,64]", err.Error())

	_, err = token.ToQuantity("0babc", 64)
	assert.Equal(t, "invalid input [0babc,64]", err.Error())

	_, err = token.ToQuantity("0abc", 64)
	assert.Equal(t, "invalid input [0abc,64]", err.Error())

	_, err = token.ToQuantity("0xabc", 2)
	assert.Equal(t, "0xabc has precision 12 > 2", err.Error())

	_, err = token.ToQuantity("10231", 64)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0xABC", 64)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0XABC", 64)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0XAbC", 64)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0xAbC", 64)
	assert.NoError(t, err)

	_, err = token.ToQuantity(IntToHex(-100), 128)
	assert.Equal(t, "invalid input [0x-64,128]", err.Error())

	_, err = token.ToQuantity("abc", 128)
	assert.Equal(t, "invalid input [abc,128]", err.Error())

	_, err = token.ToQuantity("0babc", 128)
	assert.Equal(t, "invalid input [0babc,128]", err.Error())

	_, err = token.ToQuantity("0abc", 128)
	assert.Equal(t, "invalid input [0abc,128]", err.Error())

	_, err = token.ToQuantity("0xabc", 2)
	assert.Equal(t, "0xabc has precision 12 > 2", err.Error())

	_, err = token.ToQuantity("10231", 128)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0xABC", 128)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0XABC", 128)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0XAbC", 128)
	assert.NoError(t, err)

	_, err = token.ToQuantity("0xAbC", 128)
	assert.NoError(t, err)
}

func TestDecimal(t *testing.T) {
	q, err := token.ToQuantity("10231", 64)
	assert.NoError(t, err)
	assert.Equal(t, "10231", q.Decimal())

	q, err = token.ToQuantity("10231", 128)
	assert.NoError(t, err)
	assert.Equal(t, "10231", q.Decimal())
}

func TestHex(t *testing.T) {
	q, err := token.ToQuantity("0xabc", 64)
	assert.NoError(t, err)
	assert.Equal(t, "0xabc", q.Hex())

	q, err = token.ToQuantity("0xabc", 128)
	assert.NoError(t, err)
	assert.Equal(t, "0xabc", q.Hex())
}

func TestOverflow(t *testing.T) {
	half := uint64(math.MaxUint64 / 2)
	assert.Equal(t, uint64(math.MaxUint64), half+half+1)

	a, err := token.ToQuantity(ToHex(1), 64)
	assert.NoError(t, err)
	b, err := token.ToQuantity(ToHex(uint64(math.MaxUint64)), 64)
	assert.NoError(t, err)

	assert.Panics(t, func() {
		a.Add(b)
	})
	assert.Panics(t, func() {
		b.Add(b)
	})

	a, err = token.NewUBigQuantity(ToHex(1), 64)
	assert.NoError(t, err)
	b, err = token.NewUBigQuantity(ToHex(uint64(math.MaxUint64)), 64)
	assert.NoError(t, err)

	assert.Panics(t, func() {
		a.Add(b)
	})
	assert.Panics(t, func() {
		b.Add(b)
	})
}

func TestBigQuantity_Add(t *testing.T) {
	q := token.NewZeroQuantity(128).(*token.BigQuantity)
	b := token.NewOneQuantity(128)

	result := q.Add(b)

	assert.Equal(t, "1", result.Decimal())
}

func TestBigQuantity_Sub(t *testing.T) {
	q := token.NewOneQuantity(128).(*token.BigQuantity)
	b := token.NewOneQuantity(128)

	result := q.Sub(b)

	assert.Equal(t, "0", result.Decimal())
}

func TestBigQuantity_Cmp(t *testing.T) {
	q := token.NewZeroQuantity(128).(*token.BigQuantity)
	b := token.NewOneQuantity(128)

	assert.Equal(t, -1, q.Cmp(b))

	q = token.NewOneQuantity(128).(*token.BigQuantity)
	assert.Equal(t, 0, q.Cmp(b))

	q = token.NewOneQuantity(128).(*token.BigQuantity)
	b = token.NewZeroQuantity(128)
	assert.Equal(t, 1, q.Cmp(b))
}

func TestBigQuantity_Hex(t *testing.T) {
	q := token.NewZeroQuantity(128).(*token.BigQuantity)

	assert.Equal(t, "0x0", q.Hex())
}

func TestBigQuantity_Decimal(t *testing.T) {
	q := token.NewZeroQuantity(128).(*token.BigQuantity)

	assert.Equal(t, "0", q.Decimal())
}

func TestUInt64Quantity_Add(t *testing.T) {
	q := token.NewOneQuantity(64).(*token.UInt64Quantity)
	b := token.NewZeroQuantity(64)

	result := q.Add(b)

	assert.Equal(t, "1", result.Decimal())
}

func TestUInt64Quantity_Sub(t *testing.T) {
	q := token.NewQuantityFromUInt64(2).(*token.UInt64Quantity)
	b := token.NewOneQuantity(64)

	result := q.Sub(b)

	assert.Equal(t, "1", result.Decimal())
}

func TestUInt64Quantity_Cmp(t *testing.T) {
	q := token.NewOneQuantity(64).(*token.UInt64Quantity)
	b := token.NewQuantityFromUInt64(2)

	assert.Equal(t, -1, q.Cmp(b))

	q = token.NewQuantityFromUInt64(2).(*token.UInt64Quantity)
	assert.Equal(t, 0, q.Cmp(b))

	q = token.NewQuantityFromUInt64(2).(*token.UInt64Quantity)
	b = token.NewOneQuantity(64)
	assert.Equal(t, 1, q.Cmp(b))
}

func TestUInt64Quantity_Hex(t *testing.T) {
	q := token.NewQuantityFromUInt64(255).(*token.UInt64Quantity)

	assert.Equal(t, "0xff", q.Hex())
}

func TestUInt64Quantity_Decimal(t *testing.T) {
	q := token.NewQuantityFromUInt64(123).(*token.UInt64Quantity)

	assert.Equal(t, "123", q.Decimal())
}

func TestBigQuantity_Add_Panic(t *testing.T) {
	q := token.NewZeroQuantity(2).(*token.BigQuantity) // Precision set to 2 bits
	b := token.NewOneQuantity(64)                      // Precision set to 64 bits

	assert.Panics(t, func() {
		_ = q.Add(b)
	})
}

func TestBigQuantity_Sub_Panic(t *testing.T) {
	q := token.NewZeroQuantity(128).(*token.BigQuantity)
	b := token.NewOneQuantity(128)

	assert.Panics(t, func() {
		_ = q.Sub(b)
	})
}

func TestUInt64Quantity_Add_Panic(t *testing.T) {
	q := token.NewQuantityFromUInt64(18446744073709551615) // Max uint64 value
	b := token.NewQuantityFromUInt64(1)

	assert.Panics(t, func() {
		_ = q.Add(b)
	})
}

func TestUInt64Quantity_Sub_Panic(t *testing.T) {
	q := token.NewQuantityFromUInt64(0)
	b := token.NewQuantityFromUInt64(1)

	assert.Panics(t, func() {
		_ = q.Sub(b)
	})
}

func TestNewUBigQuantity_ValidInput(t *testing.T) {
	q, err := token.NewUBigQuantity("123456789", 64)

	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, "123456789", q.Decimal())
}

func TestNewUBigQuantity_InvalidInput(t *testing.T) {
	_, err := token.NewUBigQuantity("invalid", 64)

	assert.Error(t, err)
}

func TestNewUBigQuantity_ZeroPrecision(t *testing.T) {
	_, err := token.NewUBigQuantity("123456789", 0)

	assert.Error(t, err)
}

func TestNewUBigQuantity_NegativeQuantity(t *testing.T) {
	_, err := token.NewUBigQuantity("-123", 64)

	assert.Error(t, err)
}

func TestNewUBigQuantity_Overflow(t *testing.T) {
	_, err := token.NewUBigQuantity("18446744073709551616", 64) // Max uint64 + 1

	assert.Error(t, err)
}

func TestUInt64ToQuantity_ValidInput(t *testing.T) {
	q, err := token.UInt64ToQuantity(123456789, 64)

	assert.NoError(t, err)
	assert.NotNil(t, q)
	assert.Equal(t, "123456789", q.Decimal())
}

func TestUInt64ToQuantity_ZeroPrecision(t *testing.T) {
	_, err := token.UInt64ToQuantity(123456789, 0)

	assert.Error(t, err)
}

func TestUInt64ToQuantity_NegativeQuantity(t *testing.T) {
	_, err := token.UInt64ToQuantity(18446744073709551615, 64) // Max uint64

	assert.NoError(t, err)
}

func TestUInt64ToQuantity_Precision_InsufficientError(t *testing.T) {
	_, err := token.UInt64ToQuantity(18446744073709551615, 10) // Max uint64

	assert.Error(t, err)
}

func ToHex(q uint64) string {
	return "0x" + strconv.FormatUint(q, 16)
}

func IntToHex(q int64) string {
	return "0x" + strconv.FormatInt(q, 16)
}
