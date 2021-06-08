/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/math/gurvy/bn256"
	"github.com/pkg/errors"
)

type PublicKey struct {
	Gen *bn256.G1
	H   *bn256.G1
}

type Ciphertext struct {
	C1 *bn256.G1
	C2 *bn256.G1
}

type SecretKey struct {
	*PublicKey
	x *bn256.Zr
}

func NewSecretKey(sk *bn256.Zr, gen, pk *bn256.G1) *SecretKey {
	return &SecretKey{
		x: sk,
		PublicKey: &PublicKey{
			Gen: gen,
			H:   pk,
		},
	}
}

// encrypt using Elgamal encryption
func (pk *PublicKey) Encrypt(M *bn256.G1) (*Ciphertext, *bn256.Zr, error) {
	if pk.Gen == nil || pk.H == nil {
		return nil, nil, errors.Errorf("Provide a non-nil Elgamal public key")
	}
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compute Elgamal ciphertext")
	}
	r := bn256.RandModOrder(rand)
	return &Ciphertext{
		C1: pk.Gen.Mul(r),
		C2: pk.H.Mul(r).Add(M),
	}, r, nil
}

// Decrypt using Elgamal secret key
func (sk *SecretKey) Decrypt(c *Ciphertext) *bn256.G1 {
	return c.C2.Sub(c.C1.Mul(sk.x))

}

// encrypt message in Zr using Elgamal encryption
func (pk *PublicKey) EncryptZr(m *bn256.Zr) (*Ciphertext, *bn256.Zr, error) {
	rand, err := bn256.GetRand()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compute Elgamal ciphertext")
	}
	r := bn256.RandModOrder(rand)
	return &Ciphertext{
		C1: pk.Gen.Mul(r),
		C2: pk.H.Mul(r).Add(pk.Gen.Mul(m)),
	}, r, nil
}
