/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"encoding/json"
	"math/big"

	"github.com/consensys/gurvy/bn256"
)

type G2 bn256.G2Affine

func NewG2() *G2 {
	return (*G2)(&bn256.G2Affine{})
}

func NewG2FromBytes(b []byte) (*G2, error) {
	v := &bn256.G2Affine{}
	_, err := v.SetBytes(b)
	if err != nil {
		return nil, err
	}
	return (*G2)(v), nil
}

func G2Gen() *G2 {
	_, _, _, g2 := bn256.Generators()
	return NewG2().Copy((*G2)(&g2))
}

func (g *G2) Copy(a *G2) *G2 {
	raw := (*bn256.G2Affine)(a).Bytes()
	// TODO: catch error?
	(*bn256.G2Affine)(g).SetBytes(raw[:])
	return g
}

func (g *G2) Mul(a *Zr) *G2 {
	h := NewG2()
	(*bn256.G2Affine)(h).ScalarMultiplication((*bn256.G2Affine)(g), (*big.Int)(a))
	return h
}

func (g *G2) Add(a *G2) *G2 {
	j := &bn256.G2Jac{}
	j.FromAffine((*bn256.G2Affine)(g))
	j.AddMixed((*bn256.G2Affine)(a))
	(*bn256.G2Affine)(g).FromJacobian(j)
	return g
}

func (g *G2) Sub(a *G2) *G2 {
	left := &bn256.G2Jac{}
	left.FromAffine((*bn256.G2Affine)(g))
	right := &bn256.G2Jac{}
	right.FromAffine((*bn256.G2Affine)(a))
	left.SubAssign(right)

	g = (*G2)((*bn256.G2Affine)(g).FromJacobian(left))
	return g
}

func (g *G2) Bytes() []byte {
	r := (*bn256.G2Affine)(g).Bytes()
	return r[:]
}

func (g *G2) String() string {
	return (*bn256.G2Affine)(g).String()
}

func (g *G2) Equals(h *G2) bool {
	return (*bn256.G2Affine)(g).Equal((*bn256.G2Affine)(h))
}

func (g *G2) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.Bytes())
}

func (g *G2) UnmarshalJSON(raw []byte) error {
	var r []byte
	err := json.Unmarshal(raw, &r)
	if err != nil {
		return err
	}
	v := &bn256.G2Affine{}
	_, err = v.SetBytes(r)
	if err != nil {
		return err
	}
	*g = *(*G2)(v)

	return err
}
