/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"github.com/consensys/gurvy/bn256/fr"
)

var Order *Zr

func init() {
	Order = (*Zr)(fr.Modulus())
}
