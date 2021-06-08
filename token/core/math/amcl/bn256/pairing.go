/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import "github.com/hyperledger/fabric-amcl/amcl/FP256BN"

func Pairing(P1 *G2, Q1 *G1, R1 *G2, S1 *G1) *GT {
	return (*GT)(FP256BN.Ate2(
		(*FP256BN.ECP2)(P1),
		(*FP256BN.ECP)(Q1),
		(*FP256BN.ECP2)(R1),
		(*FP256BN.ECP)(S1),
	))
}

func FinalExp(m *GT) *GT {
	return (*GT)(FP256BN.Fexp((*FP256BN.FP12)(m)))
}
