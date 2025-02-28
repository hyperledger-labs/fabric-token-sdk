/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	v1 "github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken/v1"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/interop/htlc"
	x510 "github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer() *Deserializer {
	ownerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	ownerDeserializer.AddTypedVerifierDeserializer(x510.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&x510.IdentityDeserializer{}, &x510.AuditMatcherDeserializer{}))
	ownerDeserializer.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc.NewTypedIdentityDeserializer(ownerDeserializer))

	auditorIssuerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	auditorIssuerDeserializer.AddTypedVerifierDeserializer(x510.IdentityType, deserializer.NewTypedIdentityVerifierDeserializer(&x510.IdentityDeserializer{}, &x510.AuditMatcherDeserializer{}))

	return &Deserializer{
		Deserializer: common.NewDeserializer(
			x510.IdentityType,
			auditorIssuerDeserializer,
			ownerDeserializer, // owner
			auditorIssuerDeserializer,
			ownerDeserializer,
			ownerDeserializer,
		),
	}
}

type PublicParamsDeserializer struct{}

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*v1.PublicParams, error) {
	return v1.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer = deserializer.EIDRHDeserializer

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	d := deserializer.NewEIDRHDeserializer()
	d.AddDeserializer(x510.IdentityType, &x510.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc.NewAuditDeserializer(&x510.AuditInfoDeserializer{}))
	return d
}
