/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
	idemix "github.com/IBM/idemix/bccsp"
	"github.com/IBM/idemix/bccsp/keystore"
	bccsp "github.com/IBM/idemix/bccsp/schemes"
	idemix2 "github.com/IBM/idemix/bccsp/schemes/dlog/crypto"
	"github.com/IBM/idemix/bccsp/schemes/dlog/crypto/translator/amcl"
	math "github.com/IBM/mathlib"
	"github.com/pkg/errors"
)

// NewBCCSP returns an instance of the idemix BCCSP for the given curve
func NewBCCSP(curveID math.CurveID) (bccsp.BCCSP, error) {
	curve := math.Curves[curveID]
	var tr idemix2.Translator
	switch curveID {
	case math.BN254:
		tr = &amcl.Gurvy{C: curve}
	case math.BLS12_377_GURVY:
		tr = &amcl.Gurvy{C: curve}
	case math.FP256BN_AMCL:
		tr = &amcl.Fp256bn{C: curve}
	case math.FP256BN_AMCL_MIRACL:
		tr = &amcl.Fp256bnMiracl{C: curve}
	default:
		return nil, errors.Errorf("unsupported curve ID: %d", curveID)
	}

	cryptoProvider, err := idemix.New(&keystore.Dummy{}, curve, tr, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}
	return cryptoProvider, nil
}
