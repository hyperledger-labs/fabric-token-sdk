/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"github.com/consensys/gurvy/bn256"
)

func Pairing(P1 *G2, Q1 *G1, R1 *G2, S1 *G1) *GT {
	t, err := bn256.MillerLoop([]bn256.G1Affine{(bn256.G1Affine)(*Q1), (bn256.G1Affine)(*S1)}, []bn256.G2Affine{(bn256.G2Affine)(*P1), (bn256.G2Affine)(*R1)})
	if err != nil {
		panic("failed to compute pairing")
	}
	return (*GT)(&t)
}

func FinalExp(m *GT) *GT {
	t := bn256.FinalExponentiation((*bn256.GT)(m))
	return (*GT)(&t)
}
