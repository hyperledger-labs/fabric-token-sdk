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

type G1 bn256.G1Affine

func NewG1() *G1 {
	return (*G1)(&bn256.G1Affine{})
}

func NewG1FromBytes(b []byte) (*G1, error) {
	v := &bn256.G1Affine{}
	_, err := v.SetBytes(b)
	if err != nil {
		return nil, err
	}
	return (*G1)(v), nil
}

func G1Gen() *G1 {
	_, _, g1, _ := bn256.Generators()
	return NewG1().Copy((*G1)(&g1))
}

func (g *G1) Copy(a *G1) *G1 {
	raw := (*bn256.G1Affine)(a).Bytes()
	// TODO: catch error?
	(*bn256.G1Affine)(g).SetBytes(clone(raw[:]))
	return g
}

func (g *G1) Equals(a *G1) bool {
	return (*bn256.G1Affine)(g).Equal((*bn256.G1Affine)(a))
}

func (g *G1) Bytes() []byte {
	r := (*bn256.G1Affine)(g).Bytes()
	return r[:]
}

func (g *G1) Mul(a *Zr) *G1 {
	return (*G1)((*bn256.G1Affine)(NewG1().Copy(g)).ScalarMultiplication((*bn256.G1Affine)(g), (*big.Int)(a)))
}

func (g *G1) Add(a *G1) *G1 {
	j := &bn256.G1Jac{}
	j.FromAffine((*bn256.G1Affine)(g))
	j.AddMixed((*bn256.G1Affine)(a))
	(*bn256.G1Affine)(g).FromJacobian(j)
	return g
}

func (g *G1) Sub(a *G1) *G1 {
	left := &bn256.G1Jac{}
	left.FromAffine((*bn256.G1Affine)(g))
	right := &bn256.G1Jac{}
	right.FromAffine((*bn256.G1Affine)(a))
	left.SubAssign(right)

	g = (*G1)((*bn256.G1Affine)(g).FromJacobian(left))
	return g
}

func (g *G1) String() string {
	return (*bn256.G1Affine)(g).String()
}

func HashToG1(message []byte) (*G1, error) {
	g, err := bn256.HashToCurveG1Svdw(message, nil)
	if err != nil {
		return nil, err
	}
	return (*G1)(&g), nil
}

func (g *G1) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.Bytes())
}

func (g *G1) UnmarshalJSON(raw []byte) error {
	var r []byte
	err := json.Unmarshal(raw, &r)
	if err != nil {
		return err
	}
	v := &bn256.G1Affine{}
	_, err = v.SetBytes(clone(r))
	if err != nil {
		return err
	}
	*g = *(*G1)(v)
	return err
}

func clone(a []byte) []byte {
	res := make([]byte, len(a))
	copy(res, a)
	return res
}
