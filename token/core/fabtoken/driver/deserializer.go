/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/deserializer"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
	htlc2 "github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/htlc"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/interop/pledge"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer() *Deserializer {
	m := deserializer.NewTypedVerifierDeserializerMultiplex(&x509.AuditMatcherDeserializer{})
	m.AddTypedVerifierDeserializer(msp.X509Identity, deserializer.NewTypedIdentityVerifierDeserializer(&x509.MSPIdentityDeserializer{}))
	m.AddTypedVerifierDeserializer(htlc2.ScriptType, htlc2.NewTypedIdentityDeserializer(m))
	m.AddTypedVerifierDeserializer(pledge.ScriptType, pledge.NewTypedIdentityDeserializer(m))

	return &Deserializer{
		Deserializer: common.NewDeserializer(
			msp.X509Identity,
			&x509.MSPIdentityDeserializer{}, // audit
			m,                               // owner
			&x509.MSPIdentityDeserializer{}, // issuer
			m,
			m,
		),
	}
}

type PublicParamsDeserializer struct{}

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*fabtoken.PublicParams, error) {
	return fabtoken.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer = deserializer.EIDRHDeserializer

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	d := deserializer.NewEIDRHDeserializer()
	d.AddDeserializer(msp.X509Identity, &x509.AuditInfoDeserializer{})
	d.AddDeserializer(htlc2.ScriptType, htlc2.NewAuditDeserializer(&x509.AuditInfoDeserializer{}))
	d.AddDeserializer(pledge.ScriptType, pledge.NewAuditDeserializer(&x509.AuditInfoDeserializer{}))
	return d
}
