/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	idemix2 "github.com/IBM/idemix/bccsp"
	"github.com/IBM/idemix/bccsp/keystore"
	bccsp "github.com/IBM/idemix/bccsp/types"
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// NewKVSBCCSP returns a new BCCSP for the passed curve, if the curve is BLS12_381_BBS, it returns the BCCSP implementation
// based on aries.
func NewKVSBCCSP(kvsStore keystore.KVS, curveID math.CurveID) (bccsp.BCCSP, error) {
	if curveID == math.BLS12_381_BBS {
		logger.Debugf("new aries KVS-based BCCSP")
		return NewKSVBCCSP(kvsStore, curveID, true)
	}
	logger.Debugf("new dlog KVS-based BCCSP")
	return NewKSVBCCSP(kvsStore, curveID, false)
}

// NewAriesBCCSP returns an instance of the idemix BCCSP for the given curve based on aries
func NewAriesBCCSP() (bccsp.BCCSP, error) {
	logger.Infof("new aries no-KeyStore BCCSP")
	curve, tr, err := GetCurveAndTranslator(math.BLS12_381_BBS)
	if err != nil {
		return nil, err
	}
	cryptoProvider, err := idemix2.NewAries(&keystore.Dummy{}, curve, tr, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}
	return cryptoProvider, nil
}
