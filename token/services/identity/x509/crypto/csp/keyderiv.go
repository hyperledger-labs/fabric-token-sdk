/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"crypto/ecdsa"
	"math/big"

	"github.com/hyperledger-labs/fabric-smart-client/pkg/utils/errors"
	"github.com/hyperledger/fabric-lib-go/bccsp"
)

type ecdsaPublicKeyKeyDeriver struct{}

func (kd *ecdsaPublicKeyKeyDeriver) KeyDeriv(key bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
	// Validate opts
	if opts == nil {
		return nil, errors.New("invalid opts parameter. It must not be nil")
	}

	ecdsaK := key.(*ecdsaPublicKey)

	// Re-randomized an ECDSA private key
	reRandOpts, ok := opts.(*bccsp.ECDSAReRandKeyOpts)
	if !ok {
		return nil, errors.Errorf("unsupported 'KeyDerivOpts' provided [%v]", opts)
	}

	tempSK := &ecdsa.PublicKey{
		Curve: ecdsaK.pubKey.Curve,
		X:     new(big.Int),
		Y:     new(big.Int),
	}

	k := new(big.Int).SetBytes(reRandOpts.ExpansionValue())
	one := new(big.Int).SetInt64(1)
	n := new(big.Int).Sub(ecdsaK.pubKey.Params().N, one)
	k.Mod(k, n)
	k.Add(k, one)

	// Compute temporary public key
	tempX, tempY := ecdsaK.pubKey.ScalarBaseMult(k.Bytes())
	tempSK.X, tempSK.Y = tempSK.Add(
		ecdsaK.pubKey.X, ecdsaK.pubKey.Y,
		tempX, tempY,
	)

	// Verify temporary public key is a valid point on the reference curve
	isOn := tempSK.Curve.IsOnCurve(tempSK.X, tempSK.Y)
	if !isOn {
		return nil, errors.New("failed temporary public key IsOnCurve check")
	}

	return &ecdsaPublicKey{tempSK}, nil
}

type ecdsaPrivateKeyKeyDeriver struct{}

func (kd *ecdsaPrivateKeyKeyDeriver) KeyDeriv(key bccsp.Key, opts bccsp.KeyDerivOpts) (bccsp.Key, error) {
	// Validate opts
	if opts == nil {
		return nil, errors.New("invalid opts parameter, it must not be nil")
	}

	ecdsaK := key.(*ecdsaPrivateKey)

	// Re-randomized an ECDSA private key
	reRandOpts, ok := opts.(*bccsp.ECDSAReRandKeyOpts)
	if !ok {
		return nil, errors.Errorf("unsupported 'KeyDerivOpts' provided [%v]", opts)
	}

	tempSK := &ecdsa.PrivateKey{
		PublicKey: ecdsa.PublicKey{
			Curve: ecdsaK.privKey.Curve,
			X:     new(big.Int),
			Y:     new(big.Int),
		},
		D: new(big.Int),
	}

	k := new(big.Int).SetBytes(reRandOpts.ExpansionValue())
	one := new(big.Int).SetInt64(1)
	n := new(big.Int).Sub(ecdsaK.privKey.Params().N, one)
	k.Mod(k, n)
	k.Add(k, one)

	tempSK.D.Add(ecdsaK.privKey.D, k)
	tempSK.D.Mod(tempSK.D, ecdsaK.privKey.PublicKey.Params().N)

	// Compute temporary public key
	tempX, tempY := ecdsaK.privKey.PublicKey.ScalarBaseMult(k.Bytes())
	tempSK.PublicKey.X, tempSK.PublicKey.Y =
		tempSK.PublicKey.Add(
			ecdsaK.privKey.PublicKey.X, ecdsaK.privKey.PublicKey.Y,
			tempX, tempY,
		)

	// Verify temporary public key is a valid point on the reference curve
	isOn := tempSK.Curve.IsOnCurve(tempSK.PublicKey.X, tempSK.PublicKey.Y)
	if !isOn {
		return nil, errors.New("failed temporary public key IsOnCurve check")
	}

	return &ecdsaPrivateKey{tempSK}, nil
}
