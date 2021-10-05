/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal

import (
	"github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

type PublicKey struct {
	Gen   *math.G1
	H     *math.G1
	Curve *math.Curve
}

type Ciphertext struct {
	C1 *math.G1
	C2 *math.G1
}

type SecretKey struct {
	*PublicKey
	x *math.Zr
}

func NewSecretKey(sk *math.Zr, gen, pk *math.G1, c *math.Curve) *SecretKey {
	return &SecretKey{
		x: sk,
		PublicKey: &PublicKey{
			Gen:   gen,
			H:     pk,
			Curve: c,
		},
	}
}

// encrypt using Elgamal encryption
func (pk *PublicKey) Encrypt(M *math.G1) (*Ciphertext, *math.Zr, error) {
	if pk.Gen == nil || pk.H == nil {
		return nil, nil, errors.Errorf("Provide a non-nil Elgamal public key")
	}
	rand, err := pk.Curve.Rand()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compute Elgamal ciphertext")
	}
	r := pk.Curve.NewRandomZr(rand)
	c := &Ciphertext{
		C1: pk.Gen.Mul(r),
	}
	c.C2 = pk.H.Mul(r)
	c.C2.Add(M)
	return c, r, nil
}

// Decrypt using Elgamal secret key
func (sk *SecretKey) Decrypt(c *Ciphertext) *math.G1 {
	c.C2.Sub(c.C1.Mul(sk.x))
	return c.C2

}

// encrypt message in Zr using Elgamal encryption
func (pk *PublicKey) EncryptZr(m *math.Zr) (*Ciphertext, *math.Zr, error) {
	rand, err := pk.Curve.Rand()
	if err != nil {
		return nil, nil, errors.Wrapf(err, "failed to compute Elgamal ciphertext")
	}
	r := pk.Curve.NewRandomZr(rand)
	c := &Ciphertext{
		C1: pk.Gen.Mul(r),
	}
	c.C2 = pk.H.Mul(r)
	c.C2.Add(pk.Gen.Mul(m))
	return c, r, nil
}
