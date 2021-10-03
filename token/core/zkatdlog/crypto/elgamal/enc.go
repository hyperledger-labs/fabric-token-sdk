/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal

import (
	"github.com/pkg/errors"
	bn256 "github.ibm.com/fabric-research/mathlib"
)

type PublicKey struct {
	Gen   *bn256.G1
	H     *bn256.G1
	Curve *bn256.Curve
}

type Ciphertext struct {
	C1 *bn256.G1
	C2 *bn256.G1
}

type SecretKey struct {
	*PublicKey
	x *bn256.Zr
}

func NewSecretKey(sk *bn256.Zr, gen, pk *bn256.G1, c *bn256.Curve) *SecretKey {
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
func (pk *PublicKey) Encrypt(M *bn256.G1) (*Ciphertext, *bn256.Zr, error) {
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
func (sk *SecretKey) Decrypt(c *Ciphertext) *bn256.G1 {
	c.C2.Sub(c.C1.Mul(sk.x))
	return c.C2

}

// encrypt message in Zr using Elgamal encryption
func (pk *PublicKey) EncryptZr(m *bn256.Zr) (*Ciphertext, *bn256.Zr, error) {
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
