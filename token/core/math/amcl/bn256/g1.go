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

type G1 FP256BN.ECP

func NewG1() *G1 {
	return (*G1)(FP256BN.NewECP())
}

func NewG1FromBytes(b []byte) *G1 {
	return (*G1)(FP256BN.ECP_fromBytes(b))
}

func G1Gen() *G1 {
	return (*G1)(FP256BN.ECP_generator())
}

func (g *G1) Copy(a *G1) *G1 {
	(*FP256BN.ECP)(g).Copy((*FP256BN.ECP)(a))
	return g
}

func (g *G1) Equals(a *G1) bool {
	return (*FP256BN.ECP)(g).Equals((*FP256BN.ECP)(a))
}

func (g *G1) Bytes() []byte {
	b := make([]byte, 2*FP256BN.EFS+1)
	(*FP256BN.ECP)(g).ToBytes(b, false)
	return b
}

func (g *G1) Mul(a *Zr) *G1 {
	return (*G1)((*FP256BN.ECP)(g).Mul((*FP256BN.BIG)(a)))
}

func (g *G1) Add(a *G1) *G1 {
	(*FP256BN.ECP)(g).Add((*FP256BN.ECP)(a))
	return g
}

func (g *G1) Sub(a *G1) *G1 {
	(*FP256BN.ECP)(g).Sub((*FP256BN.ECP)(a))
	return g
}

func (g *G1) String() string {
	return (*FP256BN.ECP)(g).ToString()
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
	if len(r) != 2*EFS+1 {
		return errors.Errorf("failed to unmarshal an element in G1")
	}
	*g = *((*G1)(FP256BN.ECP_fromBytes(r)))
	return nil
}

func HashToG1(message string) *G1 {
	return (*G1)(FP256BN.Bls_hash(message))
}
