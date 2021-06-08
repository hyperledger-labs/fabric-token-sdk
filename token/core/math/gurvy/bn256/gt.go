/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"encoding/json"
	"github.com/consensys/gurvy/bn256"
)

type GT bn256.GT

func NewGT() *GT {
	return (*GT)(&bn256.GT{})
}

func (g *GT) IsUnity() bool {
	unity := &bn256.GT{}
	unity.SetOne()
	return (*bn256.GT)(g).Equal(unity)
}

func (g *GT) Mul(a *GT) {
	(*bn256.GT)(g).Mul((*bn256.GT)(a), (*bn256.GT)(g))
}

func (g *GT) Inverse() {
	(*bn256.GT)(g).Inverse((*bn256.GT)(g))
}

func (g *GT) Bytes() []byte {
	r := (*bn256.GT)(g).Bytes()
	return r[:]
}

func (g *GT) MarshalJSON() ([]byte, error) {
	return json.Marshal(g.Bytes())
}

func (g *GT) UnmarshalJSON(raw []byte) error {
	var r []byte
	err := json.Unmarshal(raw, &r)
	if err != nil {
		return err
	}
	(*bn256.GT)(g).SetBytes(r)
	return err
}
