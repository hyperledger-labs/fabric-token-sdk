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

type G2 FP256BN.ECP2

func NewG2() *G2 {
	return (*G2)(FP256BN.NewECP2())
}

func NewG2FromBytes(b []byte) *G2 {
	return (*G2)(FP256BN.ECP2_fromBytes(b))
}

func G2Gen() *G2 {
	return (*G2)(FP256BN.ECP2_generator())
}

func (g *G2) Mul(z *Zr) *G2 {
	return (*G2)((*FP256BN.ECP2)(g).Mul((*FP256BN.BIG)(z)))
}

func (g *G2) Add(a *G2) {
	(*FP256BN.ECP2)(g).Add((*FP256BN.ECP2)(a))
}

func (g *G2) Sub(a *G2) {
	(*FP256BN.ECP2)(g).Sub((*FP256BN.ECP2)(a))
}

func (g *G2) Bytes() []byte {
	b := make([]byte, 4*FP256BN.MODBYTES)
	(*FP256BN.ECP2)(g).ToBytes(b)
	return b
}

func (g *G2) String() string {
	return (*FP256BN.ECP2)(g).ToString()
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
	if len(r) != 4*MODBYTES {
		return errors.Errorf("failed to unmarshal an element in G2")
	}
	*g = *((*G2)(FP256BN.ECP2_fromBytes(r)))
	return nil
}
