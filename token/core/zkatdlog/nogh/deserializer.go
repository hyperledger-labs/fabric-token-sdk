/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package nogh

import (
	idemix2 "github.com/hyperledger-labs/fabric-smart-client/platform/fabric/core/generic/msp/idemix"
	driver2 "github.com/hyperledger-labs/fabric-smart-client/platform/view/driver"
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"
	"github.com/pkg/errors"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/zkatdlog/crypto"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type verifierProvider interface {
	GetVerifier(id view.Identity) (driver.Verifier, error)
}

type idemixProvider interface {
	DeserializeVerifier(raw []byte) (driver2.Verifier, error)
}

type deserializer struct {
	auditorDeserializer verifierProvider
	ownerDeserializer   verifierProvider
	issuerDeserializer  verifierProvider
}

func NewDeserializer(pp *crypto.PublicParams) (*deserializer, error) {
	idemixDes, err := idemix2.NewDeserializer(pp.IdemixPK)
	if err != nil {
		return nil, errors.Wrapf(err, "failed getting idemix deserializer for passed public params")
	}

	return &deserializer{
		auditorDeserializer: &fabric.MSPX509IdentityDeserializer{},
		issuerDeserializer:  identity.NewRawOwnerIdentityDeserializer(&fabric.MSPX509IdentityDeserializer{}),
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(&idemixDeserializer{provider: idemixDes}),
	}, nil
}

func (d *deserializer) GetOwnerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.ownerDeserializer.GetVerifier(id)
}

func (d *deserializer) GetIssuerVerifier(id view.Identity) (driver.Verifier, error) {
	return d.issuerDeserializer.GetVerifier(id)
}

func (d *deserializer) GetAuditorVerifier(id view.Identity) (driver.Verifier, error) {
	return d.auditorDeserializer.GetVerifier(id)
}

type idemixDeserializer struct {
	provider idemixProvider
}

func (i *idemixDeserializer) GetVerifier(id view.Identity) (driver.Verifier, error) {
	return i.provider.DeserializeVerifier(id)
}
