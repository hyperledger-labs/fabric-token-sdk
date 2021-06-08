/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"crypto/rand"
	"math/big"

	"github.com/consensys/gurvy/bn256/fr"
)

var order = fr.Modulus()

type Rand = func([]byte) (int, error)

// RandModOrder returns a random element in 0, ..., GroupOrder-1
func RandModOrder(rng Rand) *Zr {
	res := new(big.Int)
	v := &fr.Element{}
	v.SetRandom()

	return (*Zr)(v.ToBigIntRegular(res))
}

func GetRand() (Rand, error) {
	return rand.Read, nil
}
