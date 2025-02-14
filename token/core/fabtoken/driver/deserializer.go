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
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer() *Deserializer {
	ownerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	ownerDeserializer.AddTypedVerifierDeserializer(msp.X509Identity, deserializer.NewTypedIdentityVerifierDeserializer(&x509.MSPIdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))
	ownerDeserializer.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc.NewTypedIdentityDeserializer(ownerDeserializer))

	auditorIssuerDeserializer := deserializer.NewTypedVerifierDeserializerMultiplex()
	auditorIssuerDeserializer.AddTypedVerifierDeserializer(msp.X509Identity, deserializer.NewTypedIdentityVerifierDeserializer(&x509.MSPIdentityDeserializer{}, &x509.AuditMatcherDeserializer{}))

	return &Deserializer{
		Deserializer: common.NewDeserializer(
			msp.X509Identity,
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
	d.AddDeserializer(msp.X509Identity, &x509.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc.NewAuditDeserializer(&x509.AuditInfoDeserializer{}))
	return d
}
