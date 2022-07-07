/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/
package elgamal

import (
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// SecretKey is Elgamal secret key x
type SecretKey struct {
	*PublicKey
	x *math.Zr
}

// PublicKey is Elgamal public key (g, h= g^x)
type PublicKey struct {
	Gen   *math.G1
	H     *math.G1
	Curve *math.Curve
}

// Ciphertext is an Elgamal ciphertext C = (g^r,  Mh^r)
type Ciphertext struct {
	C1 *math.G1
	C2 *math.G1
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

// Encrypt returns an Elgamal ciphertext of elliptic curve point M
// and the randomness used to compute it
func (pk *PublicKey) Encrypt(M *math.G1) (*Ciphertext, *math.Zr, error) {
	if pk.Gen == nil || pk.H == nil {
		return nil, nil, errors.Errorf("Provide a non-nil Elgamal public key")
	}
	if M == nil {
		return nil, nil, errors.Errorf("Provide a non-nil message")
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

// Decrypt returns a elliptic curve point M such that C = (g^r, Mh^r)
func (sk *SecretKey) Decrypt(c *Ciphertext) (*math.G1, error) {
	if sk.x == nil {
		return nil, errors.Errorf("Provide a non-nil secret key")
	}
	c.C2.Sub(c.C1.Mul(sk.x))
	return c.C2, nil

}

// EncryptZr encrypts a message m  in Zr using Elgamal encryption
// EncryptZr returns C = (g^r, g^mh^r) and randomness r
func (pk *PublicKey) EncryptZr(m *math.Zr) (*Ciphertext, *math.Zr, error) {
	// safety checks
	if pk.Gen == nil || pk.H == nil {
		return nil, nil, errors.Errorf("Provide a non-nil Elgamal public key")
	}
	if m == nil {
		return nil, nil, errors.Errorf("Provide a non-nil message")
	}
	if pk.Curve == nil {
		return nil, nil, errors.New("failed to compute Elgamal ciphertext: please initialize curve")
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
	c.C2.Add(pk.Gen.Mul(m))
	return c, r, nil
}
