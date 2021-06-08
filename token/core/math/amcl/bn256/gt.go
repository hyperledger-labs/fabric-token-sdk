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

type GT FP256BN.FP12

func (g *GT) IsUnity() bool {
	return (*FP256BN.FP12)(g).Isunity()
}

func (g *GT) Mul(a *GT) {
	(*FP256BN.FP12)(g).Mul((*FP256BN.FP12)(a))
}

func (g *GT) Inverse() {
	(*FP256BN.FP12)(g).Inverse()
}

func (g *GT) Bytes() []byte {
	b := make([]byte, 12*FP256BN.MODBYTES)
	(*FP256BN.FP12)(g).ToBytes(b)
	return b
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
	if len(r) != 12*MODBYTES {
		return errors.Errorf("failed to unmarshal an element in G2")
	}
	*g = *((*GT)(FP256BN.FP12_fromBytes(r)))
	return nil
}
