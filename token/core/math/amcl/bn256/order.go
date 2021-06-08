/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import "github.com/hyperledger/fabric-amcl/amcl/FP256BN"

var Order = (*Zr)(FP256BN.NewBIGints(FP256BN.CURVE_Order))

const MODBYTES = 32
const EFS = MODBYTES
