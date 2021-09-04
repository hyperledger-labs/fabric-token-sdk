/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package fabtoken

import (
	"github.com/hyperledger-labs/fabric-smart-client/platform/view/view"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/identity/fabric"
	"github.com/hyperledger-labs/fabric-token-sdk/token/driver"
)

type verifierProvider interface {
	GetVerifier(id view.Identity) (driver.Verifier, error)
}

type deserializer struct {
	auditorDeserializer verifierProvider
	ownerDeserializer   verifierProvider
	issuerDeserializer  verifierProvider
}

func NewDeserializer() *deserializer {
	return &deserializer{
		auditorDeserializer: &fabric.MSPX509IdentityDeserializer{},
		issuerDeserializer:  &fabric.MSPX509IdentityDeserializer{},
		ownerDeserializer:   identity.NewRawOwnerIdentityDeserializer(&fabric.MSPX509IdentityDeserializer{}),
	}
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
