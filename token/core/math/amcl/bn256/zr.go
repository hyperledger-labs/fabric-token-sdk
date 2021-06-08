/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"encoding/json"

	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

type Zr FP256BN.BIG

func NewZr() *Zr {
	return (*Zr)(FP256BN.NewBIG())
}

func NewZrCopy(a *Zr) *Zr {
	return (*Zr)(FP256BN.NewBIGcopy((*FP256BN.BIG)(a)))
}

func NewZrInt(i int) *Zr {
	return (*Zr)(FP256BN.NewBIGint(i))
}

func NewZrFromBytes(b []byte) *Zr {
	return (*Zr)(FP256BN.FromBytes(b))
}

func (z *Zr) Mod(order *Zr) {
	(*FP256BN.BIG)(z).Mod((*FP256BN.BIG)(order))
}

func (z *Zr) Plus(b *Zr) *Zr {
	return (*Zr)((*FP256BN.BIG)(z).Plus((*FP256BN.BIG)(b)))
}

func (z *Zr) Minus(a *Zr) *Zr {
	return (*Zr)((*FP256BN.BIG)(z).Minus((*FP256BN.BIG)(a)))
}

/* Convert to Hex String */
func (z *Zr) String() string {
	return (*FP256BN.BIG)(z).ToString()
}

func (z *Zr) Bytes() []byte {
	b := make([]byte, FP256BN.MODBYTES)
	(*FP256BN.BIG)(z).ToBytes(b)
	return b
}

func (z *Zr) InvModP(a *Zr) {
	(*FP256BN.BIG)(z).Invmodp((*FP256BN.BIG)(a))
}

func (z *Zr) PowMod(a *Zr, b *Zr) *Zr {
	return (*Zr)((*FP256BN.BIG)(z).Powmod((*FP256BN.BIG)(a), (*FP256BN.BIG)(b)))
}

func Sum(values []*Zr) *Zr {
	sum := NewZr()
	for i := 0; i < len(values); i++ {
		sum = ModAdd(sum, values[i], Order)
	}
	return sum
}

func ModNeg(a1, m *Zr) *Zr {
	a := NewZrCopy(a1)
	a.Mod(m)
	return m.Minus(a)
}

func ModAdd(a, b, m *Zr) *Zr {
	c := a.Plus(b)
	c.Mod(m)
	return c
}

func ModSub(a, b, m *Zr) *Zr {
	c := ModNeg(b, m)
	return ModAdd(a, c, m)
}

func ModMul(a1, b1, m *Zr) *Zr {
	return (*Zr)(FP256BN.Modmul((*FP256BN.BIG)(a1), (*FP256BN.BIG)(b1), (*FP256BN.BIG)(m)))
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
	if len(r) != MODBYTES {
		return errors.Errorf("failed to unmarshal an element in Zr")
	}
	*z = *((*Zr)(FP256BN.FromBytes(r)))
	return nil
}
