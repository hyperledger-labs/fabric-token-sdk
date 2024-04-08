/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package msp

import (
	idemix "github.com/IBM/idemix/bccsp"
	"github.com/IBM/idemix/bccsp/keystore"
	idemix3 "github.com/IBM/idemix/bccsp/schemes/dlog/crypto"
	"github.com/IBM/idemix/bccsp/schemes/dlog/crypto/translator/amcl"
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

func NewKeyStore(curveID math.CurveID, backend keystore.KVS) (bccsp.KeyStore, error) {
	curve, tr, err := GetCurveAndTranslator(curveID)
	if err != nil {
		return nil, err
	}
	keyStore := &keystore.KVSStore{
		Curve:      curve,
		KVS:        backend,
		Translator: tr,
	}
	return keyStore, nil
}

// NewBCCSP returns an instance of the idemix BCCSP for the given curve and kvsStore
func NewBCCSP(keyStore bccsp.KeyStore, curveID math.CurveID, aries bool) (bccsp.BCCSP, error) {
	curve, tr, err := GetCurveAndTranslator(curveID)
	if err != nil {
		return nil, err
	}

	var cryptoProvider bccsp.BCCSP
	if aries {
		cryptoProvider, err = idemix.NewAries(keyStore, curve, tr, true)
	} else {
		cryptoProvider, err = idemix.New(keyStore, curve, tr, true)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}

	return cryptoProvider, nil
}

// NewBCCSPWithDummyKeyStore returns an instance of the idemix BCCSP for the given curve
func NewBCCSPWithDummyKeyStore(curveID math.CurveID, aries bool) (bccsp.BCCSP, error) {
	curve, tr, err := GetCurveAndTranslator(curveID)
	if err != nil {
		return nil, err
	}
	var cryptoProvider bccsp.BCCSP
	if aries {
		cryptoProvider, err = idemix.NewAries(&keystore.Dummy{}, curve, tr, true)
	} else {
		cryptoProvider, err = idemix.New(&keystore.Dummy{}, curve, tr, true)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}
	return cryptoProvider, nil
}

func GetCurveAndTranslator(curveID math.CurveID) (*math.Curve, idemix3.Translator, error) {
	curve := math.Curves[curveID]
	var tr idemix3.Translator
	switch curveID {
	case math.BN254:
		tr = &amcl.Gurvy{C: curve}
	case math.BLS12_377_GURVY:
		tr = &amcl.Gurvy{C: curve}
	case math.FP256BN_AMCL:
		tr = &amcl.Fp256bn{C: curve}
	case math.FP256BN_AMCL_MIRACL:
		tr = &amcl.Fp256bnMiracl{C: curve}
	case math.BLS12_381_BBS:
		tr = &amcl.Gurvy{C: curve}
	default:
		return nil, nil, errors.Errorf("unsupported curve ID: %d", curveID)
	}
	return curve, tr, nil
}
