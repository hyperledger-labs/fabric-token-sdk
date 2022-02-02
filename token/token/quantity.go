/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package token

import (
	"fmt"
	"math/big"
	"strconv"

	"github.com/pkg/errors"
)

// Quantity models an immutable token quantity and its basic operations.
type Quantity interface {

	// Add returns this + b modifying this.
	// If an overflow occurs, it returns an error.
	Add(b Quantity) Quantity

	// Sub returns this - b modifying this.
	// If an overflow occurs, it returns an error.
	Sub(b Quantity) Quantity

	// Cmp compare this with b
	Cmp(b Quantity) int

	// Hex returns the hexadecimal representation of this quantity
	Hex() string

	// Decimal returns the decimal representation of this quantity
	Decimal() string

	// ToBigInt returns the big int representation of this quantity
	ToBigInt() *big.Int
}

type BigQuantity struct {
	*big.Int
	Precision uint64
}

// ToQuantity converts a string q to a BigQuantity of a given precision.
// Argument q is supposed to be formatted following big.Int#scan specification.
// The precision is expressed in bits.
func ToQuantity(q string, precision uint64) (Quantity, error) {
	if precision == 0 {
		return nil, errors.New("precision be larger than 0")
	}
	v, success := big.NewInt(0).SetString(q, 0)
	if !success {
		return nil, errors.Errorf("invalid input [%s,%d]", q, precision)
	}
	if v.Cmp(big.NewInt(0)) < 0 {
		return nil, errors.New("quantity must be larger than 0")
	}
	if v.BitLen() > int(precision) {
		return nil, errors.Errorf("%s has precision %d > %d", q, v.BitLen(), precision)
	}

	switch precision {
	case 64:
		return &UInt64Quantity{Value: v.Uint64()}, nil
	default:
		return &BigQuantity{Int: v, Precision: precision}, nil
	}
}

func ToBigQuantity(q string, precision uint64) (Quantity, error) {
	if precision == 0 {
		return nil, errors.New("precision be larger than 0")
	}
	v, success := big.NewInt(0).SetString(q, 0)
	if !success {
		return nil, errors.Errorf("invalid input [%s,%d]", q, precision)
	}
	if v.Cmp(big.NewInt(0)) < 0 {
		return nil, errors.New("quantity must be larger than 0")
	}
	if v.BitLen() > int(precision) {
		return nil, errors.Errorf("%s has precision %d > %d", q, v.BitLen(), precision)
	}

	return &BigQuantity{Int: v, Precision: precision}, nil
}

// NewZeroQuantity returns to zero quantity at the passed precision/
// The precision is expressed in bits.
func NewZeroQuantity(precision uint64) Quantity {
	switch precision {
	case 64:
		return &UInt64Quantity{Value: 0}
	default:
		return &BigQuantity{Int: big.NewInt(0), Precision: precision}
	}
}

func NewQuantityFromUInt64(q uint64) Quantity {
	return &UInt64Quantity{Value: q}
}

func NewQuantityFromBig64(q *big.Int) Quantity {
	if q.BitLen() > 64 {
		panic(fmt.Sprintf("invalid precision, expected at most 64 bits"))
	}
	return &UInt64Quantity{Value: q.Uint64()}
}

func (q *BigQuantity) Add(b Quantity) Quantity {
	bq, ok := b.(*BigQuantity)
	if !ok {
		panic(fmt.Sprintf("expected BigQuantity, got '%t", b))
	}

	// Check overflow
	sum := big.NewInt(0)
	sum = sum.Add(q.Int, bq.Int)

	if sum.BitLen() > int(q.Precision) {
		panic(fmt.Sprintf("%s < %s", q.Text(10), b.Decimal()))
	}

	q.Int = sum
	return q
}

func (q *BigQuantity) Sub(b Quantity) Quantity {
	bq, ok := b.(*BigQuantity)
	if !ok {
		panic(fmt.Sprintf("expected BigQuantity, got '%t", b))
	}

	// Check overflow
	if q.Int.Cmp(bq.Int) < 0 {
		panic(fmt.Sprintf("%s < %s", q.Text(10), b.Decimal()))
	}
	diff := big.NewInt(0)
	diff.Sub(q.Int, b.(*BigQuantity).Int)

	q.Int = diff
	return q
}

func (q *BigQuantity) Cmp(b Quantity) int {
	bq, ok := b.(*BigQuantity)
	if !ok {
		panic(fmt.Sprintf("expected BigQuantity, got '%t", b))
	}

	return q.Int.Cmp(bq.Int)
}

func (q *BigQuantity) Hex() string {
	return "0x" + q.Int.Text(16)
}

func (q *BigQuantity) Decimal() string {
	return q.Int.Text(10)
}

func (q *BigQuantity) String() string {
	return q.Int.Text(10)
}

func (q *BigQuantity) ToBigInt() *big.Int {
	return (&big.Int{}).Set(q.Int)
}

type UInt64Quantity struct {
	Value uint64
}

func (q *UInt64Quantity) Add(b Quantity) Quantity {
	bq, ok := b.(*UInt64Quantity)
	if !ok {
		panic(fmt.Sprintf("expected UInt64Quantity, got '%t", b))
	}

	// Check overflow
	var sum uint64
	sum = q.Value + bq.Value

	if sum < q.Value {
		panic(fmt.Sprintf("%d < %d", q.Value, bq.Value))
	}

	q.Value = sum
	return q
}

func (q *UInt64Quantity) Sub(b Quantity) Quantity {
	bq, ok := b.(*UInt64Quantity)
	if !ok {
		panic(fmt.Sprintf("expected UInt64Quantity, got '%t", b))
	}

	// Check overflow
	if bq.Value > q.Value {
		panic(fmt.Sprintf("%d < %d", q.Value, bq.Value))
	}
	diff := q.Value - bq.Value

	q.Value = diff
	return q
}

func (q *UInt64Quantity) Cmp(b Quantity) int {
	bq, ok := b.(*UInt64Quantity)
	if !ok {
		panic(fmt.Sprintf("expected UInt64Quantity, got '%t", b))
	}

	if q.Value < bq.Value {
		return -1
	} else if q.Value > bq.Value {
		return 1
	}
	return 0
}

func (q *UInt64Quantity) Hex() string {
	return "0x" + strconv.FormatUint(q.Value, 16)
}

func (q *UInt64Quantity) Decimal() string {
	return strconv.FormatUint(q.Value, 10)
}

func (q *UInt64Quantity) ToBigInt() *big.Int {
	return big.NewInt(0).SetUint64(q.Value)
}
