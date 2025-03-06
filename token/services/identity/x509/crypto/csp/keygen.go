/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"

	"github.com/hyperledger/fabric/bccsp"
	"github.com/pkg/errors"
)

type ecdsaKeyGenerator struct {
	curve elliptic.Curve
}

func (kg *ecdsaKeyGenerator) KeyGen(opts bccsp.KeyGenOpts) (bccsp.Key, error) {
	privKey, err := ecdsa.GenerateKey(kg.curve, rand.Reader)
	if err != nil {
		return nil, errors.Wrapf(err, "failed generating ECDSA key for [%v]", kg.curve)
	}

	return &ecdsaPrivateKey{privKey}, nil
}
