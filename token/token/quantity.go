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

	// Add returns this + b without modify this.
	// If an overflow occurs, it returns an error.
	Add(b Quantity) Quantity

	// Add returns this - b without modify this.
	// If an overflow occurs, it returns an error.
	Sub(b Quantity) Quantity

	// Cmd compare this with b
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
	v, success := big.NewInt(0).SetString(q, 0)
	if !success {
		return nil, errors.Errorf("invalid input [%s,%d]", q, precision)
	}
	if v.Cmp(big.NewInt(0)) < 0 {
		return nil, errors.New("quantity must be larger than 0")
	}
	if precision == 0 {
		return nil, errors.New("precision be larger than 0")
	}
	if v.BitLen() > int(precision) {
		return nil, errors.Errorf("%s has precision %d > %d", q, v.BitLen(), precision)
	}

	return &BigQuantity{Int: v, Precision: precision}, nil
}

// NewZeroQuantity returns to zero quantity at the passed precision/
// The precision is expressed in bits.
func NewZeroQuantity(precision uint64) Quantity {
	b := BigQuantity{Int: big.NewInt(0), Precision: precision}
	return &b
}

func NewQuantityFromUInt64(q uint64) Quantity {
	v, _ := big.NewInt(0).SetString(strconv.FormatUint(q, 10), 10)
	return &BigQuantity{Int: v, Precision: 64}
}

func NewQuantityFromBig64(q *big.Int) Quantity {
	if q.BitLen() > 64 {
		panic(fmt.Sprintf("invalid precision, expected at most 64 bits"))
	}
	return &BigQuantity{Int: q, Precision: 64}
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

	sumq := BigQuantity{Int: sum, Precision: q.Precision}
	return &sumq
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

	diffq := BigQuantity{Int: diff, Precision: q.Precision}
	return &diffq
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
