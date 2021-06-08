/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"crypto/sha256"
)

func HashModOrder(data []byte) *Zr {
	digest := sha256.Sum256(data)
	digestBig := NewZrFromBytes(digest[:])
	digestBig.Mod(Order)
	return digestBig
}
