/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package idemix

import (
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

func (d *Deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.Deserializer.DeserializeAuditInfo(raw)
}

// NewDeserializer returns a new deserializer for the idemix ExpectEidNymRhNym verification strategy
func NewDeserializer(ipk []byte, curveID math.CurveID) (*Deserializer, error) {
	cryptoProvider, err := idemix.NewBCCSP(curveID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
	}
	return NewDeserializerWithProvider(ipk, bccsp.ExpectEidNymRhNym, nil, cryptoProvider)
}

func NewDeserializerAries(ipk []byte, curveID math.CurveID) (*Deserializer, error) {
	cryptoProvider, err := idemix.NewBCCSP(curveID)
	if err != nil {
		return nil, errors.WithMessagef(err, "failed to instantiate crypto provider for curve [%d]", curveID)
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
