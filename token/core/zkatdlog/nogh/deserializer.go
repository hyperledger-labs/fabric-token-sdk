/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric/idemix"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric/x509"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type VerifierDES interface {
	DeserializeVerifier(id view.Identity) (driver.Verifier, error)
}

type AuditDES interface {
	DeserializeAuditInfo(raw []byte) (driver.Matcher, error)
}

type deserializer struct {
	auditorDeserializer VerifierDES
	ownerDeserializer   VerifierDES
	issuerDeserializer  VerifierDES
	auditDeserializer   AuditDES
}

func NewDeserializer(pp *crypto.PublicParams) (*deserializer, error) {
	idemixDes, err := idemix.NewDeserializer(pp.IdemixPK)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params")
	}

	return &deserializer{
		auditorDeserializer: &x509.MSPIdentityDeserializer{},
		issuerDeserializer:  &x509.MSPIdentityDeserializer{},
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(idemixDes),
		auditDeserializer:   idemixDes,
	}, nil
}

func (d *deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.DeserializeVerifier(id)
}

func (d *deserializer) GetOwnerMatcher(raw []byte) (driver.Matcher, error) {
	return d.auditDeserializer.DeserializeAuditInfo(raw)
}
