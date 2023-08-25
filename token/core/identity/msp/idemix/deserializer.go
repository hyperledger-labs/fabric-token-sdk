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
	"github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
	"github.com/pkg/errors"
)

type Deserializer struct {
	*idemix.Deserializer
}

func (d *Deserializer) DeserializeVerifier(id view.Identity) (driver.Verifier, error) {
	return d.Deserializer.DeserializeVerifier(id)
}

func (d *Deserializer) DeserializeVerifierAgainstNymEID(id view.Identity, nymEID []byte) (driver.Verifier, error) {
	return d.Deserializer.DeserializeVerifierAgainstNymEID(id, nymEID)
}

func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.Deserializer.DeserializeAuditInfo(raw)
}

// NewDeserializer returns a new deserializer for the idemix ExpectEidNymRhNym verification strategy
func NewDeserializer(ipk []byte, curveID math.CurveID) (*Deserializer, error) {
	if curveID == math.BLS12_381_BBS {
		return NewDeserializerAries(ipk)
	}
	logger.Infof("new deserialized for dlog idemix")
	cryptoProvider, err := idemix.NewBCCSP(curveID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializerWithProvider(ipk, bccsp.ExpectEidNymRhNym, nil, cryptoProvider)
}

func NewDeserializerAries(ipk []byte) (*Deserializer, error) {
	cryptoProvider, err := NewAriesBCCSP()
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate aries crypto provider")
	}
	return NewDeserializerWithProvider(ipk, bccsp.ExpectEidNymRhNym, nil, cryptoProvider)
}

// NewDeserializerWithProvider returns a new serialized for the passed arguments
func NewDeserializerWithProvider(
	ipk []byte,
	verType bccsp.VerificationType,
	nymEID []byte,
	cryptoProvider bccsp.BCCSP,
) (*Deserializer, error) {
	d, err := idemix.NewDeserializerWithBCCSP(ipk, verType, nymEID, cryptoProvider)
	if err != nil {
		return nil, err
	}
	return &Deserializer{Deserializer: d}, nil
}

// NewKVSBCCSP returns a new BCCSP for the passed curve, if the curve is BLS12_381_BBS, it returns the BCCSP implementation
// based on aries.
func NewKVSBCCSP(kvsStore keystore.KVS, curveID math.CurveID) (bccsp.BCCSP, error) {
	if curveID == math.BLS12_381_BBS {
		logger.Debugf("new aries KVS-based BCCSP")
		return idemix.NewKSVBCCSP(kvsStore, curveID, true)
	}
	logger.Debugf("new dlog KVS-based BCCSP")
	return idemix.NewKSVBCCSP(kvsStore, curveID, false)
}

// NewAriesBCCSP returns an instance of the idemix BCCSP for the given curve based on aries
func NewAriesBCCSP() (bccsp.BCCSP, error) {
	logger.Infof("new aries no-KeyStore BCCSP")
	curve, tr, err := idemix.GetCurveAndTranslator(math.BLS12_381_BBS)
	if err != nil {
		return nil, err
	}
	cryptoProvider, err := idemix2.NewAries(&keystore.Dummy{}, curve, tr, true)
	if err != nil {
		return nil, errors.Wrap(err, "failed getting crypto provider")
	}
	return cryptoProvider, nil
}
