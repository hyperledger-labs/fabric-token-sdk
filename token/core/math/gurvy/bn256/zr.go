/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"encoding/json"
	"math/big"
)

type Zr big.Int

func NewZr() *Zr {
	return (*Zr)(big.NewInt(0))
}

func NewZrCopy(a *Zr) *Zr {
	return (*Zr)(big.NewInt(0).SetBytes((*big.Int)(a).Bytes()))
}

func NewZrInt(i int) *Zr {
	return (*Zr)(big.NewInt(int64(i)))
}

func NewZrFromBytes(b []byte) *Zr {
	return (*Zr)(big.NewInt(0).SetBytes(b))
}

func (z *Zr) SetUint64(x uint64) *Zr {
	return (*Zr)((*big.Int)(z).SetUint64(x))
}

func (z *Zr) Mod(order *Zr) {
	(*big.Int)(z).Mod((*big.Int)(z), (*big.Int)(order))
}

func (z *Zr) Plus(b *Zr) *Zr {
	r := (&big.Int{}).Set((*big.Int)(z))

	return (*Zr)(r.Add((*big.Int)(z), (*big.Int)(b)))
}

func (z *Zr) Minus(a *Zr) *Zr {
	r := (&big.Int{}).Set((*big.Int)(z))

	return (*Zr)(r.Sub((*big.Int)(z), (*big.Int)(a)))
}

func (z *Zr) String() string {
	return (*big.Int)(z).Text(16)
}

func (z *Zr) Bytes() []byte {
	return (*big.Int)(z).Bytes()
}

func (z *Zr) InvModP(a *Zr) {
	(*big.Int)(z).ModInverse((*big.Int)(z), (*big.Int)(a))
}

func (z *Zr) PowMod(a *Zr, b *Zr) *Zr {
	r := (&big.Int{}).Set((*big.Int)(z))

	return (*Zr)(r.Exp((*big.Int)(z), (*big.Int)(a), (*big.Int)(b)))
}

func (z *Zr) Cmp(a *Zr) int {
	return (*big.Int)(z).Cmp((*big.Int)(a))
}

func Sum(values []*Zr) *Zr {
	sum := NewZr()
	for i := 0; i < len(values); i++ {
		sum = ModAdd(sum, values[i], Order)
	}
	return sum
}

func ModNeg(a1, m *Zr) *Zr {
	if a1 == nil {
		return NewZrInt(0)
	}
	a := NewZrCopy(a1)
	a.Mod(m)
	return m.Minus(a)
}

func ModAdd(a, b, m *Zr) *Zr {
	if a == nil {
		a = NewZrInt(0)
	}
	if b == nil {
		b = NewZrInt(0)
	}
	c := a.Plus(b)
	c.Mod(m)
	return c
}

func ModSub(a, b, m *Zr) *Zr {
	c := ModNeg(b, m)
	return ModAdd(a, c, m)
}

func ModMul(a1, b1, m *Zr) *Zr {
	if a1 == nil || b1 == nil {
		return NewZrInt(0)
	}
	a := NewZrCopy(a1)
	b := NewZrCopy(b1)
	a.Mod(m)
	b.Mod(m)
	d := (*big.Int)(a).Mul((*big.Int)(a), (*big.Int)(b))
	return (*Zr)(d.Mod(d, (*big.Int)(m)))
}

func (z *Zr) IsZero() bool {
	if z.Cmp(NewZrInt(0)) == 0 {
		return true
	}
	return false
}

func (z *Zr) MarshalJSON() ([]byte, error) {
	return json.Marshal(z.Bytes())
}

func (z *Zr) UnmarshalJSON(raw []byte) error {
	var r []byte
	err := json.Unmarshal(raw, &r)
	if err != nil {
		return err
	}
	*z = *((*Zr)(big.NewInt(0).SetBytes(r)))

	return err
}

func (z *Zr) Int64() int64 {
	return (*big.Int)(z).Int64()
}
