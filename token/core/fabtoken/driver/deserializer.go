/*
Copyright IBM Corp. All Rights Reserved.

SPDX-License-Identifier: Apache-2.0
*/

package driver

import (
	"github.com/hyperledger-labs/fabric-token-sdk/token/core/common"

	"github.com/hyperledger-labs/fabric-token-sdk/token/core/fabtoken"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp"
	"github.com/hyperledger-labs/fabric-token-sdk/token/services/identity/msp/x509"
)

// Deserializer deserializes verifiers associated with issuers, owners, and auditors
type Deserializer struct {
	*common.Deserializer
}

// NewDeserializer returns a deserializer
func NewDeserializer() *Deserializer {
	return &Deserializer{
		Deserializer: common.NewDeserializer(
			msp.X509Identity,
			&x509.MSPIdentityDeserializer{}, // audit
			&x509.MSPIdentityDeserializer{}, // owner
			&x509.MSPIdentityDeserializer{}, // issuer
			&x509.AuditMatcherDeserializer{},
		),
	}
}

type PublicParamsDeserializer struct{}

func (p *PublicParamsDeserializer) DeserializePublicParams(raw []byte, label string) (*fabtoken.PublicParams, error) {
	return fabtoken.NewPublicParamsFromBytes(raw, label)
}

// EIDRHDeserializer returns enrollment ID and revocation handle behind the owners of token
type EIDRHDeserializer = common.EIDRHDeserializer[*x509.AuditInfo]

// NewEIDRHDeserializer returns an enrollmentService
func NewEIDRHDeserializer() *EIDRHDeserializer {
	return common.NewEIDRHDeserializer[*x509.AuditInfo](&x509.AuditInfoDeserializer{})
}
