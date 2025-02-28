/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package ecdsa

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"

	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity"
	x510 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	"github.com/hyperledger/fabric/bccsp/utils"
	"github.com/pkg/errors"
)

var (
	// curveHalfOrders contains the precomputed curve group orders halved.
	// It is used to ensure that signature' S value is lower or equal to the
	// curve group order halved. We accept only low-S signatures.
	// They are precomputed for efficiency reasons.
	curveHalfOrders = map[elliptic.Curve]*big.Int{
		elliptic.P224(): new(big.Int).Rsh(elliptic.P224().Params().N, 1),
		elliptic.P256(): new(big.Int).Rsh(elliptic.P256().Params().N, 1),
		elliptic.P384(): new(big.Int).Rsh(elliptic.P384().Params().N, 1),
		elliptic.P521(): new(big.Int).Rsh(elliptic.P521().Params().N, 1),
	}
)

type Signature struct {
	R, S *big.Int
}

type Signer struct {
	*Verifier
	SK *ecdsa.PrivateKey
}

func (d *Signer) Sign(message []byte) ([]byte, error) {
	dgst := sha256.Sum256(message)

	r, s, err := ecdsa.Sign(rand.Reader, d.SK, dgst[:])
	if err != nil {
		return nil, err
	}

	s, _, err = ToLowS(&d.SK.PublicKey, s)
	if err != nil {
		return nil, err
	}

	return utils.MarshalECDSASignature(r, s)
}

func (d *Signer) Serialize() ([]byte, error) {
	return d.Verifier.Serialize()
}

type Verifier struct {
	PK *ecdsa.PublicKey
}

func NewECDSASigner() (*Signer, error) {
	// Create ephemeral key and store it in the context
	sk, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, err
	}
	return &Signer{SK: sk, Verifier: &Verifier{PK: &sk.PublicKey}}, nil
}

func (v *Verifier) Verify(message, sigma []byte) error {
	signature := &Signature{}
	_, err := asn1.Unmarshal(sigma, signature)
	if err != nil {
		return err
	}

	hash := sha256.New()
	n, err := hash.Write(message)
	if n != len(message) {
		return errors.Errorf("hash failure")
	}
	if err != nil {
		return err
	}
	digest := hash.Sum(nil)

	lowS, err := IsLowS(v.PK, signature.S)
	if err != nil {
		return err
	}
	if !lowS {
		return errors.New("signature is not in lowS")
	}

	valid := ecdsa.Verify(v.PK, digest, signature.R, signature.S)
	if !valid {
		return errors.Errorf("signature not valid")
	}

	return nil
}

func (v *Verifier) Serialize() ([]byte, error) {
	pkRaw, err := PemEncodeKey(v.PK)
	if err != nil {
		return nil, errors.Wrap(err, "failed marshalling public key")
	}

	wrap, err := identity.WrapWithType(x510.IdentityType, pkRaw)
	if err != nil {
		return nil, errors.Wrap(err, "failed wrapping identity")
	}

	return wrap, nil
}

// PemEncodeKey takes a Go key and converts it to bytes
func PemEncodeKey(key interface{}) ([]byte, error) {
	var encoded []byte
	var err error
	var keyType string

	switch key.(type) {
	case *ecdsa.PrivateKey, *rsa.PrivateKey:
		keyType = "PRIVATE"
		encoded, err = x509.MarshalPKCS8PrivateKey(key)
	case *ecdsa.PublicKey, *rsa.PublicKey:
		keyType = "PUBLIC"
		encoded, err = x509.MarshalPKIXPublicKey(key)
	default:
		err = errors.Errorf("Programming error, unexpected key type %T", key)
	}
	if err != nil {
		return nil, err
	}

	return pem.EncodeToMemory(&pem.Block{Type: keyType + " KEY", Bytes: encoded}), nil
}

// IsLowS checks that s is a low-S
func IsLowS(k *ecdsa.PublicKey, s *big.Int) (bool, error) {
	halfOrder, ok := curveHalfOrders[k.Curve]
	if !ok {
		return false, fmt.Errorf("curve not recognized [%s]", k.Curve)
	}

	return s.Cmp(halfOrder) != 1, nil

}

func ToLowS(k *ecdsa.PublicKey, s *big.Int) (*big.Int, bool, error) {
	lowS, err := IsLowS(k, s)
	if err != nil {
		return nil, false, err
	}

	if !lowS {
		// Set s to N - s that will be then in the lower part of signature space
		// less or equal to half order
		s.Sub(k.Params().N, s)

		return s, true, nil
	}

	return s, false, nil
}
