/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package bn256

import (
	"crypto/rand"

	"github.com/hyperledger/fabric-amcl/amcl"
	"github.com/hyperledger/fabric-amcl/amcl/FP256BN"
	"github.com/pkg/errors"
)

var order = FP256BN.NewBIGints(FP256BN.CURVE_Order)

type Rand amcl.RAND

// RandModOrder returns a random element in 0, ..., GroupOrder-1
func RandModOrder(rng *Rand) *Zr {
	// Take random element in Zq
	return (*Zr)(FP256BN.Randomnum(order, (*amcl.RAND)(rng)))
}

// GetRand returns a new *amcl.RAND with a fresh seed
func GetRand() (*Rand, error) {
	seedLength := 32
	b := make([]byte, seedLength)
	_, err := rand.Read(b)
	if err != nil {
		return nil, errors.Wrap(err, "error getting blindingFactors for seed")
	}
	rng := amcl.NewRAND()
	rng.Clean()
	rng.Seed(seedLength, b)
	return (*Rand)(rng), nil
}
