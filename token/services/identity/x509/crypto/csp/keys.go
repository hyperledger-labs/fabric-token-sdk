/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package csp

import (
	"crypto/ecdsa"
	"crypto/x509"
	"errors"
)

func derToPrivateKey(der []byte) (key interface{}, err error) {
	if key, err = x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}

	if key, err = x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key.(type) {
		case *ecdsa.PrivateKey:
			return
		default:
			return nil, errors.New("found unknown private key type in PKCS#8 wrapping")
		}
	}

	if key, err = x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("invalid key type. The DER must contain an ecdsa.PrivateKey")
}

func derToPublicKey(raw []byte) (pub interface{}, err error) {
	if len(raw) == 0 {
		return nil, errors.New("invalid DER. It must be different from nil")
	}

	key, err := x509.ParsePKIXPublicKey(raw)

	return key, err
}
